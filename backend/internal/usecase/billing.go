package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/hub"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

var (
	ErrBillingForbidden = errors.New("tidak berhak atas data keuangan ini")
	ErrBillingNotFound  = errors.New("tagihan tidak ditemukan")
	ErrSudahLunas       = errors.New("tagihan sudah lunas")
	ErrBelumLunas       = errors.New("tagihan belum lunas")
	ErrGatewayBelumAktif = errors.New("QRIS belum bisa diterbitkan: akun pembayaran belum aktif")
)

// Billing — tagihan iuran: pembuatan bulanan, QRIS, kas tunai, buku kas.
type Billing struct {
	Store  *postgres.Store
	Hub    *hub.Client
	Notify Notifier
}

func (b *Billing) isStaff(sub *domain.SubjectContext) bool {
	return domain.CanAccess(*sub, domain.ResourceContext{TenantID: sub.TenantID}, domain.ActionManageBilling)
}

// rupiah — format nominal untuk pesan WhatsApp.
func rupiah(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(c)
	}
	return "Rp" + out
}

// HasilGenerate — ringkasan proses pembuatan tagihan massal.
type HasilGenerate struct {
	Periode    string `json:"periode"`
	Dibuat     int    `json:"dibuat"`
	Dilewati   int    `json:"dilewati"`
	Ditagih    int    `json:"total_unit"`
	TerkirimWA int    `json:"notifikasi_terkirim"`
	GagalWA    int    `json:"notifikasi_gagal"`
}

// GenerateBulanan — menerbitkan tagihan untuk semua unit pada satu periode.
// Aturan PRD: unit kosong dapat komponen terbatas, tunggakan ikut ditagihkan,
// dan satu unit hanya boleh punya satu invoice per periode.
func (b *Billing) GenerateBulanan(ctx context.Context, sub *domain.SubjectContext, periode string) (*HasilGenerate, error) {
	if !b.isStaff(sub) {
		return nil, ErrBillingForbidden
	}
	periode = strings.TrimSpace(periode)
	if periode == "" {
		periode = time.Now().Format("2006-01")
	}
	awal, err := time.Parse("2006-01", periode)
	if err != nil {
		return nil, fmt.Errorf("%w: periode harus format YYYY-MM", ErrBadInput)
	}
	_, dueDays, err := b.Store.BillingSettings(ctx, sub.TenantID)
	if err != nil {
		return nil, err
	}
	fees, err := b.Store.FeeItems(ctx, sub.TenantID)
	if err != nil {
		return nil, err
	}
	units, err := b.Store.UnitUntukTagihan(ctx, sub.TenantID)
	if err != nil {
		return nil, err
	}
	tgl, _ := b.Store.TenantNama(ctx, sub.TenantID)
	// kolom period bertipe DATE: simpan hari pertama bulan tersebut
	tglPeriode := awal.Format("2006-01-02")

	out := &HasilGenerate{Periode: periode, Ditagih: len(units)}
	for _, u := range units {
		ada, err := b.Store.InvoiceSudahAda(ctx, sub.TenantID, u.UnitID, tglPeriode)
		if err != nil {
			return nil, err
		}
		if ada {
			out.Dilewati++
			continue
		}
		kategori := domain.UnitOccupied
		if u.Status == domain.UnitVacant {
			kategori = domain.UnitVacant
		}
		items := []domain.InvoiceItem{}
		base := 0.0
		for _, f := range fees {
			if !f.IsActive {
				continue
			}
			if f.AppliesTo == "ALL" || f.AppliesTo == kategori {
				items = append(items, domain.InvoiceItem{Label: f.Label, Amount: f.Amount, Kind: "CURRENT"})
				base += f.Amount
			}
		}
		if base == 0 {
			out.Dilewati++
			continue
		}
		tunggakan, tItems, err := b.Store.TunggakanUnit(ctx, sub.TenantID, u.UnitID, tglPeriode)
		if err != nil {
			return nil, err
		}
		items = append(items, tItems...)
		nomor := fmt.Sprintf("INV-%s-%s%s", periode, u.Block, u.Number)
		inv := domain.Invoice{
			HouseUnitID:   u.UnitID,
			InvoiceNumber: nomor,
			Period:        tglPeriode,
			BaseAmount:    base,
			ArrearsAmount: tunggakan,
			TotalAmount:   base + tunggakan,
			DueDate:       awal.AddDate(0, 1, dueDays).Format("2006-01-02"),
		}
		if _, err := b.Store.BuatInvoice(ctx, sub.TenantID, inv, items); err != nil {
			return nil, err
		}
		out.Dibuat++
		// Notifikasi WhatsApp ke seluruh penghuni terverifikasi unit ini.
		pesan := fmt.Sprintf("Tagihan iuran %s untuk rumah %s: %s (jatuh tempo %s). Bayar via QRIS di aplikasi Smarthub atau setor ke Bendahara.",
			periode, u.NamaUnit, rupiah(inv.TotalAmount), inv.DueDate)
		for _, hp := range u.PenghuniHP {
			if strings.TrimSpace(hp) == "" {
				continue
			}
			if err := b.Notify.SendTagihan(hp, pesan, tgl); err != nil {
				log.Printf("[billing] notifikasi tagihan gagal: %v", err)
				out.GagalWA++
				continue
			}
			out.TerkirimWA++
		}
		_ = b.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "INVOICE_GENERATE", "invoice", nomor,
			map[string]any{"unit": u.NamaUnit, "total": inv.TotalAmount})
	}
	return out, nil
}

