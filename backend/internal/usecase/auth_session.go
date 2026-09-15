package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
)

// Pencabutan token (logout).
//
// Token JWT bersifat stateless: sekali diterbitkan ia sah sampai kedaluwarsa
// (12 jam). Tanpa daftar-tolak, token yang sudah dicuri atau tertinggal di HP
// yang hilang tetap bisa dipakai walau pengguna sudah keluar. Karena itu tiap
// token punya jti, dan jti yang dicabut dicatat di Redis.

const prefiksTolak = "rev:"

// ErrTolakTakTersedia: pencabutan tidak bisa diandalkan saat daftar-tolak mati.
var ErrTolakTakTersedia = errors.New("daftar pencabutan token tidak tersedia")

// Logout mencatat jti token ini sebagai dicabut, dengan masa berlaku = SISA umur
// token (bukan 12 jam penuh) supaya kunci lama tidak menumpuk di Redis.
func (a *Auth) Logout(ctx context.Context, c *security.Claims) error {
	if c == nil || c.ID == "" {
		return nil // token terbitan lama (sebelum ada jti): tidak ada yang dicabut
	}
	if c.ExpiresAt == nil {
		return nil
	}
	sisa := time.Until(c.ExpiresAt.Time)
	if sisa <= 0 {
		return nil // sudah kedaluwarsa sendiri, tak perlu dicatat
	}
	if a.Cache == nil || !a.Cache.Enabled() {
		return ErrTolakTakTersedia
	}
	return a.Cache.Set(ctx, prefiksTolak+c.ID, "1", sisa)
}

// TokenDicabut melaporkan apakah jti ini sudah dicabut.
//
// Sengaja gagal-TERBUKA saat Redis mati: menolak SEMUA permintaan hanya karena
// cache mati akan mematikan aplikasi bagi seluruh warga. Karena itu pemanggil
// (RequireAuth) mencatat peringatan ke log, bukan mengembalikan galat.
func (a *Auth) TokenDicabut(ctx context.Context, jti string) bool {
	if jti == "" || a.Cache == nil || !a.Cache.Enabled() {
		return false
	}
	_, ada := a.Cache.Get(ctx, prefiksTolak+jti)
	return ada
}
