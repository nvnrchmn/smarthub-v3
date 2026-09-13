package http

import (
	"io"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

func (s *Server) subject(c fiber.Ctx) *domain.SubjectContext {
	sub, _ := c.Locals("subject").(*domain.SubjectContext)
	return sub
}

// MyProfile — data sensus milik warga yang sedang masuk.
func (s *Server) MyProfile(c fiber.Ctx) error {
	p, err := s.Census.MyProfile(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(p)
}

// SubmitMyProfile — warga mengisi/memperbarui data sensusnya sendiri.
func (s *Server) SubmitMyProfile(c fiber.Ctx) error {
	var in postgres.ProfileInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	p, err := s.Census.SubmitMyProfile(c.Context(), s.subject(c), in)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(p)
}

// ListCensus — antrean/daftar sensus untuk pengurus.
func (s *Server) ListCensus(c fiber.Ctx) error {
	items, err := s.Census.List(c.Context(), s.subject(c), c.Query("status"), 0)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": items, "jumlah": len(items)})
}

// CensusDetail — satu profil (data PII tetap tersamar).
func (s *Server) CensusDetail(c fiber.Ctx) error {
	p, err := s.Census.Detail(c.Context(), s.subject(c), c.Params("id"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(p)
}

// RevealPII — membuka NIK & nomor KK asli; setiap akses tercatat di audit_logs.
func (s *Server) RevealPII(c fiber.Ctx) error {
	nik, kk, err := s.Census.RevealPII(c.Context(), s.subject(c), c.Params("id"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"nik": nik, "nomor_kk": kk})
}

// VerifyCensus — Sekretaris/Ketua menyetujui atau menolak dokumen.
func (s *Server) VerifyCensus(c fiber.Ctx) error {
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Census.Verify(c.Context(), s.subject(c), c.Params("id"), in.Status, in.Reason); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": in.Status})
}

// UploadDocument — warga mengunggah KTP/KK ke penyimpanan privat.
func (s *Server) UploadDocument(c fiber.Ctx) error {
	fh, err := c.FormFile("file")
	if err != nil {
		log.Printf("ERROR unggah: kind=%s ct=%s: %v", c.Params("kind"), c.Get("Content-Type"), err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "berkas tidak ditemukan"})
	}
	f, err := fh.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "berkas tidak bisa dibaca"})
	}
	defer f.Close()
	key, err := s.Census.UploadDocument(c.Context(), s.subject(c), c.Params("id"), c.Params("kind"),
		f, fh.Size, fh.Header.Get("Content-Type"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"tersimpan": true, "kunci": key})
}

// DownloadDocument — pengurus yang berhak membuka dokumen (dicatat audit).
func (s *Server) DownloadDocument(c fiber.Ctx) error {
	rc, ctype, size, err := s.Census.DownloadDocument(c.Context(), s.subject(c), c.Params("id"), c.Params("kind"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	c.Set("Content-Type", ctype)
	c.Set("Content-Disposition", "inline")
	// Penting: fasthttp menulis badan respons SETELAH handler kembali, jadi
	// objek tidak boleh ditutup di sini. Pembungkus menutupnya saat habis dibaca.
	return c.SendStream(&closeAfterRead{rc}, int(size))
}

// closeAfterRead — menutup sumber setelah pembacaan selesai (atau gagal).
type closeAfterRead struct{ rc io.ReadCloser }

func (c *closeAfterRead) Read(p []byte) (int, error) {
	n, err := c.rc.Read(p)
	if err != nil {
		_ = c.rc.Close()
	}
	return n, err
}

// CensusStats — ringkasan untuk dasbor.
func (s *Server) CensusStats(c fiber.Ctx) error {
	st, err := s.Census.Stats(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(st)
}

// CreateUnit / ListUnits / AssignUnit / SetLifecycle — pengelolaan rumah & status warga.
func (s *Server) CreateUnit(c fiber.Ctx) error {
	var in struct {
		Block  string `json:"block"`
		Number string `json:"number"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	u, err := s.Census.CreateUnit(c.Context(), s.subject(c), in.Block, in.Number, in.Notes)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.Status(fiber.StatusCreated).JSON(u)
}

func (s *Server) ListUnits(c fiber.Ctx) error {
	items, err := s.Census.ListUnits(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": items, "jumlah": len(items)})
}

func (s *Server) AssignUnit(c fiber.Ctx) error {
	var in domain.OccupancyInput
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if in.ResidentID == "" {
		in.ResidentID = c.Params("id")
	}
	if err := s.Census.AssignUnit(c.Context(), s.subject(c), in); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *Server) SetLifecycle(c fiber.Ctx) error {
	var in struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Census.SetLifecycle(c.Context(), s.subject(c), c.Params("id"), in.Status); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": in.Status})
}
