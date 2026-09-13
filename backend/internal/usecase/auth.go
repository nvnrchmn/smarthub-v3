package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/notify"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

type Notifier interface {
	SendOTP(phone, code, tenantName string) error
	SendInviteLink(phone, link, tenantName string) error
}

type Auth struct {
	Store     *postgres.Store
	Notify    Notifier
	JWTSecret string
	BaseURL   string // dasar tautan aktivasi, mis. https://smarthub.logikraf.id
}

var (
	ErrInvalidInput = errors.New("data tidak valid")
	ErrConflict     = errors.New("sudah terdaftar")
	ErrExpired      = errors.New("undangan kedaluwarsa")
	ErrUsed         = errors.New("undangan sudah dipakai")
	ErrAttempts     = errors.New("terlalu banyak percobaan")
	ErrCredentials  = errors.New("email atau kata sandi salah")
	ErrInactive     = errors.New("akun belum aktif")
)

var validRoles = map[string]bool{
	domain.RoleResident: true, domain.RoleSecretary: true,
	domain.RoleTreasurer: true, domain.RoleTenantManager: true,
}

// CreateInvite — pengelola/sekretaris mengundang warga atau pengurus.
// Mengembalikan tautan aktivasi (berlaku 7 hari) dan mengirim OTP ke nomor tujuan.
// CreateInvite — membuat undangan lalu mengirim tautan + OTP ke nomor warga.
// Bila pengiriman WhatsApp gagal, undangan tetap dibuat dan tautan dikembalikan
// supaya pengurus bisa meneruskannya sendiri (terkirim=false).
func (a *Auth) CreateInvite(ctx context.Context, tenantID, email, phone, role, tenantName string) (link string, terkirim bool, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.ToUpper(strings.TrimSpace(role))
	if email == "" || !strings.Contains(email, "@") {
		return "", false, fmt.Errorf("%w: email tidak valid", ErrBadInput)
	}
	if !validRoles[role] {
		return "", false, fmt.Errorf("%w: peran tidak dikenal", ErrBadInput)
	}
	nomor := notify.NormalisasiNomor(phone)
	if len(nomor) < 10 || len(nomor) > 15 || !strings.HasPrefix(nomor, "62") {
		return "", false, fmt.Errorf("%w: nomor HP tidak valid, contoh 08123456789", ErrBadInput)
	}

	plain, tokenHash, err := security.NewToken()
	if err != nil {
		return "", false, err
	}
	code, otpHash, err := security.NewOTP()
	if err != nil {
		return "", false, err
	}
	inv := domain.Invite{
		TenantID:  tenantID,
		Email:     email,
		Phone:     nomor,
		Role:      role,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(domain.InviteLinkTTL),
	}
	if err := a.Store.CreateInvite(ctx, inv, otpHash); err != nil {
		return "", false, err
	}

	link = strings.TrimRight(a.BaseURL, "/") + "/aktivasi?token=" + plain
	// Undangan dikirim ke nomor warga; kalau belum ada kanal, kode masuk log server.
	errLink := a.Notify.SendInviteLink(nomor, link, tenantName)
	errOTP := a.Notify.SendOTP(nomor, code, tenantName)
	if errLink != nil || errOTP != nil {
		// Jangan gagalkan pembuatan undangan: tautan masih bisa diteruskan manual.
		gagal := errOTP
		if gagal == nil {
			gagal = errLink
		}
		log.Printf("[undangan] pengiriman WhatsApp gagal untuk %s: %v", email, gagal)
		return link, false, nil
	}
	return link, true, nil
}

