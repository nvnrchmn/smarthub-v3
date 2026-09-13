package domain

import "time"

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"tenant_id"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone,omitempty"`
	FullName     string     `json:"full_name"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Invite — undangan berbasis tautan (berlaku 7 hari) + OTP untuk memvalidasi
// nomor telepon saat aktivasi (sesuai PRD alur invite-only).
type Invite struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Email      string     `json:"email"`
	Phone      string     `json:"phone"`
	Role       string     `json:"role"`
	OTPHash    string     `json:"-"`
	TokenHash  string     `json:"-"`
	ExpiresAt  time.Time  `json:"expires_at"`
	OTPSentAt  *time.Time `json:"otp_sent_at,omitempty"`
	Attempts   int        `json:"attempts"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

const (
	InviteLinkTTL = 7 * 24 * time.Hour
	OTPTTL        = 10 * time.Minute
	OTPMaxAttempts = 5
)
