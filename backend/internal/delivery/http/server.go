package http

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
)

type Server struct {
	Auth *usecase.Auth
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

	return app
}
