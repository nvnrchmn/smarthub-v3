package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// ErrNotFound sudah dipakai di paket ini? Tidak — sentinel lokal untuk billing.
var ErrBillingNotFound = errors.New("data tagihan tidak ditemukan")

// ---- Pengaturan penagihan -------------------------------------------------

func (s *Store) BillingSettings(ctx context.Context, tenantID string) (int, int, error) {
	day, due := 1, 14
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `select billing_day, due_days from tenant_billing_settings where tenant_id = $1`, tenantID)
		if err := row.Scan(&day, &due); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// belum diatur: pakai default PRD (terbit tanggal 1, jatuh tempo 14 hari)
				_, insErr := tx.Exec(ctx, `insert into tenant_billing_settings (tenant_id, billing_day, due_days)
					values ($1, 1, 14) on conflict (tenant_id) do nothing`, tenantID)
				return insErr
			}
			return err
		}
		return nil
	})
	return day, due, err
}

func (s *Store) SetBillingSettings(ctx context.Context, tenantID string, day, dueDays int) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `insert into tenant_billing_settings (tenant_id, billing_day, due_days, updated_at)
			values ($1,$2,$3, now())
			on conflict (tenant_id) do update set billing_day = excluded.billing_day,
			  due_days = excluded.due_days, updated_at = now()`, tenantID, day, dueDays)
		return err
	})
}

// ---- Master iuran -------------------------------------------------------

func (s *Store) FeeItems(ctx context.Context, tenantID string) ([]domain.FeeItem, error) {
	out := []domain.FeeItem{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select id, code, label, amount::float8, applies_to, is_active
			from fee_items order by is_active desc, label`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var f domain.FeeItem
			if err := rows.Scan(&f.ID, &f.Code, &f.Label, &f.Amount, &f.AppliesTo, &f.IsActive); err != nil {
				return err
			}
			out = append(out, f)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) UpsertFeeItem(ctx context.Context, tenantID, id, code, label string, amount float64, appliesTo string, active bool) (string, error) {
	var out string
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if id == "" {
			if err := tx.QueryRow(ctx, `insert into fee_items (tenant_id, code, label, amount, applies_to, is_active)
				values ($1,$2,$3,$4,$5,$6) returning id`,
				tenantID, code, label, amount, appliesTo, active).Scan(&out); err != nil {
				if isUniqueViolation(err) {
					return ErrDuplicate
				}
				return err
			}
			return nil
		}
		return tx.QueryRow(ctx, `update fee_items set code=$2, label=$3, amount=$4, applies_to=$5, is_active=$6
			where id=$1 returning id`, id, code, label, amount, appliesTo, active).Scan(&out)
	})
	return out, err
}

// ---- Unit & penghuni (dasar pembuatan tagihan) ---------------------------

// UnitUntukTagihan — tiap unit beserta penghuni terverifikasi dan pemiliknya.
type UnitUntukTagihan struct {
	UnitID       string
	Block        string
	Number       string
	Status       string
	PenghuniIDs  []string
	PenghuniHP   []string
	PenghuniNama []string
	NamaUnit     string
}

func (s *Store) UnitUntukTagihan(ctx context.Context, tenantID string) ([]UnitUntukTagihan, error) {
	out := []UnitUntukTagihan{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select u.id, u.block, u.unit_number, u.occupancy_status
			from house_units u order by u.block, u.unit_number`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var u UnitUntukTagihan
			if err := rows.Scan(&u.UnitID, &u.Block, &u.Number, &u.Status); err != nil {
				return err
			}
			u.NamaUnit = u.Block + "-" + u.Number
			out = append(out, u)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		// penghuni terverifikasi per unit (yang boleh melihat & membayar tagihan)
		prows, err := tx.Query(ctx, `select o.house_unit_id, p.id, coalesce(u.phone,''), p.full_name
			from house_occupancies o
			join resident_profiles p on p.id = o.resident_id
			left join users u on u.id = p.account_id
			where o.end_date is null and p.verification_status = 'VERIFIED'
			  and p.lifecycle_status = 'ACTIVE'`)
		if err != nil {
			return err
		}
		defer prows.Close()
		idx := map[string]int{}
		for i, u := range out {
			idx[u.UnitID] = i
		}
		for prows.Next() {
			var unitID, pid, hp, nama string
			if err := prows.Scan(&unitID, &pid, &hp, &nama); err != nil {
				return err
			}
			if i, ok := idx[unitID]; ok {
				out[i].PenghuniIDs = append(out[i].PenghuniIDs, pid)
				out[i].PenghuniHP = append(out[i].PenghuniHP, hp)
				out[i].PenghuniNama = append(out[i].PenghuniNama, nama)
			}
		}
		return prows.Err()
	})
	return out, err
}

