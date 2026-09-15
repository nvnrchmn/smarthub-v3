package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// InvoiceWithUnit — invoice dengan info unit dan penghuni untuk pengingat.
type InvoiceWithUnit struct {
	InvoiceID     string
	InvoiceNumber string
	TenantID      string
	TenantName    string
	Total         float64
	DueDate       time.Time
	PenghuniID    string
	PenghuniPhone  string
}

// TunggakanPerluDiingatkan — daftar invoice yang perlu diingatkan.
// Rate limiting: 1 pengingat per invoice per `maxPerHari` jam.
// Jangkauan: invoice yang jatuh tempo dalam `hari` hari ke depan.
func (s *Store) TunggakanPerluDiingatkan(ctx context.Context, tenantID string, hari int, maxPerHari int) ([]InvoiceWithUnit, error) {
	out := []InvoiceWithUnit{}
	rows, err := s.Pool.Query(ctx, `
		SELECT i.id, i.invoice_number, t.name, i.total, i.due_date,
		       p.id as penghuni_id, coalesce(p.phone,'') as phone
		FROM invoices i
		JOIN house_units u ON u.id = i.house_unit_id
		JOIN house_occupancies o ON o.house_unit_id = u.id AND o.end_date IS NULL AND o.is_primary_payer = true
		JOIN resident_profiles p ON p.id = o.resident_id
		JOIN tenants t ON t.id = i.tenant_id
		WHERE i.tenant_id = $1
		  AND i.status = 'UNPAID'
		  AND i.due_date BETWEEN now() AND now() + ($2 || ' days')::interval
		  AND (i.last_reminder_at IS NULL OR i.last_reminder_at < now() - ($3 || ' hours')::interval)
		ORDER BY i.due_date ASC
		LIMIT 200`,
		tenantID, hari, maxPerHari)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var inv InvoiceWithUnit
		if err := rows.Scan(&inv.InvoiceID, &inv.InvoiceNumber, &inv.TenantName, &inv.Total, &inv.DueDate, &inv.PenghuniID, &inv.PenghuniPhone); err != nil {
			return nil, err
		}
		inv.TenantID = tenantID
		out = append(out, inv)
	}
	return out, rows.Err()
}

// CatatPengingatTerkirim — catat timestamp supaya tidak spam hari berikutnya.
func (s *Store) CatatPengingatTerkirim(ctx context.Context, tenantID, invoiceID string) error {
	return s.WithTenant(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE invoices SET last_reminder_at = now() WHERE id = $1`, invoiceID)
		return err
	})
}

func init() {
	_ = sql.ErrNoRows
	_ = time.Now
	_ = errors.New
	_ = pgx.ErrNoRows
}
