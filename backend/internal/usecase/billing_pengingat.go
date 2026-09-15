// Sprint 7 — Pengingat tunggakan.

package usecase

import (
	"context"
	"fmt"
	"log"
)

// HasilPengingat — ringkasan pengiriman pengingat.
type HasilPengingat struct {
	Dikirim  int `json:"dikirim"`
	Dilewati int `json:"dilewati"`
	Gagal    int `json:"gagal"`
}

// PengingatTunggakan — kirim pengingat ke penghuni yang tagihannya belum dibayar
// dan sudah mendekati jatuh tempo (default: 3 hari sebelum jatuh tempo).
// Hanya mengirim 1 pengingat per invoice per hari (rate limiting).
func (b *Billing) PengingatTunggakan(ctx context.Context, tenantID string) (*HasilPengingat, error) {
	if b.Notify == nil {
		return &HasilPengingat{}, nil
	}
	hasil := &HasilPengingat{}
	daftar, err := b.Store.TunggakanPerluDiingatkan(ctx, tenantID, 3, 24)
	if err != nil {
		return hasil, err
	}
	for _, inv := range daftar {
		pesan := fmt.Sprintf("Tagihan %s sebesar Rp %s jatuh tempo pada %s", inv.InvoiceNumber, rupiah(inv.Total), inv.DueDate.Format("02 Jan 2006"))
		if err := b.Notify.SendPengingatTunggakan(inv.PenghuniPhone, pesan, inv.TenantName); err != nil {
			hasil.Gagal++
			log.Printf("[pengingat] gagal kirim ke %s: %v", inv.PenghuniPhone, err)
			continue
		}
		if err := b.Store.CatatPengingatTerkirim(ctx, tenantID, inv.InvoiceID); err != nil {
			hasil.Gagal++
			continue
		}
		hasil.Dikirim++
	}
	return hasil, nil
}
