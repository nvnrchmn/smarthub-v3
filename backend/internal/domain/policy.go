package domain

// Peran aplikasi (RBAC baseline + ABAC granular).
const (
	RoleResident      = "RESIDENT"
	RoleSecretary     = "SECRETARY"
	RoleTreasurer     = "TREASURER"
	RoleTenantManager = "TENANT_MANAGER"
)

type Action string

const (
	ActionViewInvoice Action = "VIEW_INVOICE"
	ActionPayInvoice  Action = "PAY_INVOICE"
	ActionViewPII     Action = "VIEW_PII"
	ActionVerifyPII   Action = "VERIFY_PII"
	ActionManageUser  Action = "MANAGE_USER"
	ActionManageBilling Action = "MANAGE_BILLING"
	ActionEditProfile Action = "EDIT_PROFILE"
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
	// 1. Batas tenant: ditegakkan lebih dulu, apa pun perannya.
	if sub.TenantID == "" || sub.TenantID != res.TenantID {
		return false
	}
	// 2. Siklus hidup akun.
	if sub.LifecycleStatus != "ACTIVE" {
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
