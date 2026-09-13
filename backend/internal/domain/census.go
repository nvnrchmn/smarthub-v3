package domain

import "time"

// ResidentProfile — data sensus warga. NIK & nomor KK tidak pernah dikirim
// dalam bentuk asli ke luar; daftar memakai nik_last4 (tersamar).
type ResidentProfile struct {
	ID                 string    `json:"id"`
	AccountID          string    `json:"account_id,omitempty"`
	FamilyCardID       string    `json:"family_card_id,omitempty"`
	FullName           string    `json:"full_name"`
	NIKLast4           string    `json:"nik_last4,omitempty"`
	FamilyRole         string    `json:"family_role"`
	BirthPlace         string    `json:"birth_place,omitempty"`
	BirthDate          string    `json:"birth_date,omitempty"`
	Gender             string    `json:"gender,omitempty"`
	Religion           string    `json:"religion,omitempty"`
	MaritalStatus      string    `json:"marital_status,omitempty"`
	Occupation         string    `json:"occupation,omitempty"`
	Education          string    `json:"education,omitempty"`
	VerificationStatus string    `json:"verification_status"`
	RejectionReason    string    `json:"rejection_reason,omitempty"`
	LifecycleStatus    string    `json:"lifecycle_status"`
	HasKTP             bool      `json:"has_ktp"`
	HasKK              bool      `json:"has_kk"`
	HouseUnit          string    `json:"house_unit,omitempty"`
	HouseUnitID        string    `json:"house_unit_id,omitempty"`
	OccupancyType      string    `json:"occupancy_type,omitempty"`
	IsPrimaryPayer     bool      `json:"is_primary_payer"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// HouseUnit — rumah/unit di lingkungan tenant.
type HouseUnit struct {
	ID              string `json:"id"`
	Block           string `json:"block"`
	UnitNumber      string `json:"unit_number"`
	OccupancyStatus string `json:"occupancy_status"`
	Notes           string `json:"notes,omitempty"`
	OccupantCount   int    `json:"occupant_count"`
	PrimaryOccupant string `json:"primary_occupant,omitempty"`
}

// OccupancyInput — penempatan warga pada sebuah unit (dipakai Sekretaris).
type OccupancyInput struct {
	ResidentID     string `json:"resident_id"`
	HouseUnitID    string `json:"house_unit_id"`
	OccupancyType  string `json:"occupancy_type"`
	IsPrimaryPayer bool   `json:"is_primary_payer"`
}

// Peran dalam keluarga (ERD: family_role).
const (
	FamilyRoleHead   = "HEAD_OF_FAMILY"
	FamilyRoleSpouse = "SPOUSE"
	FamilyRoleChild  = "CHILD"
	FamilyRoleOther  = "OTHER"
)

// Status verifikasi dokumen (ERD: verification_status).
const (
	VerifyUnverified = "UNVERIFIED"
	VerifyVerified   = "VERIFIED"
	VerifyRejected   = "REJECTED"
)

// Status siklus hidup warga (ERD: lifecycle_status).
const (
	LifeActive   = "ACTIVE"
	LifeMovedOut = "MOVED_OUT"
	LifeDeceased = "DECEASED"
)
// FamilyCard — Kartu Keluarga. Nomor KK disimpan terenkripsi; daftar memakai
// number_last4 (4 digit terakhir) agar tidak perlu dekripsi untuk menampilkan.
type FamilyCard struct {
	ID          string             `json:"id"`
	NumberLast4 string             `json:"number_last4"`
	HasFile     bool               `json:"has_file"`
	MemberCount int                `json:"member_count"`
	Members     []FamilyCardMember `json:"members"`
	CreatedAt   time.Time          `json:"created_at"`
}

type FamilyCardMember struct {
	ID                 string `json:"id"`
	FullName           string `json:"full_name"`
	FamilyRole         string `json:"family_role"`
	VerificationStatus string `json:"verification_status"`
	LifecycleStatus    string `json:"lifecycle_status"`
}

// Status hunian rumah (house_units.occupancy_status).
const (
	UnitOccupied   = "OCCUPIED"
	UnitVacant     = "VACANT"
	UnitRenovation = "RENOVATION"
)

// Jenis hunian (house_occupancies.occupancy_type).
const (
	OccupancyOwnerOccupant   = "OWNER_OCCUPANT"
	OccupancyTenant          = "TENANT"
	OccupancyOwnerNonResident = "OWNER_NON_RESIDENT"
)
