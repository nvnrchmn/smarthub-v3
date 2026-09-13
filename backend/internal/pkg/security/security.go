package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword — bcrypt (sesuai PRD: Argon2/Bcrypt).
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(pw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// NewOTP — kode 6 digit acak kriptografis + hash bcrypt untuk disimpan.
// Kode aslinya hanya dikirim ke pemilik nomor, tidak pernah disimpan.
func NewOTP() (code string, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", err
	}
	code = fmt.Sprintf("%06d", n.Int64())
	hash, err = HashPassword(code)
	return code, hash, err
}

func CheckOTP(code, hash string) bool {
	return CheckPassword(code, hash)
}

// NewToken — token acak untuk tautan undangan. Yang disimpan di database hanya
// hash SHA-256-nya, sehingga bocornya isi tabel tidak memberi tautan yang bisa dipakai.
func NewToken() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	return plain, HashToken(plain), nil
}

func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
