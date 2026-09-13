package postgres

import (
	"context"

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
		rows, err := tx.Query(ctx, `select id, block, unit_number, occupancy_status, coalesce(notes,'')
			from house_units order by block, unit_number`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var u domain.HouseUnit
			if err := rows.Scan(&u.ID, &u.Block, &u.UnitNumber, &u.OccupancyStatus, &u.Notes); err != nil {
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