// bolehLihatInvoice — warga hanya unit yang dihuninya; pengurus seluruh tenant.
func (b *Billing) unitWarga(ctx context.Context, sub *domain.SubjectContext) ([]string, error) {
	if b.isStaff(sub) {
		return nil, nil // nil = tanpa filter
	}
	return b.Store.UnitUntukWarga(ctx, sub.TenantID, sub.AccountID)
}

func (b *Billing) Daftar(ctx context.Context, sub *domain.SubjectContext, status string) ([]domain.Invoice, error) {
	units, err := b.unitWarga(ctx, sub)
	if err != nil {
		return nil, err
	}
	if units != nil && len(units) == 0 {
		return []domain.Invoice{}, nil
	}
	return b.Store.DaftarInvoice(ctx, sub.TenantID, units, status)
}

func (b *Billing) Detail(ctx context.Context, sub *domain.SubjectContext, id string) (*domain.Invoice, error) {
	inv, err := b.Store.InvoiceByID(ctx, sub.TenantID, id)
	if err != nil {
		if errors.Is(err, postgres.ErrBillingNotFound) {
			return nil, ErrBillingNotFound
		}
		return nil, err
	}
	res := domain.ResourceContext{TenantID: sub.TenantID, HouseUnitID: inv.HouseUnitID}
	if !domain.CanAccess(*sub, res, domain.ActionViewInvoice) {
		return nil, ErrBillingForbidden
	}
	return inv, nil
}

// QRIS — hasil pembuatan QRIS dinamis.
type QRIS struct {
	InvoiceID string `json:"invoice_id"`
	Reference string `json:"reference_id"`
	QRString  string `json:"qr_string"`
	Amount    float64 `json:"amount"`
	ExpiresAt string `json:"expires_at"`
	DibuatBaru bool  `json:"dibuat_baru"`
}

// BuatQRIS — membuat (atau memakai ulang) QRIS dinamis untuk sebuah invoice.
func (b *Billing) BuatQRIS(ctx context.Context, sub *domain.SubjectContext, invoiceID string) (*QRIS, error) {
	inv, err := b.prepareBayar(ctx, sub, invoiceID)
	if err != nil {
		return nil, err
	}
	ref, qr, exp := "", "", time.Time{}
	if r, q, e, err := b.Store.QRISTersimpan(ctx, sub.TenantID, inv.ID); err == nil && r != "" && e.After(time.Now()) {
		ref, qr, exp = r, q, e
	}
	if ref == "" {
		if !b.Hub.Enabled() {
			return nil, fmt.Errorf("gateway QRIS belum dikonfigurasi")
		}
		resp, err := b.Hub.CreateQRIS(ctx, b.Hub.ExternalID(inv.InvoiceNumber), inv.TotalAmount,
			"Iuran "+inv.Period+" "+inv.HouseUnit)
		if errors.Is(err, hub.ErrGatewayBermasalah) {
			log.Printf("[billing] gateway menolak pembuatan QRIS untuk %s: %v", inv.InvoiceNumber, err)
			return nil, ErrGatewayBelumAktif
		}
		if errors.Is(err, hub.ErrQRISTidakSah) {
			// Menolak menampilkan QR palsu ke warga lebih baik daripada QRIS gagal scan.
			log.Printf("[billing] hub mengembalikan QR placeholder untuk %s (sub-akun pembayaran belum aktif)", inv.InvoiceNumber)
			return nil, ErrGatewayBelumAktif
		}
		if err != nil {
			return nil, err
		}
		ref, qr = resp.ReferenceID, resp.QRString
		exp = time.Now().Add(30 * time.Minute)
		if t, err := time.Parse(time.RFC3339, resp.ExpiresAt); err == nil {
			exp = t
		}
		if err := b.Store.SimpanQRIS(ctx, sub.TenantID, inv.ID, ref, qr, exp); err != nil {
			return nil, err
		}
		_ = b.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "INV_QRIS_CREATE", "invoice", inv.InvoiceNumber,
			map[string]any{"reference": ref, "total": inv.TotalAmount})
	}
	return &QRIS{InvoiceID: inv.ID, Reference: ref, QRString: qr, Amount: inv.TotalAmount,
		ExpiresAt: exp.Format(time.RFC3339), DibuatBaru: false}, nil
}

