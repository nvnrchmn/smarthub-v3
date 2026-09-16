package usecase

import (
	"testing"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// TestRupiah — format mata uang untuk WA.
func TestRupiah(t *testing.T) {
	cases := map[float64]string{
		0:       "Rp0",
		1000:    "Rp1.000",
		275000:  "Rp275.000",
		1500000: "Rp1.500.000",
	}
	for in, want := range cases {
		got := rupiah(in)
		if got != want {
			t.Errorf("rupiah(%v) = %q, want %q", in, got, want)
		}
	}
}

// TestIsStaff_Penagihan — hanya TENANT_MANAGER/TREASURER yang boleh kelola tagihan.
func TestIsStaff_Penagihan(t *testing.T) {
	b := &Billing{}
	for _, tt := range []struct {
		role    string
		allowed bool
	}{
		{domain.RoleResident, false},
		{domain.RoleSecretary, false},
		{domain.RoleTreasurer, true},
		{domain.RoleTenantManager, true},
	} {
		sub := &domain.SubjectContext{
			TenantID:        "t1",
			AppRoles:        []string{tt.role},
			LifecycleStatus: "ACTIVE",
		}
		got := b.isStaff(sub)
		if got != tt.allowed {
			t.Errorf("isStaff(%s) = %v, want %v", tt.role, got, tt.allowed)
		}
	}
}

// TestIsStaff_RoleKosong — tanpa role = bukan staff.
func TestIsStaff_RoleKosong(t *testing.T) {
	b := &Billing{}
	for _, sub := range []*domain.SubjectContext{
		{TenantID: "t1", AppRoles: []string{}, LifecycleStatus: "ACTIVE"},
		{TenantID: "t1", AppRoles: nil, LifecycleStatus: "ACTIVE"},
	} {
		if b.isStaff(sub) {
			t.Error("tanpa role tidak boleh staff")
		}
	}
}

// TestIsStaff_BukanAktif — akun nonaktif tidak boleh akses.
func TestIsStaff_BukanAktif(t *testing.T) {
	b := &Billing{}
	sub := &domain.SubjectContext{
		TenantID:        "t1",
		AppRoles:        []string{domain.RoleTreasurer},
		LifecycleStatus: "INACTIVE",
	}
	if b.isStaff(sub) {
		t.Error("akun nonaktif tidak boleh akses billing")
	}
}
