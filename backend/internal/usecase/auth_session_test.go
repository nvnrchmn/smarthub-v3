package usecase

import (
	"context"
	"testing"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/cache"
)

const rahasiaUji = "rahasia-uji-jangan-dipakai-di-produksi"

// tokenUji menerbitkan token lalu mengurai klaim-nya kembali.
func tokenUji(t *testing.T, uid string) *security.Claims {
	t.Helper()
	tok, err := security.GenerateToken(uid, "tenant-uji", "warga", rahasiaUji)
	if err != nil {
		t.Fatalf("terbitkan token: %v", err)
	}
	c, err := security.ParseToken(tok, rahasiaUji)
	if err != nil {
		t.Fatalf("urai token: %v", err)
	}
	return c
}

func TestLogoutMencabutToken(t *testing.T) {
	ctx := context.Background()
	c := cache.New("127.0.0.1:6379")
	if !c.Enabled() {
		t.Skip("Redis tidak tersedia — uji pencabutan dilewati")
	}
	a := &Auth{Cache: c, JWTSecret: rahasiaUji}

	klaim := tokenUji(t, "user-1")
	t.Cleanup(func() { _ = c.Del(ctx, "rev:"+klaim.ID) })

	if klaim.ID == "" {
		t.Fatal("token terbitan baru tidak punya jti — pencabutan tidak mungkin")
	}
	if a.TokenDicabut(ctx, klaim.ID) {
		t.Fatal("token yang baru terbit seharusnya BELUM dicabut")
	}
	if err := a.Logout(ctx, klaim); err != nil {
		t.Fatalf("logout gagal: %v", err)
	}
	if !a.TokenDicabut(ctx, klaim.ID) {
		t.Fatal("setelah logout, token seharusnya sudah dicabut")
	}

	// Token milik sesi/perangkat lain tidak boleh ikut tercabut.
	lain := tokenUji(t, "user-2")
	t.Cleanup(func() { _ = c.Del(ctx, "rev:"+lain.ID) })
	if a.TokenDicabut(ctx, lain.ID) {
		t.Fatal("token lain ikut tercabut — logout tidak boleh memangsa sesi lain")
	}
}
