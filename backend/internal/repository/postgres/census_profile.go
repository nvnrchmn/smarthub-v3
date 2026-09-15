package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// ProfileInput — data yang dikirim warga/pengurus saat mengisi sensus.
type ProfileInput struct {
	FullName      string `json:"full_name"`
	NIK           string `json:"nik"`
	KKNumber      string `json:"kk_number"`
	FamilyRole    string `json:"family_role"`
	BirthPlace    string `json:"birth_place"`
	BirthDate     string `json:"birth_date"` // YYYY-MM-DD
	Gender        string `json:"gender"`
	Religion      string `json:"religion"`
	MaritalStatus string `json:"marital_status"`
	Occupation    string `json:"occupation"`
	Education     string `json:"education"`
}

const profileFrom = `resident_profiles p
	left join family_cards f on f.id = p.family_card_id
	left join house_occupancies o on o.resident_id = p.id and o.end_date is null
	left join house_units h on h.id = o.house_unit_id`

const profileCols = `p.id, p.full_name, coalesce(p.nik_last4,''), p.family_role,
	coalesce(p.birth_place,''), coalesce(to_char(p.birth_date,'YYYY-MM-DD'),''),
	coalesce(p.gender,''), coalesce(p.religion,''), coalesce(p.marital_status,''),
	coalesce(p.occupation,''), coalesce(p.education,''), p.verification_status,
	coalesce(p.rejection_reason,''), p.lifecycle_status,
	coalesce(p.family_card_id::text,''), coalesce(p.account_id::text,''),
	(p.ktp_file_path is not null and p.ktp_file_path <> '') as has_ktp,
	(f.kk_file_path is not null and f.kk_file_path <> '') as has_kk,
	coalesce(h.block || '-' || h.unit_number,''), coalesce(o.house_unit_id::text,''), coalesce(o.occupancy_type,''),
	coalesce(o.is_primary_payer,false),
	p.created_at, p.updated_at`
