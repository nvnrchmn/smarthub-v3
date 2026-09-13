package http

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
)

func (s *Server) health(c fiber.Ctx) error {
	dbOK := db.Pool.Ping(c.Context()) == nil
	code, status := fiber.StatusOK, "ok"
	if !dbOK {
		code, status = fiber.StatusServiceUnavailable, "degraded"
	}
	return c.Status(code).JSON(fiber.Map{"status": status, "service": "smarthub-api", "database": dbOK})
}

// errStatus — pemetaan error usecase ke kode HTTP. Pesan sengaja umum supaya
// tidak membocorkan apakah sebuah email terdaftar atau tidak.
// detail — memakai penjelasan spesifik dari galat bila ada (mis. "rumah masih
// dihuni, akhiri hunian dulu") agar pengguna tidak menerima pesan yang menyesatkan.
func detail(err error, bawaan string) string {
	t := err.Error()
	if i := strings.Index(t, ": "); i >= 0 && i+2 < len(t) {
		return strings.TrimSpace(t[i+2:])
	}
	return bawaan
}

func errStatus(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrCredentials):
		return fiber.StatusUnauthorized, "email atau kata sandi salah"
	case errors.Is(err, usecase.ErrInactive):
		return fiber.StatusForbidden, "akun belum aktif"
	case errors.Is(err, usecase.ErrUsed):
		return fiber.StatusConflict, "undangan sudah dipakai"
	case errors.Is(err, usecase.ErrExpired):
		return fiber.StatusGone, "undangan atau kode sudah kedaluwarsa"
	case errors.Is(err, usecase.ErrAttempts):
		return fiber.StatusTooManyRequests, "terlalu banyak percobaan, minta kode baru"
	case errors.Is(err, usecase.ErrInvalidInput):
		return fiber.StatusBadRequest, "data tidak valid"
	case errors.Is(err, usecase.ErrBadInput):
		// pesan validasi kita sendiri (aman ditampilkan, memudahkan pengguna)
		return fiber.StatusBadRequest, strings.TrimPrefix(err.Error(), usecase.ErrBadInput.Error()+": ")
	case errors.Is(err, usecase.ErrForbidden):
		return fiber.StatusForbidden, "tidak berhak mengakses data ini"
	case errors.Is(err, usecase.ErrNotFound):
		return fiber.StatusNotFound, "data tidak ditemukan"
	case errors.Is(err, usecase.ErrConflict):
		return fiber.StatusConflict, detail(err, "sudah terdaftar")
	case errors.Is(err, usecase.ErrBillingForbidden):
		return fiber.StatusForbidden, "tidak berhak atas data keuangan ini"
	case errors.Is(err, usecase.ErrBillingNotFound):
		return fiber.StatusNotFound, "tagihan tidak ditemukan"
	case errors.Is(err, usecase.ErrSudahLunas):
		return fiber.StatusConflict, "tagihan sudah lunas"
	case errors.Is(err, usecase.ErrGatewayBelumAktif):
		return fiber.StatusServiceUnavailable, "QRIS belum aktif: kunci Xendit di hub masih mode uji (kunci kembali placeholder). Isi kunci produksi di pengaturan hub, lalu coba lagi. Sementara pakai kas tunai ke Bendahara."
	case errors.Is(err, usecase.ErrGatewaySementara):
		return fiber.StatusServiceUnavailable, "gateway pembayaran sedang bermasalah, coba lagi sebentar lagi."
	case errors.Is(err, usecase.ErrBelumLunas):
		return fiber.StatusConflict, "tagihan tidak bisa dibayar pada status ini"
	}
	// Galat tak terduga wajib tercatat: tanpa ini kegagalan 500 tak bisa dilacak.
	log.Printf("ERROR internal: %v", err)
	return fiber.StatusInternalServerError, "terjadi kesalahan"
}

func (s *Server) Login(c fiber.Ctx) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	token, user, err := s.Auth.Login(c.Context(), in.Email, in.Password)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"token": token, "role": user.Role, "name": user.FullName,
		"tenant_id": user.TenantID, "user_id": user.ID})
}

func (s *Server) Me(c fiber.Ctx) error {
	sub, _ := c.Locals("subject").(*domain.SubjectContext)
	return c.JSON(fiber.Map{"user_id": sub.AccountID, "tenant_id": sub.TenantID,
		"roles": sub.AppRoles, "status": sub.LifecycleStatus})
}

// CreateInvite — hanya pengelola/sekretaris (dijaga RequireRoles + RBAC).
func (s *Server) CreateInvite(c fiber.Ctx) error {
	sub, _ := c.Locals("subject").(*domain.SubjectContext)
	var in struct {
		Email string `json:"email"`
		Phone string `json:"phone"`
		Role  string `json:"role"`
		Name  string `json:"tenant_name"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	link, terkirim, err := s.Auth.CreateInvite(c.Context(), sub.TenantID, in.Email, in.Phone, in.Role, in.Name)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	resp := fiber.Map{"invite_link": link, "expires_in_days": 7, "wa_terkirim": terkirim}
	if !terkirim {
		resp["peringatan"] = "Undangan dibuat, tetapi pesan WhatsApp gagal dikirim. Teruskan tautan ini secara manual."
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (s *Server) ResendOTP(c fiber.Ctx) error {
	var in struct {
		Token      string `json:"token"`
		TenantName string `json:"tenant_name"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Auth.ResendOTP(c.Context(), in.Token, in.TenantName); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "kode baru dikirim"})
}

func (s *Server) AcceptInvite(c fiber.Ctx) error {
	var in struct {
		Token      string `json:"token"`
		OTP        string `json:"otp"`
		FullName   string `json:"full_name"`
		Password   string `json:"password"`
		TenantName string `json:"tenant_name"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	token, user, err := s.Auth.AcceptInvite(c.Context(), in.Token, in.OTP, in.FullName, in.Password, in.TenantName)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"token": token, "user": user})
}
