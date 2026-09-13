package usecase

import (
	"context"
	"strings"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

type Superadmin struct {
	Store *postgres.Store
}

func (s *Superadmin) Login(ctx context.Context, email, password string) (*domain.Superadmin, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		// Sengaja pakai ErrCredentials (401), bukan 400: login tidak boleh
		// membocorkan aturan format kata sandi lewat kode status.
		return nil, ErrCredentials
	}
	sa, err := s.Store.SuperadminByEmail(ctx, email)
	if err != nil {
		return nil, ErrCredentials
	}
	if sa.Status != "ACTIVE" {
		return nil, ErrInactive
	}
	if !security.CheckPassword(password, sa.PasswordHash) {
		return nil, ErrCredentials
	}
	_ = s.Store.SetLastLogin(ctx, sa.ID)
	return sa, nil
}

func (s *Superadmin) ResetPassword(ctx context.Context, id, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrInvalidInput
	}
	sa, err := s.Store.SuperadminByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if !security.CheckPassword(oldPassword, sa.PasswordHash) {
		return ErrCredentials
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.Store.ResetSuperadminPassword(ctx, sa.ID, hash)
}

func (s *Superadmin) Tenants(ctx context.Context) ([]domain.Tenant, error) {
	return s.Store.AllTenants(ctx)
}

func (s *Superadmin) AuditLog(ctx context.Context) ([]domain.AuditLog, error) {
	return s.Store.AuditLogGlobal(ctx, 100)
}

func (s *Superadmin) GetSetting(ctx context.Context, key string) (string, error) {
	return s.Store.GetSetting(ctx, key)
}

// AllSettings — semua pengaturan global sekaligus (untuk halaman Pengaturan).
func (s *Superadmin) AllSettings(ctx context.Context) (map[string]string, error) {
	return s.Store.AllSettings(ctx)
}

func (s *Superadmin) SetSetting(ctx context.Context, key, value string) error {
	return s.Store.SetSetting(ctx, key, value)
}

func (s *Superadmin) ByID(ctx context.Context, id string) (*domain.Superadmin, error) {
	return s.Store.SuperadminByID(ctx, id)
}
