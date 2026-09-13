package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nvnrchmn/smarthub-v3/backend/internal/db"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/pkg/security"
	"github.com/nvnrchmn/smarthub-v3/backend/internal/repository/postgres"
)

// smarthub-cli — tugas operasional yang sengaja TIDAK lewat API publik.
// Bootstrap tenant pertama memakai kredensial admin (ADMIN_DATABASE_URL) karena
// pendaftaran mandiri tidak disediakan (invite-only).
func main() {
	cmd := flag.String("cmd", "", "perintah: create-tenant")
	name := flag.String("name", "", "nama perumahan")
	slug := flag.String("slug", "", "slug unik, mis. griya-asri")
	email := flag.String("email", "", "email pengelola pertama")
	fullName := flag.String("full-name", "", "nama lengkap pengelola")
	password := flag.String("password", "", "kata sandi pengelola (min 8 karakter)")
	adminDSN := flag.String("dsn", os.Getenv("ADMIN_DATABASE_URL"), "DSN admin (env ADMIN_DATABASE_URL)")
	flag.Parse()

	if *cmd != "create-tenant" {
		log.Fatal("perintah tidak dikenal; pakai -cmd create-tenant")
	}
	if *name == "" || *slug == "" || *email == "" || len(*password) < 8 || *adminDSN == "" {
		log.Fatal("wajib: -name -slug -email -password(min 8) dan ADMIN_DATABASE_URL")
	}

	ctx := context.Background()
	if err := db.Connect(ctx, *adminDSN); err != nil {
		log.Fatalf("database admin: %v", err)
	}

	hash, err := security.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	if *fullName == "" {
		*fullName = *email
	}
	store := postgres.New(db.Pool)
	tenantID, userID, err := store.CreateTenantWithManager(ctx, *name, strings.ToLower(*slug), *email, hash, *fullName)
	if err != nil {
		log.Fatalf("gagal membuat tenant: %v", err)
	}
	fmt.Printf("tenant dibuat: %s (%s)\npengelola: %s\n", *name, tenantID, userID)
}
