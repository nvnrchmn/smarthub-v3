package main

import (
	"context"
	"log"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/config"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
	delivery "github.com/nvnrchmn/smarthub-v3/backend/internal/delivery/http"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/notify"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/usecase"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	if err := db.Connect(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("database: %v", err)
	}
	log.Println("database connected")

	store := postgres.New(db.Pool)
	auth := &usecase.Auth{
		Store:     store,
		Notify:    notify.New(),
		JWTSecret: cfg.JWTSecret,
		BaseURL:   cfg.BaseURL,
	}

	srv := &delivery.Server{Auth: auth}

	log.Printf("smarthub-api listening on :%s", cfg.Port)
	log.Fatal(srv.Router().Listen(":" + cfg.Port))
}