// UnitUntukWarga — unit yang dihuni satu akun (untuk filter tagihan warga).
func (s *Store) UnitUntukWarga(ctx context.Context, tenantID, userID string) ([]string, error) {
	out := []string{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select distinct o.house_unit_id
			from house_occupancies o
			join resident_profiles p on p.id = o.resident_id
			where p.account_id = $1 and o.end_date is null`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}

// ---- Invoice -------------------------------------------------------------

func (s *Store) InvoiceSudahAda(ctx context.Context, tenantID, unitID, period string) (bool, error) {
	ada := false
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select exists (select 1 from invoices
			where house_unit_id = $1 and period = $2::date and status <> 'VOID')`, unitID, period).Scan(&ada)
	})
	return ada, err
}

// TunggakanUnit — total tagihan belum lunas dengan periode sebelum periode ini.
func (s *Store) TunggakanUnit(ctx context.Context, tenantID, unitID, period string) (float64, []domain.InvoiceItem, error) {
	total := 0.0
	items := []domain.InvoiceItem{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select to_char(period,'YYYY-MM'), total_amount::float8
			from invoices where house_unit_id = $1 and period < $2::date and status = 'UNPAID'
			order by period`, unitID, period)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var per string
			var amt float64
			if err := rows.Scan(&per, &amt); err != nil {
				return err
			}
			total += amt
			items = append(items, domain.InvoiceItem{Label: "Tunggakan periode " + per, Amount: amt, Kind: "ARREARS"})
		}
		return rows.Err()
	})
	return total, items, err
}

// BuatInvoice — menyimpan invoice beserta baris itemnya dalam satu transaksi.
func (s *Store) BuatInvoice(ctx context.Context, tenantID string, inv domain.Invoice, items []domain.InvoiceItem) (string, error) {
	var id string
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `insert into invoices
			(tenant_id, house_unit_id, invoice_number, period, base_amount, arrears_amount, total_amount, status, due_date)
			values ($1,$2,$3,$4::date,$5,$6,$7,'UNPAID',$8::date) returning id`,
			tenantID, inv.HouseUnitID, inv.InvoiceNumber, inv.Period, inv.BaseAmount, inv.ArrearsAmount,
			inv.TotalAmount, inv.DueDate).Scan(&id); err != nil {
			return err
		}
		for _, it := range items {
			if _, err := tx.Exec(ctx, `insert into invoice_items (invoice_id, label, amount, kind)
				values ($1,$2,$3,$4)`, id, it.Label, it.Amount, it.Kind); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

const invoiceCols = `i.id, i.house_unit_id, coalesce(h.block||'-'||h.unit_number,''), i.invoice_number,
	to_char(i.period,'YYYY-MM'), i.base_amount::float8, i.arrears_amount::float8, i.total_amount::float8, i.status,
	i.due_date::text, coalesce(to_char(i.paid_at,'YYYY-MM-DD HH24:MI'),'')`

const invoiceFrom = `invoices i left join house_units h on h.id = i.house_unit_id`

func scanInvoice(row pgx.Row) (*domain.Invoice, error) {
	var v domain.Invoice
	if err := row.Scan(&v.ID, &v.HouseUnitID, &v.HouseUnit, &v.InvoiceNumber, &v.Period,
		&v.BaseAmount, &v.ArrearsAmount, &v.TotalAmount, &v.Status, &v.DueDate, &v.PaidAt); err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Store) InvoiceByID(ctx context.Context, tenantID, id string) (*domain.Invoice, error) {
	var out *domain.Invoice
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `select `+invoiceCols+` from `+invoiceFrom+` where i.id = $1`, id)
		inv, err := scanInvoice(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrBillingNotFound
			}
			return err
		}
		items, err := itemsOf(ctx, tx, inv.ID)
		if err != nil {
			return err
		}
		inv.Items = items
		out = inv
		return nil
	})
	return out, err
}

func itemsOf(ctx context.Context, tx pgx.Tx, invoiceID string) ([]domain.InvoiceItem, error) {
	out := []domain.InvoiceItem{}
	rows, err := tx.Query(ctx, `select label, amount::float8, kind from invoice_items where invoice_id = $1 order by kind, label`, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it domain.InvoiceItem
		if err := rows.Scan(&it.Label, &it.Amount, &it.Kind); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// DaftarInvoice — staff melihat seluruh tenant; warga hanya unit yang dihuninya.
func (s *Store) DaftarInvoice(ctx context.Context, tenantID string, unitIDs []string, status string) ([]domain.Invoice, error) {
	out := []domain.Invoice{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		q := `select ` + invoiceCols + ` from ` + invoiceFrom + ` where 1=1`
		args := []any{}
		n := 0
		if unitIDs != nil {
			n++
			q += fmt.Sprintf(` and i.house_unit_id = any($%d::uuid[])`, n)
			args = append(args, unitIDs)
		}
		if status != "" {
			n++
			q += fmt.Sprintf(` and i.status = $%d`, n)
			args = append(args, status)
		}
		q += ` order by i.period desc, h.block, h.unit_number limit 300`
		rows, err := tx.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			inv, err := scanInvoice(rows)
			if err != nil {
				return err
			}
			out = append(out, *inv)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) SimpanQRIS(ctx context.Context, tenantID, invoiceID, reference, qrString string, expires time.Time) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update invoices set qris_reference=$2, qris_string=$3,
			qris_expires_at=$4, updated_at=now() where id=$1`, invoiceID, reference, qrString, expires)
		return err
	})
}

