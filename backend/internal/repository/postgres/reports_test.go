package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestReportsIntegration — menguji query laporan (RekapBulanan, Tunggakan,
// Pengurus) terhadap Postgres sungguhan. Dilewati bila TEST_DATABASE_URL kosong,
// supaya `go test ./...` di CI (tanpa basis data) tetap hijau.
//
// Jalankan manual:
//
//	TEST_DATABASE_URL="postgres://.../smarthub_test" go test ./internal/repository/postgres/ -run TestReportsIntegration -v
func TestReportsIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL kosong — tes integrasi dilewati")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("koneksi gagal: %v", err)
	}
	defer pool.Close()

	s := &Store{Pool: pool}

	// ID unik supaya tidak bertabrakan dengan data lain.
	tenantID := uuid.NewString()
	userID := uuid.NewString()
	unitID := uuid.NewString()
	unitID2 := uuid.NewString()
	invPaid := uuid.NewString()
	invUnpaid := uuid.NewString()
	itemPaid := uuid.NewString()
	itemUnpaid := uuid.NewString()
	payID := uuid.NewString()

	// Satu koneksi khusus untuk seed & cleanup. RLS menuntut app.tenant_id diset,
	// dan set_config(..., is_local=true) hanya berlaku dalam satu transaksi — jadi
	// seed & cleanup masing-masing dibungkus transaksi (pola sama dengan Store.WithTenant).
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer conn.Release()

	// Bersihkan di akhir (transaksi tersendiri, urutan menuruti FK).
	defer func() {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return
		}
		defer tx.Rollback(ctx)
		_, _ = tx.Exec(ctx, `select set_config('app.tenant_id', $1, true)`, tenantID)
		_, _ = tx.Exec(ctx, `delete from payments where id = $1`, payID)
		_, _ = tx.Exec(ctx, `delete from invoice_items where invoice_id = any($1::uuid[])`, []string{invPaid, invUnpaid})
		_, _ = tx.Exec(ctx, `delete from invoices where id = any($1::uuid[])`, []string{invPaid, invUnpaid})
		_, _ = tx.Exec(ctx, `delete from house_units where id = any($1::uuid[])`, []string{unitID, unitID2})
		_, _ = tx.Exec(ctx, `delete from users where id = $1`, userID)
		_, _ = tx.Exec(ctx, `delete from tenants where id = $1`, tenantID)
		_ = tx.Commit(ctx)
	}()

	// --- Seed: tenant, pengurus (TENANT_MANAGER dengan nomor WA), satu unit,
	// dua invoice periode 2026-09 (satu LUNAS, satu BELUM), satu pembayaran tunai.
	seedTx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin seed: %v", err)
	}
	if _, err := seedTx.Exec(ctx, `select set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
		t.Fatalf("set_config: %v", err)
	}
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := seedTx.Exec(ctx, q, args...); err != nil {
			t.Fatalf("seed gagal: %v\nSQL: %s", err, q)
		}
	}

	mustExec(`insert into tenants (id, name, slug, status, created_at)
		values ($1, 'Test Tenant', 'test-tenant', 'active', now())`, tenantID)
	mustExec(`insert into users (id, tenant_id, email, phone, full_name, password_hash, role, status, created_at)
		values ($1, $2, 'test@example.com', '08123456789', 'Kepala Test', 'x', 'TENANT_MANAGER', 'ACTIVE', now())`,
		userID, tenantID)
	mustExec(`insert into house_units (id, tenant_id, block, unit_number, occupancy_status, created_at, updated_at)
		values ($1, $2, 'A', '01', 'OCCUPIED', now(), now())`, unitID, tenantID)
	mustExec(`insert into house_units (id, tenant_id, block, unit_number, occupancy_status, created_at, updated_at)
		values ($1, $2, 'A', '02', 'OCCUPIED', now(), now())`, unitID2, tenantID)

	// Invoice lunas: total 150000, item IPL 150000.
	mustExec(`insert into invoices (id, tenant_id, house_unit_id, invoice_number, period, base_amount, arrears_amount, total_amount, status, due_date, created_at, updated_at)
		values ($1, $2, $3, 'INV-PAID', '2026-09-01', 150000, 0, 150000, 'PAID', '2026-09-10', now(), now())`,
		invPaid, tenantID, unitID)
	mustExec(`insert into invoice_items (id, invoice_id, label, amount, kind)
		values ($1, $2, 'Iuran IPL', 150000, 'CURRENT')`, itemPaid, invPaid)

	// Invoice belum lunas: total 200000, item IPL 200000.
	mustExec(`insert into invoices (id, tenant_id, house_unit_id, invoice_number, period, base_amount, arrears_amount, total_amount, status, due_date, created_at, updated_at)
		values ($1, $2, $3, 'INV-UNPAID', '2026-09-01', 200000, 0, 200000, 'UNPAID', '2026-09-10', now(), now())`,
		invUnpaid, tenantID, unitID2)
	mustExec(`insert into invoice_items (id, invoice_id, label, amount, kind)
		values ($1, $2, 'Iuran IPL', 200000, 'CURRENT')`, itemUnpaid, invUnpaid)

	mustExec(`insert into payments (id, tenant_id, invoice_id, receipt_number, payment_method, amount_paid, created_at)
		values ($1, $2, $3, 'KW-001', 'MANUAL_CASH', 150000, now())`, payID, tenantID, invPaid)

	if err := seedTx.Commit(ctx); err != nil {
		t.Fatalf("commit seed: %v", err)
	}

	// --- 1) RekapBulanan ---
	rek, err := s.RekapBulanan(ctx, tenantID, "2026-09")
	if err != nil {
		t.Fatalf("RekapBulanan: %v", err)
	}
	if rek.TenantNama != "Test Tenant" {
		t.Errorf("TenantNama = %q, ingin 'Test Tenant'", rek.TenantNama)
	}
	if rek.JumlahInvoice != 2 {
		t.Errorf("JumlahInvoice = %d, ingin 2", rek.JumlahInvoice)
	}
	if rek.TotalDitagih != 350000 {
		t.Errorf("TotalDitagih = %v, ingin 350000", rek.TotalDitagih)
	}
	if rek.TotalDibayar != 150000 {
		t.Errorf("TotalDibayar = %v, ingin 150000", rek.TotalDibayar)
	}
	if rek.TotalTunggakan != 200000 {
		t.Errorf("TotalTunggakan = %v, ingin 200000", rek.TotalTunggakan)
	}
	if rek.JumlahLunas != 1 || rek.JumlahBelum != 1 {
		t.Errorf("lunas/belum = %d/%d, ingin 1/1", rek.JumlahLunas, rek.JumlahBelum)
	}
	if rek.KasTunai != 150000 || rek.KasQRIS != 0 {
		t.Errorf("kas tunai/qris = %v/%v, ingin 150000/0", rek.KasTunai, rek.KasQRIS)
	}
	if len(rek.Rincian) != 1 || rek.Rincian[0].Label != "Iuran IPL" || rek.Rincian[0].Jumlah != 2 || rek.Rincian[0].Amount != 350000 {
		t.Errorf("Rincian tidak sesuai: %+v", rek.Rincian)
	}

	// --- 2) Tunggakan ---
	tun, err := s.Tunggakan(ctx, tenantID)
	if err != nil {
		t.Fatalf("Tunggakan: %v", err)
	}
	if tun.JumlahUnit != 1 || tun.Total != 200000 {
		t.Errorf("Tunggakan unit/total = %d/%v, ingin 1/200000", tun.JumlahUnit, tun.Total)
	}
	if len(tun.Baris) != 1 {
		t.Fatalf("Baris tunggakan = %d, ingin 1", len(tun.Baris))
	}
	b := tun.Baris[0]
	if b.Unit != "A-02" || b.TotalTunggakan != 200000 || b.PeriodeTertua != "2026-09" {
		t.Errorf("Baris tidak sesuai: %+v", b)
	}
	if b.Bucket != "0-30 hari" {
		t.Errorf("Bucket = %q, ingin '0-30 hari'", b.Bucket)
	}

	// --- 3) Pengurus ---
	pg, err := s.Pengurus(ctx, tenantID)
	if err != nil {
		t.Fatalf("Pengurus: %v", err)
	}
	if len(pg) != 1 || pg[0].Nama != "Kepala Test" || pg[0].Phone != "08123456789" || pg[0].Role != "TENANT_MANAGER" {
		t.Errorf("Pengurus tidak sesuai: %+v", pg)
	}
}
