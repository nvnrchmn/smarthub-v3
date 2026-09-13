package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

// Reports — laporan bulanan & daftar tunggakan (hanya pengurus).
type Reports struct {
	Store  *postgres.Store
	Notify Notifier
}

func (r *Reports) gerbang(sub *domain.SubjectContext) error {
	if !domain.CanAccess(*sub, domain.ResourceContext{TenantID: sub.TenantID}, domain.ActionManageBilling) {
		return ErrBillingForbidden
	}
	return nil
}

// Rekap — ringkasan satu periode (default bulan berjalan).
func (r *Reports) Rekap(ctx context.Context, sub *domain.SubjectContext, periode string) (*domain.RekapBulanan, error) {
	if err := r.gerbang(sub); err != nil {
		return nil, err
	}
	periode = strings.TrimSpace(periode)
	if periode == "" {
		periode = time.Now().Format("2006-01")
	}
	if len(periode) != 7 || periode[4] != '-' {
		return nil, fmt.Errorf("%w: periode harus berformat YYYY-MM", ErrBadInput)
	}
	return r.Store.RekapBulanan(ctx, sub.TenantID, periode)
}

// Tunggakan — daftar tagihan belum lunas + kelompok umur.
func (r *Reports) Tunggakan(ctx context.Context, sub *domain.SubjectContext) (*domain.Tunggakan, error) {
	if err := r.gerbang(sub); err != nil {
		return nil, err
	}
	return r.Store.Tunggakan(ctx, sub.TenantID)
}

// CSV — dua bagian dalam satu berkas (pemisah ';' + BOM supaya rapi di Excel).
func (r *Reports) CSV(rek *domain.RekapBulanan, tun *domain.Tunggakan) []byte {
	var b bytes.Buffer
	b.WriteString("\xEF\xBB\xBF")
	w := csv.NewWriter(&b)
	w.Comma = ';'
	tulis := func(row ...string) { _ = w.Write(row) }
	tulis("Laporan iuran", rek.TenantNama, "periode "+rek.Periode)
	tulis("Jumlah tagihan", fmt.Sprintf("%d", rek.JumlahInvoice))
	tulis("Total ditagih", rupiah(rek.TotalDitagih))
	tulis("Sudah dibayar", rupiah(rek.TotalDibayar))
	tulis("Belum dibayar (periode ini)", rupiah(rek.TotalTunggakan))
	tulis("Lunas", fmt.Sprintf("%d", rek.JumlahLunas), "belum", fmt.Sprintf("%d", rek.JumlahBelum))
	tulis("Kas tunai", rupiah(rek.KasTunai), "QRIS", rupiah(rek.KasQRIS))
	tulis()
	tulis("Rincian iuran", "jumlah", "nominal")
	for _, it := range rek.Rincian {
		tulis(it.Label, fmt.Sprintf("%d", it.Jumlah), rupiah(it.Amount))
	}
	tulis()
	tulis("Tunggakan per unit", "kepala keluarga", "telepon", "periode tertua", "jumlah tagihan", "total", "hari", "umur")
	for _, t := range tun.Baris {
		tulis(t.Unit, t.KepalaKeluarga, t.Telepon, t.PeriodeTertua,
			fmt.Sprintf("%d", t.JumlahTagihan), rupiah(t.TotalTunggakan), fmt.Sprintf("%d", t.HariTerlambat), t.Bucket)
	}
	w.Flush()
	return b.Bytes()
}

// Kirim — ringkasan laporan ke WhatsApp semua pengurus yang punya nomor.
func (r *Reports) Kirim(ctx context.Context, sub *domain.SubjectContext, periode string) (int, error) {
	rek, err := r.Rekap(ctx, sub, periode)
	if err != nil {
		return 0, err
	}
	tun, err := r.Tunggakan(ctx, sub)
	if err != nil {
		return 0, err
	}
	penerima, err := r.Store.Pengurus(ctx, sub.TenantID)
	if err != nil {
		return 0, err
	}
	if len(penerima) == 0 {
		return 0, fmt.Errorf("%w: belum ada pengurus dengan nomor WhatsApp (isi nomor di menu Tim)", ErrBadInput)
	}
	pesan := fmt.Sprintf("*Laporan Iuran %s*\nPeriode: %s\n\nDitagih: %s (%d tagihan)\nDibayar: %s\nBelum bayar: %s (%d unit)\nKas tunai: %s · QRIS: %s\n\nDetail per unit: buka menu Laporan di aplikasi.",
		rek.TenantNama, rek.Periode, rupiah(rek.TotalDitagih), rek.JumlahInvoice,
		rupiah(rek.TotalDibayar), rupiah(tun.Total), tun.JumlahUnit, rupiah(rek.KasTunai), rupiah(rek.KasQRIS))
	terkirim := 0
	for _, p := range penerima {
		if err := r.Notify.SendTagihan(p.Phone, pesan, rek.TenantNama); err != nil {
			log.Printf("[laporan] gagal kirim ke pengurus (%s): %v", p.Role, err)
			continue
		}
		terkirim++
	}
	_ = r.Store.LogAudit(ctx, sub.TenantID, sub.AccountID, "REPORT_SEND", "periode", rek.Periode,
		map[string]any{"terkirim": terkirim, "penerima": len(penerima)})
	if terkirim == 0 {
		return 0, fmt.Errorf("laporan gagal terkirim ke semua pengurus; periksa kanal WhatsApp")
	}
	return terkirim, nil
}
