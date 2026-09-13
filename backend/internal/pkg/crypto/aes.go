package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

// Cipher — AES-256-GCM untuk data pribadi (NIK, No. KK).
//
// AAD berisi tenant_id: ciphertext tenant A tidak bisa dipakai di tenant B
// (copy-paste attack antar perumahan akan gagal saat dekripsi).
type Cipher struct {
	key []byte
}

func NewCipher(hexKey string) (*Cipher, error) {
	k, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	if len(k) != 32 {
		return nil, errors.New("AES_MASTER_KEY harus 32 byte (64 karakter hex)")
	}
	return &Cipher{key: k}, nil
}

func (c *Cipher) Encrypt(plaintext, tenantID string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plaintext), []byte(tenantID)), nil
}

func (c *Cipher) Decrypt(data []byte, tenantID string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext terlalu pendek")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, []byte(tenantID))
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
