package http

import (

	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/domain"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
)

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func trimPrefix(s, prefix string) string {
	return s[len(prefix):]
}

// SuperadminLogin — endpoint login superadmin (terpisah dari login tenant).
func (s *Server) SuperadminLogin(c fiber.Ctx) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	sa, err := s.Superadmin.Login(c.Context(), in.Email, in.Password)
	if err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	// Token khusus superadmin (claim tid ada tenant_id).
	tok, err := security.IssueSuperadminToken(sa.ID, domain.RoleSuperadmin, s.Auth.JWTSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal menerbitkan token"})
	}
	return c.JSON(fiber.Map{"token": tok, "admin": fiber.Map{"id": sa.ID, "email": sa.Email, "full_name": sa.FullName}})
}

// SuperadminMe — profil superadmin.
func (s *Server) SuperadminMe(c fiber.Ctx) error {
	adminID, _ := c.Locals("superadmin_id").(string)
	sa, err := s.Superadmin.ByID(c.Context(), adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal memuat profil"})
	}
	return c.JSON(fiber.Map{"id": sa.ID, "email": sa.Email, "full_name": sa.FullName})
}

// SuperadminTenants — daftar semua tenant.
func (s *Server) SuperadminTenants(c fiber.Ctx) error {
	tenants, err := s.Superadmin.Tenants(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal memuat tenant"})
	}
	return c.JSON(fiber.Map{"items": tenants, "jumlah": len(tenants)})
}

// RequireSuperadmin — middleware: hanya superadmin yang boleh.
func (s *Server) RequireSuperadmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		h := c.Get("Authorization")
		if !hasPrefix(h, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		claims, err := security.ParseSuperadminToken(trimPrefix(h, "Bearer "), s.Auth.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals("superadmin_id", claims.AdminID)
		return c.Next()
	}
}



// SuperadminAuditLog — audit log global.
func (s *Server) SuperadminAuditLog(c fiber.Ctx) error {
	items, err := s.Superadmin.AuditLog(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal memuat audit log"})
	}
	return c.JSON(fiber.Map{"items": items})
}

// SuperadminSettings — baca pengaturan global.
func (s *Server) SuperadminSettings(c fiber.Ctx) error {
	key := c.Query("key")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "key wajib"})
	}
	val, err := s.Superadmin.GetSetting(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "gagal membaca pengaturan"})
	}
	return c.JSON(fiber.Map{"key": key, "value": val})
}

// SuperadminUpdateSetting — simpan pengaturan global.
func (s *Server) SuperadminUpdateSetting(c fiber.Ctx) error {
	var in struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Superadmin.SetSetting(c.Context(), in.Key, in.Value); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "tersimpan"})
}

// SuperadminResetPassword — ganti password superadmin.
func (s *Server) SuperadminResetPassword(c fiber.Ctx) error {
	adminID, _ := c.Locals("superadmin_id").(string)
	var in struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.Bind().JSON(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "data tidak valid"})
	}
	if err := s.Superadmin.ResetPassword(c.Context(), adminID, in.OldPassword, in.NewPassword); err != nil {
		code, msg := errStatus(err)
		return c.Status(code).JSON(fiber.Map{"error": msg})
	}
	return c.JSON(fiber.Map{"status": "password direset"})
}

