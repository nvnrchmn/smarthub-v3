package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	RedisAddr   string
	JWTSecret   string
	Env         string
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
		Env:         get("APP_ENV", "production"),
	}
}
