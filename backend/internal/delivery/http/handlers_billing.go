package http

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
)

// GenerateTagihan — Bendahara/Ketua menerbitkan tagihan bulanan massal.
func (s *Server) GenerateTagihan(c fiber.Ctx) error {
	var in struct {
		Periode string `json:"periode"`
	}
	_ = c.Bind().JSON(&in)
	hasil, err := s.Billing.GenerateBulanan(c.Context(), s.subject(c), in.Periode)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(hasil)
}

// DaftarTagihan — warga melihat tagihan unit yang dihuninya, pengurus seluruh tenant.
func (s *Server) DaftarTagihan(c fiber.Ctx) error {
	list, err := s.Billing.Daftar(c.Context(), s.subject(c), c.Query("status"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	total := 0.0
	for _, inv := range list {
		if inv.Status == domain.InvoiceUnpaid {
			total += inv.TotalAmount
		}
	}
	return c.JSON(fiber.Map{"items": list, "jumlah": len(list), "total_belum_lunas": total})
}

// DetailTagihan — satu invoice beserta baris itemnya.
func (s *Server) DetailTagihan(c fiber.Ctx) error {
	inv, err := s.Billing.Detail(c.Context(), s.subject(c), c.Params("id"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(inv)
}

// QRISBuat — membuat QRIS dinamis untuk sebuah tagihan.
func (s *Server) QRISBuat(c fiber.Ctx) error {
	out, err := s.Billing.BuatQRIS(c.Context(), s.subject(c), c.Params("id"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(out)
}

// QRISCek — memeriksa status pembayaran ke gateway; bila lunas, tagihan ditutup.
func (s *Server) QRISCek(c fiber.Ctx) error {
	inv, status, err := s.Billing.CekQRIS(c.Context(), s.subject(c), c.Params("id"))
	if err != nil && inv == nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"invoice": inv, "status_gateway": status, "galat": errorText(err)})
}

// CatatTunai — Bendahara mencatat penerimaan uang tunai.
func (s *Server) CatatTunai(c fiber.Ctx) error {
	var in struct {
		Nominal float64 `json:"nominal"`
		Catatan string  `json:"catatan"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	inv, err := s.Billing.CatatTunai(c.Context(), s.subject(c), c.Params("id"), in.Nominal, in.Catatan)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(inv)
}

// BukuKas — daftar mutasi kas + saldo per sumber dana.
func (s *Server) BukuKas(c fiber.Ctx) error {
	list, saldo, err := s.Billing.BukuKas(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": list, "saldo": saldo})
}

// SetorBank — Bendahara mencatat setoran tunai ke rekening paguyuban.
func (s *Server) SetorBank(c fiber.Ctx) error {
	var in struct {
		Nominal float64 `json:"nominal"`
		Catatan string  `json:"catatan"`
		Bukti   string  `json:"bukti"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	grup, err := s.Billing.SetorBank(c.Context(), s.subject(c), in.Nominal, in.Catatan, in.Bukti)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"transfer_group": grup})
}

// SetujuiSetoran — Ketua memverifikasi mutasi setoran.
func (s *Server) SetujuiSetoran(c fiber.Ctx) error {
	if err := s.Billing.SetujuiSetoran(c.Context(), s.subject(c), c.Params("grup")); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "DISETUJUI"})
}

// MasterIuran — daftar komponen iuran.
func (s *Server) MasterIuran(c fiber.Ctx) error {
	items, err := s.Billing.MasterIuran(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": items, "jumlah": len(items)})
}

// SimpanIuran — menambah/mengubah komponen iuran.
func (s *Server) SimpanIuran(c fiber.Ctx) error {
	var in struct {
		ID        string  `json:"id"`
		Code      string  `json:"code"`
		Label     string  `json:"label"`
		Amount    float64 `json:"amount"`
		AppliesTo string  `json:"applies_to"`
		Aktif     bool    `json:"is_active"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	id, err := s.Billing.SimpanIuran(c.Context(), s.subject(c), in.ID, in.Code, in.Label, in.Amount, in.AppliesTo, in.Aktif)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"id": id})
}

// PengaturanTagihan — tanggal terbit & jatuh tempo.
func (s *Server) PengaturanTagihan(c fiber.Ctx) error {
	hari, tempo, err := s.Billing.PengaturanPenagihan(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"billing_day": hari, "due_days": tempo})
}

// SimpanPengaturanTagihan — menyimpan pengaturan penagihan.
func (s *Server) SimpanPengaturanTagihan(c fiber.Ctx) error {
	var in struct {
		BillingDay int `json:"billing_day"`
		DueDays    int `json:"due_days"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Billing.SimpanPengaturan(c.Context(), s.subject(c), in.BillingDay, in.DueDays); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "TERSIMPAN"})
}

// errorText — pesan galat yang aman ditampilkan (tanpa rincian internal).
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return "pemeriksaan ke gateway gagal, coba lagi"
}
