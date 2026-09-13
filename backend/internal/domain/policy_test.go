package domain

import "testing"

func sub(role, tenant string, units, fcs []string, life string) SubjectContext {
	return SubjectContext{
		AccountID: "u1", TenantID: tenant, AppRoles: []string{role},
		HouseUnitIDs: units, FamilyCardIDs: fcs, LifecycleStatus: life,
	}
}

func TestCanAccessTenantBoundary(t *testing.T) {
	s := sub(RoleTenantManager, "tenant-a", nil, nil, "ACTIVE")
	if CanAccess(s, ResourceContext{TenantID: "tenant-b"}, ActionViewInvoice) {
		t.Fatal("pengelola tenant A tidak boleh mengakses tenant B")
	}
	if CanAccess(sub(RoleTenantManager, "", nil, nil, "ACTIVE"),
		ResourceContext{TenantID: ""}, ActionViewInvoice) {
		t.Fatal("tenant kosong harus ditolak")
	}
}

func TestCanAccessLifecycle(t *testing.T) {
	for _, life := range []string{"PENDING", "SUSPENDED", ""} {
		s := sub(RoleTenantManager, "t", nil, nil, life)
		if CanAccess(s, ResourceContext{TenantID: "t"}, ActionViewInvoice) {
			t.Fatalf("akun berstatus %q tidak boleh dianggap aktif", life)
		}
	}
}

func TestCanAccessPII(t *testing.T) {
	res := ResourceContext{TenantID: "t", FamilyCardID: "kk-1", IsSensitive: true}
	if !CanAccess(sub(RoleSecretary, "t", nil, nil, "ACTIVE"), res, ActionViewPII) {
		t.Fatal("sekretaris berwenang melihat PII")
	}
	if CanAccess(sub(RoleTreasurer, "t", nil, nil, "ACTIVE"), res, ActionViewPII) {
		t.Fatal("bendahara tidak boleh melihat PII warga")
	}
	if CanAccess(sub(RoleResident, "t", nil, nil, "ACTIVE"), res, ActionViewPII) {
		t.Fatal("warga tidak boleh melihat KK orang lain")
	}
	if !CanAccess(sub(RoleResident, "t", nil, []string{"kk-1"}, "ACTIVE"), res, ActionViewPII) {
		t.Fatal("warga boleh melihat KK-nya sendiri")
	}
}

func TestCanAccessInvoice(t *testing.T) {
	res := ResourceContext{TenantID: "t", HouseUnitID: "unit-7"}
	if CanAccess(sub(RoleResident, "t", []string{"unit-8"}, nil, "ACTIVE"), res, ActionViewInvoice) {
		t.Fatal("warga tidak boleh melihat tagihan unit lain")
	}
	if !CanAccess(sub(RoleResident, "t", []string{"unit-7"}, nil, "ACTIVE"), res, ActionViewInvoice) {
		t.Fatal("warga boleh melihat tagihan unitnya")
	}
	if !CanAccess(sub(RoleTreasurer, "t", nil, nil, "ACTIVE"), res, ActionViewInvoice) {
		t.Fatal("bendahara boleh melihat tagihan semua unit")
	}
	if CanAccess(sub(RoleTreasurer, "t", nil, nil, "ACTIVE"), res, ActionPayInvoice) {
		t.Fatal("bayar tagihan hanya untuk warga (kas dicatat lewat aksi lain)")
	}
	if !CanAccess(sub(RoleResident, "t", []string{"unit-7"}, nil, "ACTIVE"), res, ActionPayInvoice) {
		t.Fatal("warga boleh membayar tagihan unitnya")
	}
}

func TestCanAccessDenyByDefault(t *testing.T) {
	s := sub("ROLE_ASING", "t", []string{"unit-7"}, []string{"kk-1"}, "ACTIVE")
	if CanAccess(s, ResourceContext{TenantID: "t", HouseUnitID: "unit-7"}, ActionViewInvoice) {
		t.Fatal("peran tak dikenal harus ditolak")
	}
	if CanAccess(s, ResourceContext{TenantID: "t"}, Action("AKSI_TIDAK_ADA")) {
		t.Fatal("aksi tak dikenal harus ditolak")
	}
	if !CanAccess(sub(RoleTenantManager, "t", nil, nil, "ACTIVE"), ResourceContext{TenantID: "t"}, ActionManageUser) {
		t.Fatal("pengelola berwenang mengelola anggota")
	}
}
