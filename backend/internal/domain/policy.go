package domain

// Peran aplikasi (RBAC baseline + ABAC granular).
const (
	RoleResident      = "RESIDENT"
	RoleSecretary     = "SECRETARY"
	RoleTreasurer     = "TREASURER"
	RoleTenantManager = "TENANT_MANAGER"
	RoleSuperadmin    = "SUPERADMIN"
)

type Action string

const (
	ActionViewInvoice     Action = "VIEW_INVOICE"
	ActionPayInvoice      Action = "PAY_INVOICE"
	ActionViewPII         Action = "VIEW_PII"
	ActionVerifyPII       Action = "VERIFY_PII"
	ActionManageUser      Action = "MANAGE_USER"
	ActionManageBilling   Action = "MANAGE_BILLING"
	ActionEditProfile     Action = "EDIT_PROFILE"
	ActionManagePlatform  Action = "MANAGE_PLATFORM"
	ActionViewAuditLog    Action = "VIEW_AUDIT_LOG"
)

// SubjectContext — siapa yang meminta akses.
type SubjectContext struct {
	AccountID       string
	TenantID        string
	AppRoles        []string
	HouseUnitIDs    []string
	FamilyCardIDs   []string
	LifecycleStatus string
}

// ResourceContext — apa yang diminta.
type ResourceContext struct {
	TenantID     string
	HouseUnitID  string
	FamilyCardID string
	IsSensitive  bool
}

func hasRole(sub SubjectContext, roles ...string) bool {
	for _, r := range sub.AppRoles {
		for _, want := range roles {
			if r == want {
				return true
			}
		}
	}
	return false
}

func ownsUnit(sub SubjectContext, unitID string) bool {
	if unitID == "" {
		return false
	}
	for _, id := range sub.HouseUnitIDs {
		if id == unitID {
			return true
		}
	}
	return false
}

func ownsFamilyCard(sub SubjectContext, fcID string) bool {
	if fcID == "" {
		return false
	}
	for _, id := range sub.FamilyCardIDs {
		if id == fcID {
			return true
		}
	}
	return false
}

// CanAccess — evaluator ABAC. Default MENOLAK: setiap aksi harus punya alasan
// eksplisit untuk diizinkan. Akun yang belum ACTIVE tidak boleh apa pun.
func CanAccess(sub SubjectContext, res ResourceContext, act Action) bool {
	// 1. Siklus hidup akun (sebelum pengecualian superadmin, agar akun suspended
	//    tidak bisa apa pun).
	if sub.LifecycleStatus != "ACTIVE" {
		return false
	}

	// 2. SUPERADMIN: akses global untuk manajemen platform & audit log, TIDAK untuk
	//    PII kependudukan privat (sesuai PRD: "Tidak memiliki visibilitas ke data
	//    kependudukan privat").
	if hasRole(sub, RoleSuperadmin) {
		switch act {
		case ActionViewAuditLog, ActionManagePlatform:
			return true
		}
		return false
	}

	// 3. Batas tenant: ditegakkan untuk peran selain superadmin.
	if sub.TenantID == "" || sub.TenantID != res.TenantID {
		return false
	}

	switch act {
	case ActionViewPII, ActionVerifyPII:
		// Data NIK/KK: hanya pengurus berwenang, atau warga atas KK-nya sendiri.
		if hasRole(sub, RoleSecretary, RoleTenantManager) {
			return true
		}
		// Warga hanya untuk KK-nya sendiri; peran lain tidak dapat hak ini.
		if act == ActionViewPII && hasRole(sub, RoleResident) && ownsFamilyCard(sub, res.FamilyCardID) {
			return true
		}
		return false

	case ActionViewInvoice:
		if hasRole(sub, RoleTreasurer, RoleTenantManager, RoleSecretary) {
			return true
		}
		return hasRole(sub, RoleResident) && ownsUnit(sub, res.HouseUnitID)

	case ActionPayInvoice:
		// Hanya warga pemilik unit; bendahara mencatat kas lewat aksi berbeda.
		return hasRole(sub, RoleResident) && ownsUnit(sub, res.HouseUnitID)

	case ActionManageUser:
		return hasRole(sub, RoleTenantManager)
	case ActionManageBilling:
		// Bendahara pemegang kewenangan finansial; Ketua ikut mengawasi.
		return hasRole(sub, RoleTreasurer, RoleTenantManager)
	case ActionEditProfile:
		// Sekretaris/pengelola mengurus sensus; warga hanya profilnya sendiri.
		if hasRole(sub, RoleSecretary, RoleTenantManager) {
			return true
		}
		return hasRole(sub, RoleResident) && ownsFamilyCard(sub, res.FamilyCardID)
	}

	return false
}
