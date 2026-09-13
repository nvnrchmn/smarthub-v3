package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// CreateHouseUnit — Sekretaris/Ketua mendaftarkan rumah (blok + nomor).
func (s *Store) CreateHouseUnit(ctx context.Context, tenantID, block, number, notes string) (*domain.HouseUnit, error) {
	var u domain.HouseUnit
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `insert into house_units (tenant_id, block, unit_number, notes)
			values ($1,$2,$3,nullif($4,''))
			on conflict (tenant_id, block, unit_number) do update set notes = coalesce(nullif($4,''), house_units.notes)
			returning id, block, unit_number, occupancy_status, coalesce(notes,'')`,
			tenantID, block, number, notes).Scan(&u.ID, &u.Block, &u.UnitNumber, &u.OccupancyStatus, &u.Notes)
	})
	return &u, err
}

// ListHouseUnits — daftar rumah beserta penghuni utama (bila ada).
func (s *Store) ListHouseUnits(ctx context.Context, tenantID string) ([]domain.HouseUnit, error) {
	out := []domain.HouseUnit{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select u.id, u.block, u.unit_number, u.occupancy_status, coalesce(u.notes,''),
				(select count(*) from house_occupancies o where o.house_unit_id = u.id and o.end_date is null),
				coalesce((select p.full_name from house_occupancies o
					join resident_profiles p on p.id = o.resident_id
					where o.house_unit_id = u.id and o.end_date is null
					order by o.is_primary_payer desc, p.full_name limit 1), '')
			from house_units u order by u.block, u.unit_number`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var u domain.HouseUnit
			if err := rows.Scan(&u.ID, &u.Block, &u.UnitNumber, &u.OccupancyStatus, &u.Notes, &u.OccupantCount, &u.PrimaryOccupant); err != nil {
				return err
			}
			out = append(out, u)
		}
		return rows.Err()
	})
	return out, err
}

// SetOccupancy — menempatkan warga pada unit. Selalu menutup penempatan lama
// warga tersebut agar riwayat tetap utuh (end_date, bukan delete).
func (s *Store) SetOccupancy(ctx context.Context, tenantID string, in domain.OccupancyInput) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if in.OccupancyType == "" {
			in.OccupancyType = "OWNER_OCCUPANT"
		}
		// bila warga ini sudah menempati unit lain, tutup penempatan lamanya
		if _, err := tx.Exec(ctx, `update house_occupancies set end_date = current_date
			where resident_id = $1 and end_date is null and house_unit_id <> $2`, in.ResidentID, in.HouseUnitID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `insert into house_occupancies
			(tenant_id, resident_id, house_unit_id, occupancy_type, is_primary_payer)
			values ($1,$2,$3,$4,$5)
			on conflict (house_unit_id, resident_id) where end_date is null
			do update set occupancy_type = excluded.occupancy_type,
			              is_primary_payer = excluded.is_primary_payer`,
			tenantID, in.ResidentID, in.HouseUnitID, in.OccupancyType, in.IsPrimaryPayer); err != nil {
			return err
		}
		// satu pembayar utama per unit
		if in.IsPrimaryPayer {
			if _, err := tx.Exec(ctx, `update house_occupancies set is_primary_payer = false
				where house_unit_id = $1 and resident_id <> $2 and end_date is null`,
				in.HouseUnitID, in.ResidentID); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `update house_units set occupancy_status = 'OCCUPIED'
			where id = $1 and occupancy_status <> 'OCCUPIED'`, in.HouseUnitID)
		return err
	})
}

// SetLifecycle — mutasi status warga (pindah/meninggal) oleh Sekretaris/Ketua.
func (s *Store) SetLifecycle(ctx context.Context, tenantID, profileID, status string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `update resident_profiles set lifecycle_status = $2, updated_at = now()
			where id = $1`, profileID, status); err != nil {
			return err
		}
		if status != domain.LifeActive {
			// warga yang pindah/meninggal tidak lagi menghuni unit
			_, err := tx.Exec(ctx, `update house_occupancies set end_date = current_date
				where resident_id = $1 and end_date is null`, profileID)
			return err
		}
		return nil
	})
}

// CountHouseUnits — ringkasan untuk dasbor pengurus.
func (s *Store) CountHouseUnits(ctx context.Context, tenantID string) (total, occupied int, err error) {
	err = s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `select count(*), count(*) filter (where occupancy_status = 'OCCUPIED')
			from house_units`).Scan(&total, &occupied)
	})
	return
}

// UpdateHouseUnit — mengubah blok/nomor/status/ catatan sebuah rumah.
func (s *Store) UpdateHouseUnit(ctx context.Context, tenantID, id, block, number, status, notes string) (*domain.HouseUnit, error) {
	var u domain.HouseUnit
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `update house_units
			set block = coalesce(nullif($2,''), block),
			    unit_number = coalesce(nullif($3,''), unit_number),
			    occupancy_status = coalesce(nullif($4,''), occupancy_status),
			    notes = $5,
			    updated_at = now()
			where id = $1
			returning id, block, unit_number, occupancy_status, coalesce(notes,'')`,
			id, block, number, status, notes).Scan(&u.ID, &u.Block, &u.UnitNumber, &u.OccupancyStatus, &u.Notes)
	})
	return &u, err
}

// DeleteHouseUnit — hapus rumah. Ditolak bila masih ada penghuni aktif supaya
// riwayat hunian tidak menggantung.
func (s *Store) DeleteHouseUnit(ctx context.Context, tenantID, id string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, `select count(*) from house_occupancies
			where house_unit_id = $1 and end_date is null`, id).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrStillOccupied
		}
		_, err := tx.Exec(ctx, `delete from house_units where id = $1`, id)
		return err
	})
}

// EndOccupancy — warga keluar dari rumah (pindah/kontrak selesai). Riwayat
// ditutup dengan end_date, bukan dihapus; rumah kembali VACANT bila kosong.
func (s *Store) EndOccupancy(ctx context.Context, tenantID, profileID string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `update house_occupancies set end_date = current_date
			where resident_id = $1 and end_date is null
			returning house_unit_id`, profileID)
		if err != nil {
			return err
		}
		var unitIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			unitIDs = append(unitIDs, id)
		}
		rows.Close()
		for _, uid := range unitIDs {
			if _, err := tx.Exec(ctx, `update house_units set occupancy_status = 'VACANT'
				where id = $1 and not exists (
					select 1 from house_occupancies where house_unit_id = $1 and end_date is null)`, uid); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListFamilyCards — daftar kartu keluarga beserta anggotanya.
func (s *Store) ListFamilyCards(ctx context.Context, tenantID string) ([]domain.FamilyCard, error) {
	out := []domain.FamilyCard{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select c.id, coalesce(c.number_last4,''),
				coalesce(c.kk_file_path,'') <> '', c.created_at,
				(select count(*) from resident_profiles p where p.family_card_id = c.id)
			from family_cards c
			order by c.created_at`)
		if err != nil {
			return err
		}
		defer rows.Close()
		var cards []domain.FamilyCard
		for rows.Next() {
			var c domain.FamilyCard
			if err := rows.Scan(&c.ID, &c.NumberLast4, &c.HasFile, &c.CreatedAt, &c.MemberCount); err != nil {
				return err
			}
			cards = append(cards, c)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		for i := range cards {
			// Slice kosong, bukan nil: JSON harus [] agar klien tidak mogok.
			cards[i].Members = []domain.FamilyCardMember{}
			mrows, err := tx.Query(ctx, `select id, full_name, family_role, verification_status, lifecycle_status
				from resident_profiles where family_card_id = $1
				order by case family_role when 'HEAD_OF_FAMILY' then 0 when 'SPOUSE' then 1 when 'CHILD' then 2 else 3 end, full_name`, cards[i].ID)
			if err != nil {
				return err
			}
			for mrows.Next() {
				var m domain.FamilyCardMember
				if err := mrows.Scan(&m.ID, &m.FullName, &m.FamilyRole, &m.VerificationStatus, &m.LifecycleStatus); err != nil {
					mrows.Close()
					return err
				}
				cards[i].Members = append(cards[i].Members, m)
			}
			mrows.Close()
		}
		out = cards
		return nil
	})
	return out, err
}

