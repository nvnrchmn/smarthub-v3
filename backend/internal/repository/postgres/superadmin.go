package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// SuperadminByEmail — ambil superadmin by email (untuk login).
func (s *Store) SuperadminByEmail(ctx context.Context, email string) (*domain.Superadmin, error) {
	var sa domain.Superadmin
	err := s.Pool.QueryRow(ctx, `select id, email, full_name, password_hash, status, last_login_at from superadmins where email = lower($1)`, email).
		Scan(&sa.ID, &sa.Email, &sa.FullName, &sa.PasswordHash, &sa.Status, &sa.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// SetLastLogin — catat waktu login terakhir superadmin.
func (s *Store) SetLastLogin(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `update superadmins set last_login_at = now() where id = $1`, id)
	return err
}

// ResetSuperadminPassword — ganti password superadmin (hanya milik superadmin itu).
func (s *Store) ResetSuperadminPassword(ctx context.Context, id, newHash string) error {
	tag, err := s.Pool.Exec(ctx, `update superadmins set password_hash = $2 where id = $1 and status = 'ACTIVE'`, id, newHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetSetting — baca satu pengaturan global.
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.Pool.QueryRow(ctx, `select value from settings where key = $1`, key).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSetting — simpan/ubah pengatan global.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.Pool.Exec(ctx, `insert into settings (key, value, updated_at) values ($1, $2, now())
		on conflict (key) do update set value = excluded.value, updated_at = now()`, key, value)
	return err
}

// AllSettings — baca semua pengaturan global (key -> value).
func (s *Store) AllSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.Pool.Query(ctx, `select key, value from settings order by key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// SuperadminByID — ambil superadmin by ID.
func (s *Store) SuperadminByID(ctx context.Context, id string) (*domain.Superadmin, error) {
	var sa domain.Superadmin
	err := s.Pool.QueryRow(ctx, `select id, email, full_name, password_hash, status, last_login_at from superadmins where id = $1`, id).
		Scan(&sa.ID, &sa.Email, &sa.FullName, &sa.PasswordHash, &sa.Status, &sa.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// CreateSuperadmin — buat akun superadmin (dipanggil sekali saat setup).
func (s *Store) CreateSuperadmin(ctx context.Context, email, fullName, hash string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx, `insert into superadmins (email, full_name, password_hash) values (lower($1), $2, $3) returning id`, email, fullName, hash).Scan(&id)
	return id, err
}

// Tenants — ambil seluruh tenant (manajemen platform).
//
// Lewat WithSuperadmin: RLS tenants hanya membuka baris milik tenant aktif,
// jadi tanpa GUC app.superadmin daftar ini selalu kosong.
func (s *Store) AllTenants(ctx context.Context) ([]domain.Tenant, error) {
	var out []domain.Tenant
	err := s.WithSuperadmin(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select id, name, slug, status, created_at from tenants order by created_at desc`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var t domain.Tenant
			if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt); err != nil {
				return err
			}
			out = append(out, t)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AuditLogGlobal — ambil audit_logs dari SEMUA tenant (superadmin only).
func (s *Store) AuditLogGlobal(ctx context.Context, limit int) ([]domain.AuditLog, error) {
	var out []domain.AuditLog
	err := s.WithSuperadmin(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select id, tenant_id, action, entity, entity_id, detail, created_at
			from audit_logs order by created_at desc limit $1`, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a domain.AuditLog
			if err := rows.Scan(&a.ID, &a.TenantID, &a.Action, &a.Entity, &a.EntityID, &a.Detail, &a.CreatedAt); err != nil {
				return err
			}
			out = append(out, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
