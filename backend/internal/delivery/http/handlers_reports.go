package http

import (
	"bytes"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

// LaporanRekap — ringkasan bulanan (JSON). Query: period=YYYY-MM.
func (s *Server) LaporanRekap(c fiber.Ctx) error {
	rek, err := s.Reports.Rekap(c.Context(), s.subject(c), c.Query("period"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(rek)
}

// LaporanTunggakan — daftar tunggakan per unit + umur tagihan.
func (s *Server) LaporanTunggakan(c fiber.Ctx) error {
	tun, err := s.Reports.Tunggakan(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(tun)
}

// LaporanUnduh — unduh berkas laporan. Query: format=pdf|csv, period=YYYY-MM.
func (s *Server) LaporanUnduh(c fiber.Ctx) error {
	sub := s.subject(c)
	rek, err := s.Reports.Rekap(c.Context(), sub, c.Query("period"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	tun, err := s.Reports.Tunggakan(c.Context(), sub)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	if c.Query("format") == "csv" {
		isi := s.Reports.CSV(rek, tun)
		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=laporan-iuran-%s.csv", rek.Periode))
		return c.SendStream(bytes.NewReader(isi), len(isi))
	}
	isi, err := s.Reports.PDF(rek, tun)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal membuat PDF"})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=laporan-iuran-%s.pdf", rek.Periode))
	return c.SendStream(bytes.NewReader(isi), len(isi))
}

// LaporanKirim — ringkasan laporan ke WhatsApp seluruh pengurus.
func (s *Server) LaporanKirim(c fiber.Ctx) error {
	var in struct {
		Periode string `json:"periode"`
	}
	_ = c.Bind().JSON(&in)
	terkirim, err := s.Reports.Kirim(c.Context(), s.subject(c), in.Periode)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "ok", "terkirim": terkirim})
}
