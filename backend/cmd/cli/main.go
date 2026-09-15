package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/hub"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/notify"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
)

// smarthub-cli — tugas operasional yang sengaja TIDAK lewat API publik.
// Bootstrap tenant pertama memakai kredensial admin (ADMIN_DATABASE_URL) karena
// pendaftaran mandiri tidak disediakan (invite-only).
func main() {
	cmd := flag.String("cmd", "", "perintah: create-tenant")
	name := flag.String("name", "", "nama perumahan")
	slug := flag.String("slug", "", "slug unik, mis. griya-asri")
	email := flag.String("email", "", "email pengelola pertama")
	fullName := flag.String("full-name", "", "nama lengkap pengelola")
	password := flag.String("password", "", "kata sandi pengelola (min 8 karakter)")
	adminDSN := flag.String("dsn", os.Getenv("ADMIN_DATABASE_URL"), "DSN admin (env ADMIN_DATABASE_URL)")
	periode := flag.String("period", "", "periode tagihan YYYY-MM (kosong = bulan ini)")
	paksa := flag.Bool("force", false, "abaikan tanggal terbit tenant")
	flag.Parse()

	if *cmd != "create-tenant" && *cmd != "generate-invoices" && *cmd != "reconcile-payments" && *cmd != "remind-invoices" && *cmd != "bootstrap-superadmin" {
		log.Fatal("perintah tidak dikenal; pakai -cmd create-tenant | generate-invoices | reconcile-payments | remind-invoices | bootstrap-superadmin")
	}
	// Argumen ini hanya wajib untuk pembuatan tenant, bukan untuk tugas cron.
	if *cmd == "create-tenant" && (*name == "" || *slug == "" || *email == "" || len(*password) < 8 || *adminDSN == "") {
		log.Fatal("wajib: -name -slug -email -password(min 8) dan ADMIN_DATABASE_URL")
	}

	ctx := context.Background()
	if err := db.Connect(ctx, *adminDSN); err != nil {
		log.Fatalf("database admin: %v", err)
	}

	hash, err := security.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	if *fullName == "" {
		*fullName = *email
	}
	store := postgres.New(db.Pool)
	if *cmd == "generate-invoices" {
		generateInvoices(store, *periode, *paksa)
		return
	}
	if *cmd == "reconcile-payments" {
		reconcilePayments(store)
		return
	}
	if *cmd == "remind-invoices" {
		remindInvoices(store)
		return
	}
	if *cmd == "bootstrap-superadmin" {
		if *email == "" || len(*password) < 8 || *fullName == "" {
			log.Fatal("wajib: -email -full-name -password (min 8)")
		}
		hash, err := security.HashPassword(*password)
		if err != nil {
			log.Fatal(err)
		}
		id, err := store.CreateSuperadmin(context.Background(), *email, *fullName, hash)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("superadmin dibuat: %s (%s)\n", *email, id)
		return
	}
	tenantID, userID, err := store.CreateTenantWithManager(ctx, *name, strings.ToLower(*slug), *email, hash, *fullName)
	if err != nil {
		log.Fatalf("gagal membuat tenant: %v", err)
	}
	fmt.Printf("tenant dibuat: %s (%s)\npengelola: %s\n", *name, tenantID, userID)
}

