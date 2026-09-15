package cache

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// Storage mengimplementasikan fiber.Storage di atas Redis, sehingga pembatas
// laju (rate limiter) tidak lagi hilang setiap service di-restart.
type Storage struct {
	rdb *redis.Client
}

// NewStorage mengembalikan tipe INTERFACE (bukan *Storage) dengan sengaja:
// pointer nil yang dimasukkan ke interface tidak dianggap nil oleh Go,
// sehingga limiter akan memanggil metode pada objek kosong dan panic.
func NewStorage(c *Cache) fiber.Storage {
	if c == nil || c.rdb == nil {
		return nil
	}
	return &Storage{rdb: c.rdb}
}

// Bukti saat kompilasi: bila antarmuka fiber.Storage berubah, build gagal di
// baris ini — bukan diam-diam kehilangan fungsi di runtime.
var _ fiber.Storage = (*Storage)(nil)

// Semua kunci pembatas laju diberi awalan supaya Reset() tidak menyentuh
// kunci lain (mis. daftar-tolak token) di basis data Redis yang sama.
const prefiksLaju = "rl:"

func (s *Storage) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	v, err := s.rdb.Get(ctx, prefiksLaju+key).Bytes()
	if err == redis.Nil {
		return nil, nil // kontrak fiber.Storage: kunci tak ada = (nil, nil)
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Storage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *Storage) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	if key == "" || len(val) == 0 {
		return nil
	}
	return s.rdb.Set(ctx, prefiksLaju+key, val, exp).Err()
}

func (s *Storage) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *Storage) DeleteWithContext(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, prefiksLaju+key).Err()
}

func (s *Storage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

// Reset hanya menghapus kunci berawalan rl: — memakai SCAN, bukan FLUSHDB,
// supaya kunci lain di basis data Redis yang sama tidak ikut terhapus.
func (s *Storage) ResetWithContext(ctx context.Context) error {
	iter := s.rdb.Scan(ctx, 0, prefiksLaju+"*", 200).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (s *Storage) Reset() error {
	return s.ResetWithContext(context.Background())
}

// Close sengaja tidak menutup koneksi: koneksinya milik Cache dan masih
// dipakai komponen lain (daftar-tolak token, OTP lupa sandi).
func (s *Storage) Close() error { return nil }
