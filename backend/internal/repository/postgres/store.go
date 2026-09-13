package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/crypto"
)

// Store — akses database untuk auth/onboarding.
//
// Semua query yang menyentuh tabel ber-RLS dijalankan lewat WithTenant supaya
// app.tenant_id selalu disetel; lookup tanpa konteks tenant (login, aktivasi)
// memakai fungsi SECURITY DEFINER yang hanya mengembalikan kolom terbatas.
type Store struct {
	// Cipher dipakai untuk enkripsi NIK/KK (AES-256-GCM, AAD = tenant_id).
	Cipher *crypto.Cipher
	Pool *pgxpool.Pool
}

var ErrNotFound = errors.New("tidak ditemukan")

func New(pool *pgxpool.Pool) *Store { return &Store{Pool: pool} }

// WithTenant menjalankan fn di dalam transaksi dengan app.tenant_id diset.
func (s *Store) WithTenant(ctx context.Context, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "select set_config('app.tenant_id', $1, true)", tenantID); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// WithSuperadmin menjalankan fn di dalam transaksi dengan app.superadmin='on'.
//
// Dipakai HANYA untuk operasi tingkat platform (daftar seluruh tenant, audit
// log global). Policy RLS *_superadmin (migrasi 0008) yang membukanya, dan
// hanya untuk tabel tenants + audit_logs.
func (s *Store) WithSuperadmin(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "select set_config('app.superadmin', 'on', true)"); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// LookupUserByEmail — dipakai login; lewat fungsi SECURITY DEFINER karena belum
// ada konteks tenant pada saat itu.
func (s *Store) LookupUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx,
		"select id, tenant_id, password_hash, role, status, full_name, coalesce(phone,'') from auth_lookup_user($1)",
		email,
	).Scan(&u.ID, &u.TenantID, &u.PasswordHash, &u.Role, &u.Status, &u.FullName, &u.Phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Email = email
	return &u, nil
}

// CreateTenantWithManager — bootstrap tenant pertama (dipakai CLI, bukan endpoint publik).
func (s *Store) CreateTenantWithManager(ctx context.Context, tenantName, slug, email, passwordHash, fullName string) (string, string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)

	var tenantID string
	if err := tx.QueryRow(ctx,
		"insert into tenants (name, slug) values ($1, $2) returning id", tenantName, slug,
	).Scan(&tenantID); err != nil {
		return "", "", err
	}
	var userID string
	if err := tx.QueryRow(ctx,
		`insert into users (tenant_id, email, full_name, password_hash, role, status, activated_at)
		 values ($1, lower($2), $3, $4, $5, 'ACTIVE', now()) returning id`,
		tenantID, email, fullName, passwordHash, domain.RoleTenantManager,
	).Scan(&userID); err != nil {
		return "", "", err
	}
	return tenantID, userID, tx.Commit(ctx)
}

var _ = time.Now

// cipher — memastikan kunci enkripsi tersedia sebelum data PII disimpan.
func (s *Store) cipher() *crypto.Cipher {
	if s.Cipher == nil {
		panic("cipher belum dipasang: AES_MASTER_KEY kosong")
	}
	return s.Cipher
}
