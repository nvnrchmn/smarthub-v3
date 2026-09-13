package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/config"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	if err := db.Connect(ctx, cfg.DatabaseURL); err != nil {
		log.Fatalf("database: %v", err)
	}
	log.Println("database connected")

	app := fiber.New(fiber.Config{AppName: "Smarthub API"})

	// /api/health dipakai frontend lewat vhost (proxy /api/ ke service ini);
	// /health dipakai untuk cek langsung dari server.
	app.Get("/health", health)
	app.Get("/api/health", health)

	log.Printf("smarthub-api listening on :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

// health — laporan kesiapan service; DB diping setiap permintaan supaya status
// yang dilaporkan mencerminkan keadaan saat itu, bukan saat startup.
func health(c fiber.Ctx) error {
	dbOK := db.Pool.Ping(context.Background()) == nil
	code := fiber.StatusOK
	status := "ok"
	if !dbOK {
		code = fiber.StatusServiceUnavailable
		status = "degraded"
	}
	return c.Status(code).JSON(fiber.Map{
		"status":   status,
		"service":  "smarthub-api",
		"database": dbOK,
	})
}