// ResendOTP — kirim ulang kode untuk undangan yang belum dipakai.
func (a *Auth) ResendOTP(ctx context.Context, tokenPlain, tenantName string) error {
	inv, err := a.Store.LookupInviteByToken(ctx, security.HashToken(strings.TrimSpace(tokenPlain)))
	if err != nil {
		return ErrInvalidInput
	}
	if inv.UsedAt != nil {
		return ErrUsed
	}
	if time.Now().After(inv.ExpiresAt) {
		return ErrExpired
	}
	code, otpHash, err := security.NewOTP()
	if err != nil {
		return err
	}
	if err := a.Store.UpdateInviteOTP(ctx, inv.TenantID, inv.ID, otpHash); err != nil {
		return err
	}
	return a.Notify.SendOTP(inv.Phone, code, tenantName)
}

// AcceptInvite — aktivasi: validasi OTP, set password, akun menjadi ACTIVE.
func (a *Auth) AcceptInvite(ctx context.Context, tokenPlain, otp, fullName, password, tenantName string) (string, *domain.User, error) {
	if len(password) < 8 {
		return "", nil, ErrInvalidInput
	}
	inv, err := a.Store.LookupInviteByToken(ctx, security.HashToken(strings.TrimSpace(tokenPlain)))
	if err != nil {
		return "", nil, ErrInvalidInput
	}
	if inv.UsedAt != nil {
		return "", nil, ErrUsed
	}
	if time.Now().After(inv.ExpiresAt) {
		return "", nil, ErrExpired
	}
	if inv.OTPHash == "" || inv.OTPSentAt == nil || time.Since(*inv.OTPSentAt) > domain.OTPTTL {
		return "", nil, ErrExpired
	}
	if inv.Attempts >= domain.OTPMaxAttempts {
		return "", nil, ErrAttempts
	}
	if !security.CheckOTP(strings.TrimSpace(otp), inv.OTPHash) {
		_ = a.Store.BumpInviteAttempts(ctx, inv.TenantID, inv.ID)
		return "", nil, ErrCredentials
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(fullName) == "" {
		fullName = inv.Email
	}
	userID, err := a.Store.ActivateInvite(ctx, inv, fullName, hash)
	if errors.Is(err, postgres.ErrDuplicate) {
		// email sudah punya akun: balas 409 (bukan 500) dengan pesan jelas
		return "", nil, ErrConflict
	}
	if err != nil {
		return "", nil, err
	}
	token, err := security.GenerateToken(userID, inv.TenantID, inv.Role, a.JWTSecret)
	if err != nil {
		return "", nil, err
	}
	return token, &domain.User{ID: userID, TenantID: inv.TenantID, Email: inv.Email,
		FullName: fullName, Role: inv.Role, Status: "ACTIVE"}, nil
}

// Login — verifikasi kredensial dan terbitkan token berisi tenant + peran.
func (a *Auth) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	u, err := a.Store.LookupUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return "", nil, ErrCredentials
	}
	if !security.CheckPassword(password, u.PasswordHash) {
		return "", nil, ErrCredentials
	}
	if u.Status != "ACTIVE" {
		return "", nil, ErrInactive
	}
	token, err := security.GenerateToken(u.ID, u.TenantID, u.Role, a.JWTSecret)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// SubjectFromToken — dipakai middleware untuk membangun konteks ABAC setelah
// memastikan akunnya masih ACTIVE di database (token yang dicabut tidak berlaku).
func (a *Auth) SubjectFromToken(ctx context.Context, userID, tenantID string) (*domain.SubjectContext, error) {
	u, err := a.Store.UserByID(ctx, tenantID, userID)
	if err != nil {
		return nil, ErrCredentials
	}
	if u.Status != "ACTIVE" {
		return nil, ErrInactive
	}
	// ABAC butuh tahu apa yang dimiliki warga (unit & kartu keluarga).
	units, familyCards, err := a.Store.OwnershipOf(ctx, u.TenantID, u.ID)
	if err != nil {
		units, familyCards = nil, nil
	}
	return &domain.SubjectContext{
		AccountID:       u.ID,
		TenantID:        u.TenantID,
		AppRoles:        []string{u.Role},
		HouseUnitIDs:    units,
		FamilyCardIDs:   familyCards,
		LifecycleStatus: u.Status,
	}, nil
}
