package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache — pembungkus tipis Redis. Bila Redis tidak terjangkau, seluruh operasi
// menjadi no-op (fail-open) dan API tetap melayani; kegagalannya dicatat.
type Cache struct {
	rdb *redis.Client
}

// New menyambung ke Redis. Kalau gagal, kembalikan Cache kosong supaya aplikasi
// tetap hidup — cache mati lebih baik daripada API mati.
func New(addr string) *Cache {
	if addr == "" {
		log.Println("peringatan: REDIS_ADDR kosong — cache dimatikan")
		return &Cache{}
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("peringatan: Redis %s tidak terjangkau (%v) — cache dimatikan", addr, err)
		_ = rdb.Close()
		return &Cache{}
	}
	log.Printf("Redis siap: %s", addr)
	return &Cache{rdb: rdb}
}

func (c *Cache) Enabled() bool { return c != nil && c.rdb != nil }

func (c *Cache) Client() *redis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Set menyimpan nilai dengan masa berlaku. TTL 0 = tanpa kedaluwarsa.
func (c *Cache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

// Get mengembalikan nilai; ok=false bila kunci tidak ada ATAU Redis mati.
func (c *Cache) Get(ctx context.Context, key string) (string, bool) {
	if !c.Enabled() {
		return "", false
	}
	v, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return v, true
}

func (c *Cache) Del(ctx context.Context, key string) error {
	if !c.Enabled() {
		return nil
	}
	return c.rdb.Del(ctx, key).Err()
}

// Incr menaikkan penghitung; masa berlaku disetel pada kenaikan pertama.
// Dipakai untuk membatasi jumlah percobaan (mis. OTP lupa sandi).
func (c *Cache) Incr(ctx context.Context, key string, ttl time.Duration) int64 {
	if !c.Enabled() {
		return 0
	}
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}
	if n == 1 && ttl > 0 {
		_ = c.rdb.Expire(ctx, key, ttl).Err()
	}
	return n
}
