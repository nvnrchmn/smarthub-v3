package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/jackc/pgx/v5"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "fallback-token"
	}
	return hex.EncodeToString(b)
}

func hashToken(token string) string {
	b := sha256.Sum256([]byte(token))
	return hex.EncodeToString(b[:])
}

// BuatTokenReset — membuat token acak, menyimpan hash-nya, dan return token plaintext yang dikirim ke pengguna.
func (s *Store) BuatTokenReset(ctx context.Context, userID string) (string, error) {
	plaintext := generateRandomToken()
	hash := hashToken(plaintext)
	tag, err := s.Pool.Exec(ctx, `update users
		set reset_token_hash = $2,
		    reset_sent_at = now()
		where id = $1 and status = 'ACTIVE'`, userID, hash)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return plaintext, nil
}

// AmbilUserUntukReset — mencari user berdasarkan token reset (plaintext).
// Token di-hash dulu sebelum dibandingkan dengan hash yang tersimpan.
// expired=true bila sudah lebih dari 1 jam.
func (s *Store) AmbilUserUntukReset(ctx context.Context, token string) (string, bool, error) {
	hash := hashToken(token)
	var userID string
	var expired bool
	err := s.Pool.QueryRow(ctx, `
		select id,
		       reset_sent_at < now() - interval '1 hour'
		from users
		where reset_token_hash = $1
		  and status = 'ACTIVE'`, hash).Scan(&userID, &expired)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	return userID, expired, err
}

// SetelPasswordBaru — mengganti password dan menghapus token reset agar
// tidak bisa dipakai lagi (sekali pakai).
func (s *Store) SetelPasswordBaru(ctx context.Context, userID, passwordBaruHash string) error {
	tag, err := s.Pool.Exec(ctx, `update users
		set password_hash = $2,
		    reset_token_hash = null,
		    reset_sent_at = null,
		    updated_at = now()
		where id = $1`, userID, passwordBaruHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LookupUserByPhone — mencari user berdasarkan nomor WhatsApp.
// Diperlukan untuk lupa sandi lewat phone.
func (s *Store) LookupUserByPhone(ctx context.Context, phone string) (*domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx, `
		select id, tenant_id, email, phone, full_name, password_hash,
		       role, status, activated_at, created_at
		from users where phone = $1 and status = 'ACTIVE'`, phone).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.Phone, &u.FullName, &u.PasswordHash,
		&u.Role, &u.Status, &u.ActivatedAt, &u.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
