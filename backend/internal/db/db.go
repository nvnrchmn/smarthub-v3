package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func Connect(ctx context.Context, url string) error {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return err
	}
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	Pool = p
	return p.Ping(ctx)
}
