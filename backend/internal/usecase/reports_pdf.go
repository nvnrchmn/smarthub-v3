package usecase

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// potongPDF — batasi panjang teks supaya tabel tidak meluber.
func potongPDF(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "."
}

// PDF — laporan bulanan + daftar tunggakan, siap diunduh/dikirim pengurus.
func (r *Reports) PDF(rek *domain.RekapBulanan, tun *domain.Tunggakan) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(0, 8, "Laporan Iuran Warga", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("%s - periode %s", rek.TenantNama, rek.Periode), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Ringkasan", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	ringkas := [][2]string{
		{"Jumlah tagihan", fmt.Sprintf("%d tagihan", rek.JumlahInvoice)},
		{"Total ditagih", "Rp " + rupiah(rek.TotalDitagih)},
		{"Sudah dibayar", "Rp " + rupiah(rek.TotalDibayar)},
		{"Belum dibayar (periode ini)", "Rp " + rupiah(rek.TotalTunggakan)},
		{"Lunas / belum", fmt.Sprintf("%d / %d unit", rek.JumlahLunas, rek.JumlahBelum)},
		{"Kas tunai / QRIS", "Rp " + rupiah(rek.KasTunai) + " / Rp " + rupiah(rek.KasQRIS)},
	}
	for _, b := range ringkas {
		pdf.CellFormat(75, 5.5, b[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(0, 5.5, b[1], "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Rincian iuran", "", 1, "L", false, 0, "")
	headerPDF(pdf, [][3]any{{95.0, "Jenis", "L"}, {25.0, "Jumlah", "C"}, {0.0, "Nominal", "R"}})
	pdf.SetFont("Helvetica", "", 9)
	for _, it := range rek.Rincian {
		pdf.CellFormat(95, 6, potongPDF(it.Label, 60), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%d", it.Jumlah), "1", 0, "C", false, 0, "")
		pdf.CellFormat(0, 6, rupiah(it.Amount), "1", 1, "R", false, 0, "")
	}
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, fmt.Sprintf("Tunggakan per unit (total Rp %s, %d unit)", rupiah(tun.Total), tun.JumlahUnit), "", 1, "L", false, 0, "")
	headerPDF(pdf, [][3]any{{35.0, "Unit", "L"}, {55.0, "Kepala keluarga", "L"}, {25.0, "Periode", "C"}, {20.0, "Hari", "C"}, {0.0, "Total", "R"}})
	pdf.SetFont("Helvetica", "", 9)
	batas := len(tun.Baris)
	if batas > 60 {
		batas = 60
	}
	for _, t := range tun.Baris[:batas] {
		nama := t.KepalaKeluarga
		if nama == "" {
			nama = "-"
		}
		pdf.CellFormat(35, 6, potongPDF(t.Unit, 18), "1", 0, "L", false, 0, "")
		pdf.CellFormat(55, 6, potongPDF(nama, 32), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, t.PeriodeTertua, "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", t.HariTerlambat), "1", 0, "C", false, 0, "")
		pdf.CellFormat(0, 6, rupiah(t.TotalTunggakan), "1", 1, "R", false, 0, "")
	}
	if len(tun.Baris) > batas {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d unit lain (lihat CSV)", len(tun.Baris)-batas), "", 1, "L", false, 0, "")
	}
	if len(tun.Baris) == 0 {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 6, "Tidak ada tunggakan.", "", 1, "L", false, 0, "")
	}

	pdf.Ln(3)
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 5, "Dibuat otomatis oleh Smarthub pada "+time.Now().Format("02 Jan 2006 15:04"), "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// headerPDF — baris judul tabel (lebar, teks, perataan).
func headerPDF(pdf *fpdf.Fpdf, kolom [][3]any) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(238, 240, 245)
	for _, k := range kolom {
		pdf.CellFormat(k[0].(float64), 6, k[1].(string), "1", 0, k[2].(string), true, 0, "")
	}
	pdf.Ln(-1)
}
