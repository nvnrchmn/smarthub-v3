package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

// LupaSandiResult — hasil langkah 1 lupa sandi.
type LupaSandiResult struct {
	Dikirim bool   `json:"dikirim"`
	Saluran string `json:"saluran"`
}

// ErrResetTidakDikenal — token tidak valid atau sudah kedaluwarsa.
var ErrResetTidakDikenal = errors.New("token tidak valid atau sudah kedaluwarsa")

// LupaSandiManager — mengelola alur lupa sandi.
type LupaSandiManager struct {
	Store  *postgres.Store
	Notify Notifier
}

// NewLupaSandi — buat LupaSandiManager baru.
func NewLupaSandi(store *postgres.Store, notifier Notifier) *LupaSandiManager {
	return &LupaSandiManager{
		Store:  store,
		Notify: notifier,
	}
}

// LupaSandiLangkah1 — menerima email/phone, membuat token reset, kirim ke WA.
// Sengaja TIDAK memberi tahu apakah email/phone terdaftar.
func (m *LupaSandiManager) LupaSandiLangkah1(ctx context.Context, emailOrPhone string) (LupaSandiResult, error) {
	u, err := m.Store.LookupUserByEmail(ctx, emailOrPhone)
	if errors.Is(err, postgres.ErrNotFound) {
		u, err = m.Store.LookupUserByPhone(ctx, emailOrPhone)
	}
	if err != nil {
		log.Printf("[lupa-sandi] permintaan untuk %s (tidak ditemukan)", emailOrPhone)
		return LupaSandiResult{}, nil
	}

	// Buat token reset (plaintext dikirim ke user, hash disimpan di DB)
	plaintext, err := m.Store.BuatTokenReset(ctx, u.ID)
	if err != nil {
		return LupaSandiResult{}, fmt.Errorf("simpan token: %w", err)
	}

	// Kirim ke WhatsApp
	if u.Phone != "" {
		if err := m.Notify.SendLupaSandi(u.Phone, plaintext, u.FullName); err != nil {
			log.Printf("[lupa-sandi] gagal kirim WA ke %s: %v", u.Phone, err)
		}
	}

	log.Printf("[lupa-sandi] token dibuat untuk user %s", u.ID)
	saluran := "whatsapp"
	if u.Phone == "" {
		saluran = "tidak_ada_kontak"
	}
	return LupaSandiResult{Dikirim: true, Saluran: saluran}, nil
}

// LupaSandiLangkah2 — verifikasi token & reset password.
func (m *LupaSandiManager) LupaSandiLangkah2(ctx context.Context, token, passwordBaru string) error {
	if len(passwordBaru) < 8 {
		return ErrBadInput
	}

	userID, expired, err := m.Store.AmbilUserUntukReset(ctx, token)
	if err != nil || expired || userID == "" {
		return ErrResetTidakDikenal
	}

	hash, err := security.HashPassword(passwordBaru)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := m.Store.SetelPasswordBaru(ctx, userID, hash); err != nil {
		return err
	}

	log.Printf("[lupa-sandi] password berhasil diubah untuk user %s", userID)
	return nil
}
