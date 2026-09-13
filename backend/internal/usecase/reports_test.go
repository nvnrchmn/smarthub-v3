package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// Warga (RESIDENT) tidak boleh membuka laporan; pengurus boleh.
func TestGerbangLaporan(t *testing.T) {
	r := &Reports{}
	warga := &domain.SubjectContext{AccountID: "u1", TenantID: "t1", AppRoles: []string{domain.RoleResident}, LifecycleStatus: "ACTIVE"}
	if err := r.gerbang(warga); err == nil {
		t.Fatal("warga seharusnya ditolak membuka laporan")
	}
	for _, peran := range []string{domain.RoleTreasurer, domain.RoleTenantManager} {
		sub := &domain.SubjectContext{AccountID: "u2", TenantID: "t1", AppRoles: []string{peran}, LifecycleStatus: "ACTIVE"}
		if err := r.gerbang(sub); err != nil {
			t.Errorf("%s seharusnya lolos gerbang laporan: %v", peran, err)
		}
	}
	// Sekretaris mengurus administrasi (sensus/undangan), bukan uang.
	sekretaris := &domain.SubjectContext{AccountID: "u3", TenantID: "t1", AppRoles: []string{domain.RoleSecretary}, LifecycleStatus: "ACTIVE"}
	if err := r.gerbang(sekretaris); err == nil {
		t.Error("sekretaris seharusnya tidak membuka laporan keuangan")
	}
}

// Periode yang tidak berbentuk YYYY-MM ditolak sebelum menyentuh database.
func TestPeriodeTidakValid(t *testing.T) {
	r := &Reports{}
	sub := &domain.SubjectContext{AccountID: "u2", TenantID: "t1", AppRoles: []string{domain.RoleTenantManager}, LifecycleStatus: "ACTIVE"}
	if _, err := r.Rekap(context.Background(), sub, "2026-09-13"); err == nil {
		t.Fatal("periode panjang seharusnya ditolak")
	}
}

// CSV: ada BOM (Excel membaca UTF-8), pemisah ';', dan angka tunggakan ikut tertulis.
func TestCSVLaporan(t *testing.T) {
	r := &Reports{}
	rek := &domain.RekapBulanan{TenantNama: "Griya Asri", Periode: "2026-09", JumlahInvoice: 2,
		TotalDitagih: 550000, TotalDibayar: 275000, TotalTunggakan: 275000, JumlahLunas: 1, JumlahBelum: 1}
	tun := &domain.Tunggakan{JumlahUnit: 1, Total: 275000, Baris: []domain.TunggakanBaris{
		{Unit: "A-01", KepalaKeluarga: "Budi", PeriodeTertua: "2026-09", JumlahTagihan: 1, TotalTunggakan: 275000, HariTerlambat: 3, Bucket: "0-30 hari"},
	}}
	out := string(r.CSV(rek, tun))
	if !strings.HasPrefix(out, "\ufeff") {
		t.Error("CSV harus berawalan BOM")
	}
	for _, wajib := range []string{"Griya Asri", "2026-09", "A-01", "275.000", "0-30 hari"} {
		if !strings.Contains(out, wajib) {
			t.Errorf("CSV tidak memuat %q", wajib)
		}
	}
}
