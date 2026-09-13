package crypto

import (
	"encoding/hex"
	"strings"
	"testing"
)

func testCipher(t *testing.T) *Cipher {
	t.Helper()
	key := strings.Repeat("ab", 32)
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := testCipher(t)
	nik := "3276012509900001"
	ct, err := c.Encrypt(nik, "tenant-a")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if strings.Contains(string(ct), nik) {
		t.Fatal("ciphertext masih memuat NIK apa adanya")
	}
	pt, err := c.Decrypt(ct, "tenant-a")
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if pt != nik {
		t.Fatalf("hasil dekripsi %q, ingin %q", pt, nik)
	}
}

// Ciphertext tenant lain harus GAGAL didekripsi (AAD = tenant_id).
func TestDecryptWrongTenantFails(t *testing.T) {
	c := testCipher(t)
	ct, err := c.Encrypt("3276012509900001", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Decrypt(ct, "tenant-b"); err == nil {
		t.Fatal("ciphertext tenant A tidak boleh terbaca di tenant B")
	}
}

func TestDecryptTamperedFails(t *testing.T) {
	c := testCipher(t)
	ct, err := c.Encrypt("3276012509900001", "t")
	if err != nil {
		t.Fatal(err)
	}
	ct[len(ct)-1] ^= 0xff
	if _, err := c.Decrypt(ct, "t"); err == nil {
		t.Fatal("ciphertext yang diubah harus ditolak")
	}
}

func TestCipherEmptyAndBadKey(t *testing.T) {
	c := testCipher(t)
	ct, err := c.Encrypt("", "t")
	if err != nil || ct != nil {
		t.Fatalf("nilai kosong harus jadi NULL tanpa error (ct=%v err=%v)", ct, err)
	}
	pt, err := c.Decrypt(nil, "t")
	if err != nil || pt != "" {
		t.Fatal("NULL harus kembali sebagai string kosong")
	}
	if _, err := NewCipher(hex.EncodeToString([]byte("pendek"))); err == nil {
		t.Fatal("kunci bukan 32 byte harus ditolak")
	}
}
