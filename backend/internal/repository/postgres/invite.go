package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// LookupInviteByToken — aktivasi undangan; lewat fungsi SECURITY DEFINER karena
// pemegang tautan belum punya konteks tenant.
func (s *Store) LookupInviteByToken(ctx context.Context, tokenHash string) (*domain.Invite, error) {
	var inv domain.Invite
	err := s.Pool.QueryRow(ctx,
		`select id, tenant_id, email, coalesce(phone,''), role, coalesce(otp_hash,''),
		        expires_at, used_at, attempts, otp_sent_at
		 from auth_lookup_invite($1)`, tokenHash,
	).Scan(&inv.ID, &inv.TenantID, &inv.Email, &inv.Phone, &inv.Role, &inv.OTPHash,
		&inv.ExpiresAt, &inv.UsedAt, &inv.Attempts, &inv.OTPSentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.TokenHash = tokenHash
	return &inv, nil
}

func (s *Store) CreateInvite(ctx context.Context, inv domain.Invite, otpHash string) error {
	return s.WithTenant(ctx, inv.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`insert into invite_tokens (tenant_id, email, phone, role, otp_hash, token_hash, expires_at, otp_sent_at)
			 values ($1, lower($2), $3, $4, $5, $6, $7, now())`,
			inv.TenantID, inv.Email, inv.Phone, inv.Role, otpHash, inv.TokenHash, inv.ExpiresAt)
		return err
	})
}

func (s *Store) UpdateInviteOTP(ctx context.Context, tenantID, inviteID, otpHash string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`update invite_tokens set otp_hash = $2, otp_sent_at = now(), attempts = 0
			 where id = $1`, inviteID, otpHash)
		return err
	})
}

func (s *Store) BumpInviteAttempts(ctx context.Context, tenantID, inviteID string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			"update invite_tokens set attempts = attempts + 1 where id = $1", inviteID)
		return err
	})
}

// ActivateInvite — tandai undangan terpakai dan buat akun ACTIVE dalam satu transaksi.
func (s *Store) ActivateInvite(ctx context.Context, inv *domain.Invite, fullName, passwordHash string) (string, error) {
	var userID string
	err := s.WithTenant(ctx, inv.TenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`insert into users (tenant_id, email, phone, full_name, password_hash, role, status, activated_at)
			 values ($1, lower($2), $3, $4, $5, $6, 'ACTIVE', now()) returning id`,
			inv.TenantID, inv.Email, inv.Phone, fullName, passwordHash, inv.Role,
		).Scan(&userID); err != nil {
			if isUniqueViolation(err) {
				return ErrDuplicate
			}
			return err
		}
		_, err := tx.Exec(ctx, "update invite_tokens set used_at = now() where id = $1", inv.ID)
		return err
	})
	return userID, err
}

// UserByID — dipakai middleware untuk memastikan status akun masih ACTIVE.
func (s *Store) UserByID(ctx context.Context, tenantID, userID string) (*domain.User, error) {
	var u domain.User
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`select id, tenant_id, email, coalesce(phone,''), full_name, role, status
			 from users where id = $1`, userID,
		).Scan(&u.ID, &u.TenantID, &u.Email, &u.Phone, &u.FullName, &u.Role, &u.Status)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

var _ = time.Now


// ErrDuplicate — dipakai agar pemanggil bisa membalas 409, bukan 500, saat
// email/telepon sudah terdaftar (PostgreSQL 23505).
var ErrDuplicate = errors.New("duplikat")

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