// prepareBayar — validasi izin & status sebelum pembayaran.
func (b *Billing) prepareBayar(ctx context.Context, sub *domain.SubjectContext, invoiceID string) (*domain.Invoice, error) {
	inv, err := b.Store.InvoiceByID(ctx, sub.TenantID, invoiceID)
	if err != nil {
		if errors.Is(err, postgres.ErrBillingNotFound) {
			return nil, ErrBillingNotFound
		}
		return nil, err
	}
	res := domain.ResourceContext{TenantID: sub.TenantID, HouseUnitID: inv.HouseUnitID}
	if !domain.CanAccess(*sub, res, domain.ActionPayInvoice) && !b.isStaff(sub) {
		return nil, ErrBillingForbidden
	}
	if inv.Status == domain.InvoicePaid {
		return nil, ErrSudahLunas
	}
	if inv.Status == domain.InvoiceVoid {
		return nil, ErrBelumLunas
	}
	return inv, nil
}

// CekQRIS — memeriksa status ke hub lalu menandai lunas (idempoten).
// Karena hub tidak mengirim webhook, fungsi ini yang menjadi jembatan
// "QRIS lunas -> invoice lunas", dan dipanggil juga oleh proses rekonsiliasi.
func (b *Billing) CekQRIS(ctx context.Context, sub *domain.SubjectContext, invoiceID string) (*domain.Invoice, string, error) {
	inv, err := b.Store.InvoiceByID(ctx, sub.TenantID, invoiceID)
	if err != nil {
		if errors.Is(err, postgres.ErrBillingNotFound) {
			return nil, "", ErrBillingNotFound
		}
		return nil, "", err
	}
	res := domain.ResourceContext{TenantID: sub.TenantID, HouseUnitID: inv.HouseUnitID}
	if !domain.CanAccess(*sub, res, domain.ActionViewInvoice) {
		return nil, "", ErrBillingForbidden
	}
	if inv.Status == domain.InvoicePaid {
		return inv, "PAID", nil
	}
	ref, _, exp, err := b.Store.QRISTersimpan(ctx, sub.TenantID, inv.ID)
	if err != nil || ref == "" {
		return inv, "BELUM_ADA_QRIS", nil
	}
	if !b.Hub.Enabled() {
		return inv, "GATEWAY_MATI", nil
	}
	st, err := b.Hub.Status(ctx, ref)
	if err != nil {
		return inv, "GAGAL_CEK", err
	}
	status := strings.ToUpper(strings.TrimSpace(st.Status))
	if !st.SudahLunas() {
		if strings.TrimSpace(st.Status) == "" || exp.Before(time.Now()) {
			return inv, "KEDALUWARSA", nil
		}
		return inv, status, nil
	}
	// Pengaman: nominal dari gateway harus sama dengan tagihan.
	if st.Amount > 0 && abs(st.Amount-inv.TotalAmount) > 0.01 {
		log.Printf("[billing] nominal QRIS %s tidak cocok: gateway %.0f vs tagihan %.0f", ref, st.Amount, inv.TotalAmount)
		return inv, "NOMINAL_TIDAK_COCOK", nil
	}
	if err := b.lunasi(ctx, sub.TenantID, inv, domain.PayQRIS, ref, sub.AccountID, ""); err != nil {
		return inv, status, err
	}
	baru, err := b.Store.InvoiceByID(ctx, sub.TenantID, inv.ID)
	return baru, "PAID", err
}

