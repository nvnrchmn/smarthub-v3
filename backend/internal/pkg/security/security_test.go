package security

import "testing"

func TestPasswordHashing(t *testing.T) {
	h, err := HashPassword("rahasia123")
	if err != nil {
		t.Fatal(err)
	}
	if h == "rahasia123" || len(h) < 50 {
		t.Fatal("kata sandi harus di-hash, bukan disimpan apa adanya")
	}
	if !CheckPassword("rahasia123", h) {
		t.Fatal("kata sandi benar harus lolos")
	}
	if CheckPassword("rahasia124", h) {
		t.Fatal("kata sandi salah harus gagal")
	}
}

func TestOTP(t *testing.T) {
	code, hash, err := NewOTP()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 6 {
		t.Fatalf("kode harus 6 digit, dapat %q", code)
	}
	if code == hash {
		t.Fatal("kode tidak boleh disimpan apa adanya")
	}
	if !CheckOTP(code, hash) {
		t.Fatal("kode benar harus lolos")
	}
	wrong := "000000"
	if wrong == code {
		wrong = "111111"
	}
	if CheckOTP(wrong, hash) {
		t.Fatal("kode salah harus gagal")
	}
}

func TestInviteTokenStoredAsHash(t *testing.T) {
	plain, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) != 64 {
		t.Fatalf("token harus 32 byte hex, dapat %d karakter", len(plain))
	}
	if plain == hash {
		t.Fatal("yang disimpan harus hash, bukan token")
	}
	if HashToken(plain) != hash {
		t.Fatal("HashToken harus konsisten")
	}
	plain2, _, _ := NewToken()
	if plain2 == plain {
		t.Fatal("token harus acak, tidak pernah sama")
	}
}

func TestJWT(t *testing.T) {
	secret := "secret-uji"
	token, err := GenerateToken("user-1", "tenant-1", "RESIDENT", secret)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatalf("token sah harus bisa dibaca: %v", err)
	}
	if claims.UserID != "user-1" || claims.TenantID != "tenant-1" || claims.Role != "RESIDENT" {
		t.Fatalf("klaim tidak sesuai: %+v", claims)
	}
	if _, err := ParseToken(token, "secret-lain"); err == nil {
		t.Fatal("token dengan kunci berbeda harus ditolak")
	}
	if _, err := ParseToken(token+"x", secret); err == nil {
		t.Fatal("token yang diubah harus ditolak")
	}
	if _, err := GenerateToken("u", "t", "r", ""); err == nil {
		t.Fatal("JWT_SECRET kosong harus jadi error, bukan token lemah")
	}
}
