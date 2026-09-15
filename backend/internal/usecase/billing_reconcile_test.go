package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/hub"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

// notifierDiam — pengganti notifier WhatsApp supaya tes tidak menyentuh jaringan.
type notifierDiam struct{}

func (notifierDiam) SendOTP(string, string, string) error        { return nil }
func (notifierDiam) SendInviteLink(string, string, string) error { return nil }
func (n notifierDiam) SendLupaSandi(string, string, string) error    { return nil }
func (n notifierDiam) SendTagihan(string, string, string) error    { return nil }
func (n notifierDiam) SendPengingatTunggakan(string, string, string) error { return nil }


// TestRekonsiliasiQRIS — inti perbaikan: tagihan yang SUDAH dibayar warga harus
// lunas otomatis dari cron, tanpa ada yang menekan "Cek status".
func TestRekonsiliasiQRIS(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL kosong")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("koneksi: %v", err)
	}
	defer pool.Close()

	// Hub palsu: jawaban diatur PER REFERENSI supaya tiap invoice bisa mewakili
	// skenario berbeda (lunas / masih tertunda / nominal tidak cocok).
	statusRef := map[string]string{"REF-UJI-1": "paid", "REF-UJI-2": "pending", "REF-UJI-3": "paid"}
	amountRef := map[string]float64{"REF-UJI-1": 200000, "REF-UJI-2": 200000, "REF-UJI-3": 5000}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref := path.Base(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"reference_id": ref, "status": statusRef[ref], "amount": amountRef[ref],
		})
	}))
	defer srv.Close()
	t.Setenv("HUB_BASE_URL", srv.URL)
	t.Setenv("HUB_INTERNAL_KEY", "kunci-uji")
	t.Setenv("HUB_EXT_PREFIX", "sb-")

	store := postgres.New(pool)
	b := &Billing{Store: store, Hub: hub.New(), Notify: notifierDiam{}}

	// Seed: 1 tenant, 1 unit, 3 invoice ber-QRIS. Dibungkus transaksi karena
	// app.tenant_id bersifat is_local (RLS) dan tidak bertahan di autocommit.
	tenantID, unitID := uuid.NewString(), uuid.NewString()
	invLunas, invTertunda, invNominal := uuid.NewString(), uuid.NewString(), uuid.NewString()
	// Bersihkan data uji di akhir. Tiap DELETE punya transaksi sendiri supaya
	// satu kegagalan (mis. tabel tanpa kolom tenant_id) tidak membatalkan sisanya.
	defer func() {
		cx := context.Background()
		bersih := func(q string) {
			ct, err := pool.Begin(cx)
			if err != nil {
				return
			}
			defer ct.Rollback(cx)
			_, _ = ct.Exec(cx, `select set_config('app.tenant_id', $1, true)`, tenantID)
			_, _ = ct.Exec(cx, q, tenantID)
			_ = ct.Commit(cx)
		}
		bersih(`delete from payments where tenant_id=$1`)
		bersih(`delete from invoices where tenant_id=$1`)
		bersih(`delete from house_units where tenant_id=$1`)
		bersih(`delete from tenants where id=$1`)
	}()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	// Wajib: bila seed gagal di tengah, transaksi harus melepas koneksinya.
	// Tanpa ini pool.Close() menggantung selamanya menunggu koneksi yang dipegang.
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
		t.Fatalf("set tenant: %v", err)
	}
	ins := func(q string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	ins(`insert into tenants (id, name, slug) values ($1,'Uji Rekonsiliasi',$2)`, tenantID, "uji-rekon-"+tenantID[:8])
	ins(`insert into house_units (id, tenant_id, block, unit_number) values ($1,$2,'A','01')`, unitID, tenantID)
	qInv := `insert into invoices (id, tenant_id, house_unit_id, invoice_number, period, due_date,
		status, base_amount, arrears_amount, total_amount, qris_reference)
		values ($1,$2,$3,$4,$5::date,$5::date + 14,'UNPAID',200000,0,200000,$6)`
	// Periode harus berbeda-beda: ada unique (tenant_id, house_unit_id, period).
	// invoice_number unik GLOBAL (bukan per tenant) -> beri akhiran unik per run.
	sfx := tenantID[:8]
	ins(qInv, invLunas, tenantID, unitID, "INV-UJI-1-"+sfx, "2026-09-01", "REF-UJI-1")
	ins(qInv, invTertunda, tenantID, unitID, "INV-UJI-2-"+sfx, "2026-10-01", "REF-UJI-2")
	ins(qInv, invNominal, tenantID, unitID, "INV-UJI-3-"+sfx, "2026-11-01", "REF-UJI-3")
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// --- Satu putaran rekonsiliasi melayani tiga skenario sekaligus:
	// REF-UJI-1 lunas -> ditandai PAID; REF-UJI-2 masih pending -> tetap UNPAID;
	// REF-UJI-3 nominal tidak cocok -> tetap UNPAID (pengaman uang).
	hasil, err := b.RekonsiliasiQRIS(ctx, tenantID)
	if err != nil {
		t.Fatalf("rekonsiliasi: %v", err)
	}
	if hasil.Diperiksa != 3 || hasil.Dilunasi != 1 {
		t.Fatalf("harus diperiksa=3 dan dilunasi=1, dapat %+v", hasil)
	}
	inv1, _ := store.InvoiceByID(ctx, tenantID, invLunas)
	if inv1 == nil || inv1.Status != "PAID" {
		t.Fatalf("INV-UJI-1 harus PAID, dapat %+v", inv1)
	}
	inv2, _ := store.InvoiceByID(ctx, tenantID, invTertunda)
	if inv2 == nil || inv2.Status != "UNPAID" {
		t.Fatalf("INV-UJI-2 harus tetap UNPAID, dapat %+v", inv2)
	}

	inv3, _ := store.InvoiceByID(ctx, tenantID, invNominal)
	if inv3 == nil || inv3.Status != "UNPAID" {
		t.Fatalf("INV-UJI-3 nominal tidak cocok harus tetap UNPAID, dapat %+v", inv3)
	}

	// --- Idempoten: putaran kedua tidak boleh melunasi apa pun lagi
	// (INV-UJI-1 sudah PAID sehingga tidak lagi jadi kandidat).
	hasil2, err := b.RekonsiliasiQRIS(ctx, tenantID)
	if err != nil {
		t.Fatalf("rekonsiliasi kedua: %v", err)
	}
	if hasil2.Dilunasi != 0 {
		t.Fatalf("putaran kedua tidak boleh melunasi lagi, dapat %+v", hasil2)
	}
}
