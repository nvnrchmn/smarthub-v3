package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// RekapBulanan — laporan satu periode: ditagih, dibayar, tunggakan, rincian, kas.
func (s *Store) RekapBulanan(ctx context.Context, tenantID, periode string) (*domain.RekapBulanan, error) {
	out := &domain.RekapBulanan{Periode: periode, Rincian: []domain.RekapItem{}}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `select name from tenants where id = current_setting('app.tenant_id')::uuid`).
			Scan(&out.TenantNama); err != nil && err != pgx.ErrNoRows {
			return err
		}
		if err := tx.QueryRow(ctx, `select count(*),
				coalesce(sum(total_amount),0)::float8,
				coalesce(sum(total_amount) filter (where status='PAID'),0)::float8,
				coalesce(sum(total_amount) filter (where status<>'PAID' and status<>'VOID'),0)::float8,
				count(*) filter (where status='PAID'),
				count(*) filter (where status<>'PAID' and status<>'VOID')
			from invoices where period = to_date($1, 'YYYY-MM')`, periode).
			Scan(&out.JumlahInvoice, &out.TotalDitagih, &out.TotalDibayar,
				&out.TotalTunggakan, &out.JumlahLunas, &out.JumlahBelum); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `select it.label, it.kind, count(*), coalesce(sum(it.amount),0)::float8
			from invoice_items it join invoices i on i.id = it.invoice_id
			where i.period = to_date($1, 'YYYY-MM')
			group by it.label, it.kind order by 4 desc`, periode)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var r domain.RekapItem
			if err := rows.Scan(&r.Label, &r.Kind, &r.Jumlah, &r.Amount); err != nil {
				return err
			}
			out.Rincian = append(out.Rincian, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `select
				coalesce(sum(p.amount_paid) filter (where upper(p.payment_method) not like '%QRIS%'),0)::float8,
				coalesce(sum(p.amount_paid) filter (where upper(p.payment_method) like '%QRIS%'),0)::float8
			from payments p join invoices i on i.id = p.invoice_id
			where i.period = to_date($1, 'YYYY-MM')`, periode).Scan(&out.KasTunai, &out.KasQRIS)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Tunggakan — semua tagihan belum lunas per unit, plus kelompok umur (aging).
func (s *Store) Tunggakan(ctx context.Context, tenantID string) (*domain.Tunggakan, error) {
	out := &domain.Tunggakan{Baris: []domain.TunggakanBaris{}, Buckets: []domain.BucketTunggakan{}}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `select to_char(current_date,'YYYY-MM-DD')`).Scan(&out.AsOf); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `select u.block||'-'||u.unit_number, coalesce(p.full_name,''), coalesce(us.phone,''),
				to_char(min(i.period),'YYYY-MM'), count(*), sum(i.total_amount)::float8,
				greatest(0, current_date - min(i.due_date))::int
			from invoices i
			join house_units u on u.id = i.house_unit_id
			left join house_occupancies o on o.house_unit_id = u.id and o.end_date is null
			left join resident_profiles p on p.id = o.resident_id
			left join users us on us.id = p.account_id
			where i.status = 'UNPAID'
			group by 1, p.full_name, us.phone
			order by 6 desc`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var b domain.TunggakanBaris
			if err := rows.Scan(&b.Unit, &b.KepalaKeluarga, &b.Telepon, &b.PeriodeTertua,
				&b.JumlahTagihan, &b.TotalTunggakan, &b.HariTerlambat); err != nil {
				return err
			}
			switch {
			case b.HariTerlambat <= 30:
				b.Bucket = "0-30 hari"
			case b.HariTerlambat <= 60:
				b.Bucket = "31-60 hari"
			case b.HariTerlambat <= 90:
				b.Bucket = "61-90 hari"
			default:
				b.Bucket = ">90 hari"
			}
			out.Baris = append(out.Baris, b)
			out.Total += b.TotalTunggakan
		}
		if err := rows.Err(); err != nil {
			return err
		}
		out.JumlahUnit = len(out.Baris)
		return nil
	})
	if err != nil {
		return nil, err
	}
	urutan := []string{"0-30 hari", "31-60 hari", "61-90 hari", ">90 hari"}
	for _, label := range urutan {
		bk := domain.BucketTunggakan{Label: label}
		for _, b := range out.Baris {
			if b.Bucket == label {
				bk.Jumlah++
				bk.Amount += b.TotalTunggakan
			}
		}
		out.Buckets = append(out.Buckets, bk)
	}
	return out, nil
}

// Pengurus — pengurus perumahan yang punya nomor WhatsApp (penerima laporan).
func (s *Store) Pengurus(ctx context.Context, tenantID string) ([]domain.Pengurus, error) {
	out := []domain.Pengurus{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select coalesce(full_name,''), role, phone
			from users
			where status = 'active' and coalesce(phone,'') <> ''
			  and role in ('TENANT_MANAGER','TREASURER','SECRETARY')
			order by role`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p domain.Pengurus
			if err := rows.Scan(&p.Nama, &p.Role, &p.Phone); err != nil {
				return err
			}
			out = append(out, p)
		}
		return rows.Err()
	})
	return out, err
}
