package security

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — identitas yang dibawa token. tenant_id ada di dalam token, sehingga
// permintaan ke tenant lain tidak bisa dipalsukan lewat header saja.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

const tokenTTL = 12 * time.Hour

func GenerateToken(userID, tenantID, role, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT_SECRET belum diisi")
	}
	jti, err := NewTokenID()
	if err != nil {
		return "", err
	}
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			// ID (jti) unik per token: inilah yang dicatat di daftar-tolak saat
			// pengguna keluar, sehingga token bisa dicabut sebelum kedaluwarsa.
			ID:        jti,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// NewTokenID membuat ID token acak 128-bit (hex 32 karakter).
func NewTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func ParseToken(token, secret string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("algoritma tanda tangan tidak didukung")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}

// SuperadminClaims — klaim khusus superadmin (tidak ada tenant_id, akses global).
type SuperadminClaims struct {
	AdminID string `json:"aid"`
	Role    string `json:"role"`
	jwt.RegisteredClaims
}

// IssueSuperadminToken — token JWT untuk superadmin.
func IssueSuperadminToken(adminID, role, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT_SECRET belum diisi")
	}
	claims := SuperadminClaims{
		AdminID: adminID,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   adminID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseSuperadminToken — verifikasi token superadmin.
func ParseSuperadminToken(token, secret string) (*SuperadminClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &SuperadminClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("algoritma tanda tangan tidak didukung")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*SuperadminClaims)
	if !ok || !parsed.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}
