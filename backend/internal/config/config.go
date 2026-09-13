package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	RedisAddr   string
	JWTSecret   string
	BaseURL     string
	Env         string
	AESKey      string
	MinioEndpoint string
	MinioAccess   string
	MinioSecret   string
	MinioBucketPII string
}

func Load() Config {
	get := func(k, d string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return d
	}
	return Config{
		Port:        get("PORT", "8082"),
		DatabaseURL: get("DATABASE_URL", ""),
		RedisAddr:   get("REDIS_ADDR", "127.0.0.1:6379"),
		JWTSecret:   get("JWT_SECRET", ""),
		BaseURL:     get("BASE_URL", "https://smarthub.logikraf.id"),
		Env:         get("APP_ENV", "production"),
		AESKey:      get("AES_MASTER_KEY", ""),
		MinioEndpoint: get("MINIO_ENDPOINT", "127.0.0.1:9000"),
		MinioAccess:   get("MINIO_ACCESS_KEY", ""),
		MinioSecret:   get("MINIO_SECRET_KEY", ""),
		MinioBucketPII: get("MINIO_BUCKET_PII", "smarthub-pii"),
	}
}