// lunasi — menandai invoice lunas + membuat pembayaran & baris buku kas.
// Idempoten: kalau invoice sudah PAID, tidak ada yang dibuat lagi.
func (b *Billing) lunasi(ctx context.Context, tenantID string, inv *domain.Invoice, method, gatewayRef, receivedBy, notes string) error {
	berubah, err := b.Store.TandaiLunas(ctx, tenantID, inv.ID)
	if err != nil {
		return err
	}
	if !berubah {
		return nil
	}
	seq := strings.ReplaceAll(inv.InvoiceNumber, "INV-", "")
	nomor, _ := b.Store.NomorKuitansi(ctx, tenantID, seq)
	payID, err := b.Store.BuatPembayaran(ctx, tenantID, inv.ID, "", receivedBy, nomor, method, inv.TotalAmount, gatewayRef, notes)
	if err != nil {
		return err
	}
	sumber := domain.LedgerBank
	if method == domain.PayCash {
		sumber = domain.LedgerPetty
	}
	if err := b.Store.CatatKas(ctx, tenantID, payID, receivedBy, sumber, "IN", inv.TotalAmount, "", "Pembayaran "+inv.InvoiceNumber); err != nil {
		return err
	}
	_ = b.Store.LogAudit(ctx, tenantID, receivedBy, "INVOICE_PAID", "invoice", inv.InvoiceNumber,
		map[string]any{"metode": method, "total": inv.TotalAmount, "kuitansi": nomor})
	// Kuitansi digital dikirim ke seluruh penghuni unit.
	units, err := b.Store.UnitUntukTagihan(ctx, tenantID)
	if err == nil {
		for _, u := range units {
			if u.UnitID != inv.HouseUnitID {
				continue
			}
			pesan := "Pembayaran diterima untuk rumah " + u.NamaUnit + " (" + inv.Period + ") sebesar " +
				rupiah(inv.TotalAmount) + ". Kuitansi digital: " + nomor + "."
			for _, hp := range u.PenghuniHP {
				if strings.TrimSpace(hp) == "" {
					continue
				}
				if err := b.Notify.SendTagihan(hp, pesan, ""); err != nil {
					log.Printf("[billing] kuitansi WA gagal: %v", err)
				}
			}
		}
	}
	return nil
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// CatatTunai — Bendahara mencatat uang tunai yang diterimanya (PRD: kas fisik).
func (b *Billing) CatatTunai(ctx context.Context, sub *domain.SubjectContext, invoiceID string, nominal float64, catatan string) (*domain.Invoice, error) {
	if !b.isStaff(sub) {
		return nil, ErrBillingForbidden
	}
	if nominal <= 0 {
		return nil, fmt.Errorf("%w: nominal harus lebih dari nol", ErrBadInput)
	}
	inv, err := b.Store.InvoiceByID(ctx, sub.TenantID, invoiceID)
	if err != nil {
		if errors.Is(err, postgres.ErrBillingNotFound) {
			return nil, ErrBillingNotFound
		}
		return nil, err
	}
	if inv.Status == domain.InvoicePaid {
		return nil, ErrSudahLunas
	}
	// Nominal tunai harus sama dengan tagihan; selisih = salah input, bukan diterima diam-diam.
	if abs(nominal-inv.TotalAmount) > 0.01 {
		return nil, fmt.Errorf("%w: nominal %s tidak sama dengan tagihan %s", ErrBadInput, rupiah(nominal), rupiah(inv.TotalAmount))
	}
	if err := b.lunasi(ctx, sub.TenantID, inv, domain.PayCash, "", sub.AccountID, catatan); err != nil {
		return nil, err
	}
	return b.Store.InvoiceByID(ctx, sub.TenantID, inv.ID)
}

// BukuKas — daftar mutasi + saldo per sumber dana.
func (b *Billing) BukuKas(ctx context.Context, sub *domain.SubjectContext) ([]domain.CashLedger, *domain.SaldoKas, error) {
	if !b.isStaff(sub) {
		return nil, nil, ErrBillingForbidden
	}
	list, err := b.Store.DaftarKas(ctx, sub.TenantID, 100)
	if err != nil {
		return nil, nil, err
	}
	saldo, err := b.Store.SaldoKas(ctx, sub.TenantID)
	return list, saldo, err
}

// SetorBank — mutasi kas fisik -> rekening paguyuban, menunggu persetujuan Ketua.
func (b *Billing) SetorBank(ctx context.Context, sub *domain.SubjectContext, nominal float64, catatan, bukti string) (string, error) {
	if !b.isStaff(sub) {
		return "", ErrBillingForbidden
	}
	if nominal <= 0 {
		return "", fmt.Errorf("%w: nominal harus lebih dari nol", ErrBadInput)
	}
	saldo, err := b.Store.SaldoKas(ctx, sub.TenantID)
	if err != nil {
		return "", err
	}
	if nominal > saldo.PettyCash+0.01 {
		return "", fmt.Errorf("%w: saldo kas fisik hanya %s", ErrBadInput, rupiah(saldo.PettyCash))
	}
	grup, err := b.Store.UUIDBaru(ctx)
	if err != nil {
		return "", err
	}
	if err := b.Store.CatatKas(ctx, sub.TenantID, "", sub.AccountID, domain.LedgerPetty, "OUT", nominal, grup, "Setor ke bank: "+catatan); err != nil {
		return "", err
	}
	if err := b.Store.CatatKas(ctx, sub.TenantID, "", sub.AccountID, domain.LedgerAcct, "IN", nominal, grup, "Setoran bank: "+catatan); err != nil {
		return "", err
	}
	if err := b.Store.LampirkanBukti(ctx, sub.TenantID, grup, bukti); err != nil {
		return "", err
	}
	_ = b.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "CASH_DEPOSIT", "cash_ledger", grup,
		map[string]any{"nominal": nominal, "catatan": catatan})
	return grup, nil
}