func scanProfile(row pgx.Row) (*domain.ResidentProfile, error) {
	var p domain.ResidentProfile
	err := row.Scan(&p.ID, &p.FullName, &p.NIKLast4, &p.FamilyRole,
		&p.BirthPlace, &p.BirthDate, &p.Gender, &p.Religion, &p.MaritalStatus,
		&p.Occupation, &p.Education, &p.VerificationStatus, &p.RejectionReason,
		&p.LifecycleStatus, &p.FamilyCardID, &p.AccountID,
		&p.HasKTP, &p.HasKK, &p.HouseUnit, &p.HouseUnitID, &p.OccupancyType, &p.IsPrimaryPayer,
		&p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

// UpsertMyProfile — warga menyimpan/memperbarui profil sensusnya sendiri.
func (s *Store) UpsertMyProfile(ctx context.Context, tenantID, userID string, in ProfileInput) (*domain.ResidentProfile, error) {
	var out *domain.ResidentProfile
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		var nikEnc []byte
		if in.NIK != "" {
			b, err := s.cipher().Encrypt(in.NIK, tenantID)
			if err != nil {
				return err
			}
			nikEnc = b
		}
		var fcID any
		if in.KKNumber != "" {
			enc, err := s.cipher().Encrypt(in.KKNumber, tenantID)
			if err != nil {
				return err
			}
			var id string
			err = tx.QueryRow(ctx, `insert into family_cards (tenant_id, family_card_number_enc, number_last4)
				values ($1,$2,$3) returning id`, tenantID, enc, last4(in.KKNumber)).Scan(&id)
			if err != nil {
				return err
			}
			fcID = id
		}
		var birth any
		if in.BirthDate != "" {
			birth = in.BirthDate
		}
		row := tx.QueryRow(ctx, `insert into resident_profiles
			(tenant_id, account_id, family_card_id, full_name, nik_enc, nik_last4, family_role,
			 birth_place, birth_date, gender, religion, marital_status, occupation, education, updated_at)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14, now())
			on conflict (tenant_id, account_id) where account_id is not null do update set
			  family_card_id = coalesce(excluded.family_card_id, resident_profiles.family_card_id),
			  full_name = excluded.full_name,
			  nik_enc = coalesce(excluded.nik_enc, resident_profiles.nik_enc),
			  nik_last4 = coalesce(excluded.nik_last4, resident_profiles.nik_last4),
			  family_role = excluded.family_role,
			  birth_place = coalesce(nullif(excluded.birth_place,''), resident_profiles.birth_place),
			  birth_date = coalesce(excluded.birth_date, resident_profiles.birth_date),
			  gender = coalesce(nullif(excluded.gender,''), resident_profiles.gender),
			  religion = coalesce(nullif(excluded.religion,''), resident_profiles.religion),
			  marital_status = coalesce(nullif(excluded.marital_status,''), resident_profiles.marital_status),
			  occupation = coalesce(nullif(excluded.occupation,''), resident_profiles.occupation),
			  education = coalesce(nullif(excluded.education,''), resident_profiles.education),
			  -- perubahan data pribadi mengembalikan status ke antrean verifikasi
			  verification_status = case when resident_profiles.verification_status = 'VERIFIED'
			                             and excluded.nik_enc is not null then 'UNVERIFIED'
			                             else resident_profiles.verification_status end,
			  updated_at = now()
			returning id
		`, tenantID, userID, fcID, in.FullName, nikEnc, last4(in.NIK), in.FamilyRole,
			in.BirthPlace, birth, in.Gender, in.Religion, in.MaritalStatus, in.Occupation, in.Education)
		// RETURNING hanya boleh memakai kolom tabel itu sendiri; data lengkap
		// (termasuk gabungan kartu keluarga & rumah) diambil lewat select.
		var pid string
		if err := row.Scan(&pid); err != nil {
			return err
		}
		row2 := tx.QueryRow(ctx, `select `+profileCols+` from `+profileFrom+` where p.id = $1`, pid)
		p, err := scanProfile(row2)
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}

// GetProfileByAccount — profil milik akun yang sedang masuk.
func (s *Store) GetProfileByAccount(ctx context.Context, tenantID, userID string) (*domain.ResidentProfile, error) {
	var out *domain.ResidentProfile
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `select `+profileCols+` from `+profileFrom+` where p.account_id = $1`, userID)
		p, err := scanProfile(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

// GetProfileByID — dipakai pengurus saat memeriksa antrean.
func (s *Store) GetProfileByID(ctx context.Context, tenantID, id string) (*domain.ResidentProfile, error) {
	var out *domain.ResidentProfile
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `select `+profileCols+` from `+profileFrom+` where p.id = $1`, id)
		p, err := scanProfile(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	return out, err
}

// ListProfiles — daftar sensus dengan paginasi. offset=0 berarti dari awal.
func (s *Store) ListProfiles(ctx context.Context, tenantID, status string, limit, offset int) ([]domain.ResidentProfile, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	out := []domain.ResidentProfile{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select `+profileCols+` from `+profileFrom+`
			where ($1 = '' or p.verification_status = $1)
			order by p.updated_at desc limit $2 offset $3`, status, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanProfile(rows)
			if err != nil {
				return err
			}
			out = append(out, *p)
		}
		return rows.Err()
	})
	return out, err
}

// CountProfiles — jumlah profil yang memenuhi filter (untuk paginasi).
func (s *Store) CountProfiles(ctx context.Context, tenantID, status string) (int, error) {
	var n int
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select count(*) from resident_profiles p
			where ($1 = '' or p.verification_status = $1)`, status).Scan(&n)
	})
	return n, err
}
func (s *Store) SetVerification(ctx context.Context, tenantID, id, status, byUser, reason string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update resident_profiles
			set verification_status = $2,
			    rejection_reason = nullif($4,''),
			    verified_at = case when $2 = 'VERIFIED' then now() else verified_at end,
			    verified_by = $3,
			    updated_at = now()
			where id = $1`, id, status, byUser, reason)
		return err
	})
}

// SetKTPPath — menyimpan lokasi dokumen KTP di penyimpanan objek privat.
func (s *Store) SetKTPPath(ctx context.Context, tenantID, id, path string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update resident_profiles set ktp_file_path = $2, updated_at = now() where id = $1`, id, path)
		return err
	})
}