// TandaiLunas — idempoten: hanya mengubah bila status masih UNPAID.
func (s *Store) TandaiLunas(ctx context.Context, tenantID, invoiceID string) (bool, error) {
	berubah := false
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `update invoices set status='PAID', paid_at=now(), updated_at=now()
			where id=$1 and status='UNPAID'`, invoiceID)
		if err != nil {
			return err
		}
		berubah = tag.RowsAffected() > 0
		return nil
	})
	return berubah, err
}

func (s *Store) NomorKuitansi(ctx context.Context, tenantID, seq string) (string, error) {
	return "KW/" + time.Now().Format("200601") + "/" + seq, nil
}

func (s *Store) BuatPembayaran(ctx context.Context, tenantID, invoiceID, payerResidentID, receivedBy, receiptNumber, method string, amount float64, gatewayRef, notes string) (string, error) {
	var id string
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `insert into payments
			(tenant_id, invoice_id, payer_resident_id, received_by, receipt_number, payment_method, amount_paid, gateway_reference, notes)
			values ($1,$2,nullif($3,'')::uuid,nullif($4,'')::uuid,$5,$6,$7,nullif($8,''),nullif($9,'')) returning id`,
			tenantID, invoiceID, payerResidentID, receivedBy, receiptNumber, method, amount, gatewayRef, notes).Scan(&id)
	})
	return id, err
}

func (s *Store) PembayaranInvoice(ctx context.Context, tenantID, invoiceID string) (*domain.Payment, error) {
	var out *domain.Payment
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `select id, invoice_id, receipt_number, payment_method, amount_paid::float8,
			coalesce(gateway_reference,''), coalesce(notes,''), to_char(created_at,'YYYY-MM-DD HH24:MI')
			from payments where invoice_id = $1`, invoiceID)
		var v domain.Payment
		if err := row.Scan(&v.ID, &v.InvoiceID, &v.ReceiptNumber, &v.Method, &v.AmountPaid,
			&v.GatewayRef, &v.Notes, &v.CreatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		out = &v
		return nil
	})
	return out, err
}

// ---- Buku kas ------------------------------------------------------------

func (s *Store) CatatKas(ctx context.Context, tenantID, paymentID, createdBy, sourceType, trxType string, amount float64, group, notes string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `insert into cash_ledgers
			(tenant_id, payment_id, created_by, source_type, transaction_type, amount, transfer_group, notes)
			values ($1,nullif($2,'')::uuid,nullif($3,'')::uuid,$4,$5,$6,nullif($7,'')::uuid,nullif($8,''))`,
			tenantID, paymentID, createdBy, sourceType, trxType, amount, group, notes)
		return err
	})
}