// SetujuiSetoran — Ketua memverifikasi mutasi setoran.
func (b *Billing) SetujuiSetoran(ctx context.Context, sub *domain.SubjectContext, grup string) error {
	if !domain.CanAccess(*sub, domain.ResourceContext{TenantID: sub.TenantID}, domain.ActionManageUser) {
		return ErrBillingForbidden
	}
	n, err := b.Store.SetujuiTransfer(ctx, sub.TenantID, grup, sub.AccountID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrBillingNotFound
	}
	_ = b.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "CASH_DEPOSIT_APPROVE", "cash_ledger", grup, nil)
	return nil
}

// MasterIuran — daftar & simpan komponen iuran (Bendahara).
func (b *Billing) MasterIuran(ctx context.Context, sub *domain.SubjectContext) ([]domain.FeeItem, error) {
	if !b.isStaff(sub) {
		return nil, ErrBillingForbidden
	}
	return b.Store.FeeItems(ctx, sub.TenantID)
}

func (b *Billing) SimpanIuran(ctx context.Context, sub *domain.SubjectContext, id, code, label string, amount float64, appliesTo string, aktif bool) (string, error) {
	if !b.isStaff(sub) {
		return "", ErrBillingForbidden
	}
	code = strings.ToUpper(strings.TrimSpace(code))
	appliesTo = strings.ToUpper(strings.TrimSpace(appliesTo))
	if code == "" || strings.TrimSpace(label) == "" {
		return "", fmt.Errorf("%w: kode dan nama iuran wajib", ErrBadInput)
	}
	if amount <= 0 {
		return "", fmt.Errorf("%w: nominal harus lebih dari nol", ErrBadInput)
	}
	if appliesTo != "ALL" && appliesTo != "OCCUPIED" && appliesTo != "VACANT" {
		return "", fmt.Errorf("%w: sasaran harus ALL, OCCUPIED, atau VACANT", ErrBadInput)
	}
	id, err := b.Store.UpsertFeeItem(ctx, sub.TenantID, id, code, label, amount, appliesTo, aktif)
	if errors.Is(err, postgres.ErrDuplicate) {
		return "", fmt.Errorf("%w: kode iuran %s sudah ada", ErrConflict, code)
	}
	if err == nil {
		_ = b.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "FEE_ITEM_SAVE", "fee_item", code,
			map[string]any{"nominal": amount, "sasaran": appliesTo})
	}
	return id, err
}

// PengaturanPenagihan — tanggal terbit & jatuh tempo.
func (b *Billing) PengaturanPenagihan(ctx context.Context, sub *domain.SubjectContext) (int, int, error) {
	if !b.isStaff(sub) {
		return 0, 0, ErrBillingForbidden
	}
	return b.Store.BillingSettings(ctx, sub.TenantID)
}

func (b *Billing) SimpanPengaturan(ctx context.Context, sub *domain.SubjectContext, hari, tempo int) error {
	if !b.isStaff(sub) {
		return ErrBillingForbidden
	}
	if hari < 1 || hari > 28 {
		return fmt.Errorf("%w: tanggal terbit harus 1-28", ErrBadInput)
	}
	if tempo < 1 || tempo > 60 {
		return fmt.Errorf("%w: jatuh tempo harus 1-60 hari", ErrBadInput)
	}
	return b.Store.SetBillingSettings(ctx, sub.TenantID, hari, tempo)
}