// generateInvoices — dipanggil cron harian: menerbitkan tagihan untuk tenant
// yang tanggal terbitnya jatuh hari ini (atau semua bila -force).
func generateInvoices(store *postgres.Store, periode string, paksa bool) {
	ctx := context.Background()
	daftar, err := store.Tenants(ctx)
	if err != nil {
		log.Fatalf("daftar tenant: %v", err)
	}
	if periode == "" {
		periode = time.Now().Format("2006-01")
	}
	hariIni := time.Now().Day()
	// Aktor cron: peran Bendahara pada tenant yang sedang diproses.
	sub := &domain.SubjectContext{AppRoles: []string{domain.RoleTreasurer}, LifecycleStatus: "ACTIVE"}
	notifier := notify.New()
	for _, t := range daftar {
		// Konteks tenant dipasang agar RLS tetap berlaku untuk operasi tulis.
		hari, _, err := store.BillingSettings(ctx, t.ID)
		if err != nil {
			log.Printf("tenant %s: pengaturan gagal: %v", t.Nama, err)
			continue
		}
		if !paksa && hari != hariIni {
			continue
		}
		sub.TenantID = t.ID
		b := &usecase.Billing{Store: store, Hub: hub.New(), Notify: notifier}
		hasil, err := b.GenerateBulanan(ctx, sub, periode)
		if err != nil {
			log.Printf("tenant %s: %v", t.Nama, err)
			continue
		}
		fmt.Printf("%s periode %s: dibuat=%d dilewati=%d unit=%d wa_ok=%d wa_gagal=%d\n",
			t.Nama, hasil.Periode, hasil.Dibuat, hasil.Dilewati, hasil.Ditagih, hasil.TerkirimWA, hasil.GagalWA)
	}
}

// reconcilePayments — dipanggil cron sering (mis. tiap 5 menit).
//
// Hub pembayaran tidak mengirim webhook untuk QRIS, jadi tagihan hanya lunas
// kalau ada yang menekan "Cek status". Warga yang membayar lalu langsung
// menutup aplikasi meninggalkan tagihannya nyangkut UNPAID. Fungsi ini
// menyusul status ke hub untuk SEMUA tenant; idempoten, aman diulang.
func reconcilePayments(store *postgres.Store) {
	ctx := context.Background()
	daftar, err := store.Tenants(ctx)
	if err != nil {
		log.Fatalf("daftar tenant: %v", err)
	}
	notifier := notify.New()
	total := 0
	for _, t := range daftar {
		// store.WithTenant memasang app.tenant_id per transaksi, jadi tiap
		// tenant hanya membaca datanya sendiri — RLS tetap berlaku.
		b := &usecase.Billing{Store: store, Hub: hub.New(), Notify: notifier}
		hasil, err := b.RekonsiliasiQRIS(ctx, t.ID)
		if err != nil {
			log.Printf("tenant %s: rekonsiliasi gagal: %v", t.Nama, err)
			continue
		}
		// Diam saat tidak ada kandidat, supaya log cron tetap bersih.
		if hasil.Diperiksa == 0 {
			continue
		}
		fmt.Printf("%s: diperiksa=%d lunas=%d dilewati=%d gagal=%d\n",
			t.Nama, hasil.Diperiksa, hasil.Dilunasi, hasil.Dilewati, hasil.Gagal)
		total += hasil.Dilunasi
	}
	if total > 0 {
		fmt.Printf("total tagihan dilunasi otomatis: %d\n", total)
	}
}

// remindInvoices — dipanggil cron (mis. tiap jam): kirim pengingat ke penghuni
// yang tagihannya belum dibayar dan sudah mendekati jatuh tempo.
func remindInvoices(store *postgres.Store) {
	ctx := context.Background()
	daftar, err := store.Tenants(ctx)
	if err != nil {
		log.Fatalf("daftar tenant: %v", err)
	}
	notifier := notify.New()
	total := 0
	for _, t := range daftar {
		b := &usecase.Billing{Store: store, Hub: hub.New(), Notify: notifier}
		hasil, err := b.PengingatTunggakan(ctx, t.ID)
		if err != nil {
			log.Printf("tenant %s: pengingat gagal: %v", t.Nama, err)
			continue
		}
		if hasil.Dikirim == 0 && hasil.Dilewati == 0 {
			continue
		}
		fmt.Printf("%s: dikirim=%d dilewati=%d gagal=%d\n",
			t.Nama, hasil.Dikirim, hasil.Dilewati, hasil.Gagal)
		total += hasil.Dikirim
	}
	if total > 0 {
		fmt.Printf("total pengingat terkirim: %d\n", total)
	}
}