func (s *Store) DaftarKas(ctx context.Context, tenantID string, limit int) ([]domain.CashLedger, error) {
	out := []domain.CashLedger{}
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select l.id, l.source_type, l.transaction_type, l.amount::float8,
			coalesce(l.notes,''), coalesce(i.invoice_number,''),
			coalesce(to_char(l.approved_at,'YYYY-MM-DD HH24:MI'),''), to_char(l.created_at,'YYYY-MM-DD HH24:MI'),
			coalesce(l.transfer_group::text,'')
			from cash_ledgers l
			left join payments p on p.id = l.payment_id
			left join invoices i on i.id = p.invoice_id
			order by l.created_at desc limit $1`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var v domain.CashLedger
			if err := rows.Scan(&v.ID, &v.SourceType, &v.TransactionType, &v.Amount, &v.Notes,
				&v.InvoiceNumber, &v.ApprovedAt, &v.LedgerDate, &v.TransferGroup); err != nil {
				return err
			}
			out = append(out, v)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) SaldoKas(ctx context.Context, tenantID string) (*domain.SaldoKas, error) {
	out := &domain.SaldoKas{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select source_type,
			sum(case when transaction_type='IN' then amount else -amount end)::float8
			from cash_ledgers group by source_type`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var src string
			var v float64
			if err := rows.Scan(&src, &v); err != nil {
				return err
			}
			switch src {
			case domain.LedgerBank:
				out.BankGateway = v
			case domain.LedgerPetty:
				out.PettyCash = v
			case domain.LedgerAcct:
				out.BankAccount = v
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `select count(*) from cash_ledgers where transfer_group is not null and approved_at is null`).Scan(&out.MenungguPersetujuan)
	})
	return out, err
}

func (s *Store) SetujuiTransfer(ctx context.Context, tenantID, groupID, userID string) (int64, error) {
	var n int64
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `update cash_ledgers set approved_by=$2, approved_at=now()
			where transfer_group=$1 and approved_at is null`, groupID, userID)
		if err != nil {
			return err
		}
		n = tag.RowsAffected()
		return nil
	})
	return n, err
}

// TenantNama — dipakai untuk menyebut nama lingkungan pada pesan WhatsApp.
func (s *Store) TenantNama(ctx context.Context, tenantID string) (string, error) {
	nama := ""
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select name from tenants where id = $1`, tenantID).Scan(&nama)
	})
	return nama, err
}

// QRISTersimpan — reference & payload QRIS yang terakhir dibuat untuk invoice.
func (s *Store) QRISTersimpan(ctx context.Context, tenantID, invoiceID string) (string, string, time.Time, error) {
	ref, qr := "", ""
	exp := time.Time{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select coalesce(qris_reference,''), coalesce(qris_string,''),
			coalesce(qris_expires_at, now()) from invoices where id = $1`, invoiceID).Scan(&ref, &qr, &exp)
	})
	return ref, qr, exp, err
}

// LampirkanBukti — menautkan berkas bukti setoran ke seluruh baris mutasi grup.
func (s *Store) LampirkanBukti(ctx context.Context, tenantID, groupID, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update cash_ledgers set evidence_file_path = $2 where transfer_group = $1`, groupID, path)
		return err
	})
}

// Tenants — daftar tenant aktif. Dipakai tugas cron; dijalankan dengan koneksi
// admin (root) sehingga tidak bergantung pada konteks RLS satu tenant.
func (s *Store) Tenants(ctx context.Context) ([]struct{ ID, Nama string }, error) {
	out := []struct{ ID, Nama string }{}
	rows, err := s.Pool.Query(ctx, `select id::text, name from tenants where status = 'ACTIVE' order by name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v struct{ ID, Nama string }
		if err := rows.Scan(&v.ID, &v.Nama); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UUIDBaru — UUID v4 dari database, dipakai untuk mengelompokkan mutasi setoran
// (kolom transfer_group bertipe uuid, bukan teks bebas).
func (s *Store) UUIDBaru(ctx context.Context) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `select gen_random_uuid()::text`).Scan(&id)
	return id, err
}
