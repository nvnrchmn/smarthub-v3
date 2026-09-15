package http

import (
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
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
	p := usecase.ParsePagination(c.Queries())
	items, err := s.Census.List(c.Context(), s.subject(c), c.Query("status"), p.Per, p.Offset())
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	total, err := s.Census.Count(c.Context(), s.subject(c), c.Query("status"))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	lastPage := (total + p.Per - 1) / p.Per
	if lastPage < 1 {
		lastPage = 1
	}
	return c.JSON(usecase.Paginated[[]domain.ResidentProfile]{
		Items:    items,
		Total:    total,
		Page:     p.Page,
		PerPage:  p.Per,
		LastPage: lastPage,
	})
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

// UpdateUnit — PATCH /api/houses/:id
func (s *Server) UpdateUnit(c fiber.Ctx) error {
	var in struct {
		Block  string `json:"block"`
		Number string `json:"number"`
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	u, err := s.Census.UpdateUnit(c.Context(), s.subject(c), c.Params("id"), in.Block, in.Number, in.Status, in.Notes)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(u)
}

// DeleteUnit — DELETE /api/houses/:id
func (s *Server) DeleteUnit(c fiber.Ctx) error {
	if err := s.Census.DeleteUnit(c.Context(), s.subject(c), c.Params("id")); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "terhapus"})
}

// EndOccupancy — POST /api/census/:id/occupancy/end
func (s *Server) EndOccupancy(c fiber.Ctx) error {
	if err := s.Census.EndOccupancy(c.Context(), s.subject(c), c.Params("id")); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "hunian diakhiri"})
}

// FamilyCards — GET /api/family-cards
func (s *Server) FamilyCards(c fiber.Ctx) error {
	cards, err := s.Census.FamilyCards(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": cards, "jumlah": len(cards)})
}

// CreateFamilyCard — POST /api/family-cards
func (s *Server) CreateFamilyCard(c fiber.Ctx) error {
	var in struct {
		KKNumber string `json:"kk_number"`
	}
	_ = c.Bind().JSON(&in)
	id, err := s.Census.CreateFamilyCard(c.Context(), s.subject(c), strings.TrimSpace(in.KKNumber))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

// AddFamilyMember — POST /api/family-cards/:id/members
func (s *Server) AddFamilyMember(c fiber.Ctx) error {
	var in struct {
		ResidentID string `json:"resident_id"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Census.AttachMember(c.Context(), s.subject(c), c.Params("id"), in.ResidentID); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "anggota ditambahkan"})
}

// RemoveFamilyMember — DELETE /api/family-cards/:id/members/:memberId
func (s *Server) RemoveFamilyMember(c fiber.Ctx) error {
	if err := s.Census.DetachMember(c.Context(), s.subject(c), c.Params("id"), c.Params("memberId")); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "anggota dikeluarkan"})
}

// FreeResidents — GET /api/census/free (warga yang belum punya KK)
func (s *Server) FreeResidents(c fiber.Ctx) error {
	items, err := s.Census.FreeResidents(c.Context(), s.subject(c))
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"items": items, "jumlah": len(items)})
}
