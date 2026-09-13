package main

import (
	"context"
	"log"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/config"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
	delivery "github.com/nvnrchmn/smarthub-v3/backend/internal/delivery/http"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/crypto"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/notify"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/platform/storage"
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

	// Kunci enkripsi data pribadi (NIK/KK). Wajib: tanpa ini API tidak boleh
	// melayani karena data sensus harus tersimpan terenkripsi.
	if cfg.AESKey == "" {
		log.Fatal("AES_MASTER_KEY belum diisi")
	}
	ciph, err := crypto.NewCipher(cfg.AESKey)
	if err != nil {
		log.Fatalf("kunci enkripsi: %v", err)
	}
	store.Cipher = ciph

	// Penyimpanan dokumen privat (KTP/KK). Bila MinIO belum siap, sensus tetap
	// jalan tanpa fitur unggah (endpoint dokumen mengembalikan pesan jelas).
	var objStore *storage.Store
	if cfg.MinioAccess != "" && cfg.MinioSecret != "" {
		objStore, err = storage.New(cfg.MinioEndpoint, cfg.MinioAccess, cfg.MinioSecret, cfg.MinioBucketPII)
		if err != nil {
			log.Printf("peringatan: penyimpanan dokumen tidak aktif: %v", err)
			objStore = nil
		} else {
			log.Printf("penyimpanan dokumen siap: %s/%s", cfg.MinioEndpoint, cfg.MinioBucketPII)
		}
	}

	auth := &usecase.Auth{
		Store:     store,
		Notify:    notify.New(),
		JWTSecret: cfg.JWTSecret,
		BaseURL:   cfg.BaseURL,
	}

	census := &usecase.Census{Store: store, Storage: objStore}

	srv := &delivery.Server{Auth: auth, Census: census}

	log.Printf("smarthub-api listening on :%s", cfg.Port)
	log.Fatal(srv.Router().Listen(":" + cfg.Port))
}
