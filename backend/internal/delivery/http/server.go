package http

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
)

type Server struct {
	Auth   *usecase.Auth
	Census  *usecase.Census
	Billing *usecase.Billing
}

// Auth — middleware: verifikasi JWT, lalu pastikan akun masih ACTIVE di database
// (token yang sudah tidak berhak tidak bisa dipakai walau belum kedaluwarsa).
func (s *Server) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		h := c.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		claims, err := security.ParseToken(strings.TrimPrefix(h, "Bearer "), s.Auth.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		sub, err := s.Auth.SubjectFromToken(c.Context(), claims.UserID, claims.TenantID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "akun tidak aktif"})
		}
		c.Locals("subject", sub)
		return c.Next()
	}
}

// RequireRoles — RBAC baseline; ABAC granular tetap dievaluasi di usecase.
func RequireRoles(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		sub, ok := c.Locals("subject").(*domain.SubjectContext)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		for _, want := range roles {
			for _, got := range sub.AppRoles {
				if got == want {
					return c.Next()
				}
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "peran tidak berwenang"})
	}
}

// Router — seluruh rute API v3.
func (s *Server) Router() *fiber.App {
	app := fiber.New(fiber.Config{AppName: "Smarthub API"})

	// CORS: hanya origin resmi yang diizinkan (SM01-CORS).
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://smarthub.logikraf.id", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Security headers (SM01-HEADERS).
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Content-Security-Policy", "default-src 'self'")
		return c.Next()
	})

	// Global rate limiter per-IP (120 req/menit) untuk semua route /api/* (SM01-LIMIT).
	app.Use("/api", limiter.New(limiter.Config{
		Max:        120,
		Expiration: time.Minute,
	}))

	app.Get("/health", s.health)
	app.Get("/api/health", s.health)

	api := app.Group("/api")
	public := limiter.New(limiter.Config{Max: 15, Expiration: time.Minute})
	api.Post("/auth/login", public, s.Login)
	api.Post("/auth/invite/accept", public, s.AcceptInvite)
	api.Post("/auth/invite/resend-otp", public, s.ResendOTP)

	auth := api.Group("", s.RequireAuth())
	auth.Get("/me", s.Me)
	auth.Post("/invite", RequireRoles(domain.RoleTenantManager, domain.RoleSecretary), s.CreateInvite)

	// Sensus: warga mengurus datanya sendiri; pengurus memeriksa & memutuskan.
	auth.Get("/census/me", s.MyProfile)
	auth.Post("/census/me", s.SubmitMyProfile)
	auth.Get("/census/stats", s.CensusStats)
	auth.Get("/census", RequireRoles(domain.RoleTenantManager, domain.RoleSecretary), s.ListCensus)
	auth.Get("/census/:id", s.CensusDetail)
	auth.Post("/census/:id/reveal", s.RevealPII)
	auth.Post("/census/:id/verify", RequireRoles(domain.RoleTenantManager, domain.RoleSecretary), s.VerifyCensus)
	auth.Post("/census/:id/lifecycle", RequireRoles(domain.RoleTenantManager, domain.RoleSecretary), s.SetLifecycle)
	auth.Post("/census/:id/occupancy", RequireRoles(domain.RoleTenantManager, domain.RoleSecretary), s.AssignUnit)
	auth.Post("/census/:id/documents/:kind", s.UploadDocument)
	auth.Get("/census/:id/documents/:kind", s.DownloadDocument)
	staff := RequireRoles(domain.RoleTenantManager, domain.RoleSecretary)
	// Ketua (Tenant Manager) saja: persetujuan mutasi kas.
	manager := RequireRoles(domain.RoleTenantManager)
	auth.Get("/houses", staff, s.ListUnits)
	auth.Post("/houses", staff, s.CreateUnit)
	auth.Patch("/houses/:id", staff, s.UpdateUnit)
	auth.Delete("/houses/:id", staff, s.DeleteUnit)
	auth.Post("/census/:id/occupancy/end", staff, s.EndOccupancy)
	auth.Get("/family-cards/candidates", staff, s.FreeResidents)
	auth.Get("/family-cards", staff, s.FamilyCards)
	auth.Post("/family-cards", staff, s.CreateFamilyCard)
	auth.Post("/family-cards/:id/members", staff, s.AddFamilyMember)
	auth.Delete("/family-cards/:id/members/:memberId", staff, s.RemoveFamilyMember)

	// Tagihan & kas: warga melihat tagihan unitnya, pengurus mengelola keuangan.
	auth.Get("/billing/invoices", s.DaftarTagihan)
	auth.Get("/billing/invoices/:id", s.DetailTagihan)
	auth.Get("/billing/invoices/:id/qris", s.QRISCek)
	auth.Post("/billing/invoices/:id/qris", s.QRISBuat)
	auth.Post("/billing/invoices/:id/cash", s.CatatTunai)
	auth.Get("/billing/fee-items", staff, s.MasterIuran)
	auth.Post("/billing/fee-items", staff, s.SimpanIuran)
	auth.Get("/billing/settings", staff, s.PengaturanTagihan)
	auth.Post("/billing/settings", staff, s.SimpanPengaturanTagihan)
	auth.Post("/billing/generate", staff, s.GenerateTagihan)
	auth.Get("/billing/ledger", staff, s.BukuKas)
	auth.Post("/billing/ledger/deposit", staff, s.SetorBank)
	auth.Post("/billing/ledger/deposit/:grup/approve", manager, s.SetujuiSetoran)

	return app
}