// SetKKPath — menyimpan lokasi dokumen KK untuk kartu keluarga terkait.
func (s *Store) SetKKPath(ctx context.Context, tenantID, familyCardID, path string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update family_cards set kk_file_path = $2 where id = $1`, familyCardID, path)
		return err
	})
}

// DocumentPaths — lokasi dokumen untuk keperluan pemeriksaan pengurus.
func (s *Store) DocumentPaths(ctx context.Context, tenantID, id string) (ktp, kk string, err error) {
	err = s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select coalesce(p.ktp_file_path,''), coalesce(f.kk_file_path,'')
			from resident_profiles p left join family_cards f on f.id = p.family_card_id
			where p.id = $1`, id).Scan(&ktp, &kk)
	})
	return ktp, kk, err
}

// DecryptPII — membuka NIK & nomor KK. Hanya dipanggil setelah pemeriksaan ABAC.
func (s *Store) DecryptPII(ctx context.Context, tenantID, id string) (nik, kk string, err error) {
	var nikEnc, kkEnc []byte
	err = s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select p.nik_enc, f.family_card_number_enc
			from resident_profiles p left join family_cards f on f.id = p.family_card_id
			where p.id = $1`, id).Scan(&nikEnc, &kkEnc)
	})
	if err != nil {
		return "", "", err
	}
	if len(nikEnc) > 0 {
		if nik, err = s.cipher().Decrypt(nikEnc, tenantID); err != nil {
			return "", "", fmt.Errorf("dekripsi NIK: %w", err)
		}
	}
	if len(kkEnc) > 0 {
		if kk, err = s.cipher().Decrypt(kkEnc, tenantID); err != nil {
			return "", "", fmt.Errorf("dekripsi KK: %w", err)
		}
	}
	return nik, kk, nil
}

// LogAudit — jejak audit (append-only). Setiap pembukaan PII wajib tercatat.
func (s *Store) LogAudit(ctx context.Context, tenantID, actorID, action, entity, entityID string, detail map[string]any) error {
	b, _ := json.Marshal(detail)
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `insert into audit_logs (tenant_id, actor_id, action, entity, entity_id, detail)
			values ($1,$2,$3,$4,$5,$6)`, tenantID, actorID, action, entity, entityID, b)
		return err
	})
}

// SensusStats — ringkasan untuk dasbor.
func (s *Store) SensusStats(ctx context.Context, tenantID string) (total, verified, pending int, err error) {
	err = s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select count(*),
			count(*) filter (where verification_status = 'VERIFIED'),
			count(*) filter (where verification_status = 'UNVERIFIED')
			from resident_profiles`).Scan(&total, &verified, &pending)
	})
	return
}


// OwnershipOf — unit rumah & kartu keluarga milik akun ini; dipakai ABAC.
func (s *Store) OwnershipOf(ctx context.Context, tenantID, userID string) (units, familyCards []string, err error) {
	units, familyCards = []string{}, []string{}
	err = s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select o.house_unit_id::text, coalesce(p.family_card_id::text,'')
			from resident_profiles p
			left join house_occupancies o on o.resident_id = p.id and o.end_date is null
			where p.account_id = $1`, userID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var unit, fc *string
			if err := rows.Scan(&unit, &fc); err != nil {
				return err
			}
			if unit != nil {
				units = append(units, *unit)
			}
			if fc != nil && *fc != "" {
				familyCards = append(familyCards, *fc)
			}
		}
		return rows.Err()
	})
	return
}