// CreateFamilyCard — membuat kartu keluarga; nomor KK disimpan terenkripsi.
func (s *Store) CreateFamilyCard(ctx context.Context, tenantID, kkNumber string) (string, error) {
	var id string
	enc, tail := []byte(nil), ""
	if strings.TrimSpace(kkNumber) != "" {
		b, err := s.cipher().Encrypt(kkNumber, tenantID)
		if err != nil {
			return "", err
		}
		enc, tail = b, last4(kkNumber)
	}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `insert into family_cards (tenant_id, family_card_number_enc, number_last4)
			values ($1,$2,nullif($3,'')) returning id`, tenantID, enc, tail).Scan(&id)
	})
	return id, err
}

// AttachMember — memindahkan warga ke sebuah kartu keluarga.
func (s *Store) AttachMember(ctx context.Context, tenantID, cardID, profileID string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update resident_profiles set family_card_id = $2, updated_at = now()
			where id = $1`, profileID, cardID)
		return err
	})
}

// DetachMember — mengeluarkan warga dari kartu keluarga.
func (s *Store) DetachMember(ctx context.Context, tenantID, cardID, profileID string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update resident_profiles set family_card_id = null, updated_at = now()
			where id = $1 and family_card_id = $2`, profileID, cardID)
		return err
	})
}

// SetKKFile — menautkan berkas KK yang diunggah warga ke kartu keluarga.
func (s *Store) SetKKFile(ctx context.Context, tenantID, cardID, path string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update family_cards set kk_file_path = $2 where id = $1`, cardID, path)
		return err
	})
}

// FreeResidentsForCard — kandidat anggota: warga aktif yang belum punya KK.
func (s *Store) FreeResidents(ctx context.Context, tenantID string) ([]domain.FamilyCardMember, error) {
	out := []domain.FamilyCardMember{}
	err := s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `select id, full_name, family_role, verification_status, lifecycle_status
			from resident_profiles where family_card_id is null order by full_name`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var m domain.FamilyCardMember
			if err := rows.Scan(&m.ID, &m.FullName, &m.FamilyRole, &m.VerificationStatus, &m.LifecycleStatus); err != nil {
				return err
			}
			out = append(out, m)
		}
		return rows.Err()
	})
	return out, err
}
