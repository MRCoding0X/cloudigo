// Command bootstrapadmin creates the first admin account for a Cloudigo instance.
// It is meant to be run once, manually, after migrations have applied.
//
// Usage:
//
//	go run ./cmd/bootstrapadmin -email admin@example.com -password "at-least-8-chars"
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"cloudigo/backend/internal/config"
	"cloudigo/backend/internal/db"
	"cloudigo/backend/internal/model"
	"cloudigo/backend/internal/repository"
	"cloudigo/backend/internal/service"
)

func main() {
	email := flag.String("email", "", "admin email address")
	password := flag.String("password", "", "admin password (min 8 chars)")
	flag.Parse()

	if *email == "" || len(*password) < 8 {
		fmt.Fprintln(os.Stderr, "usage: bootstrapadmin -email <email> -password <min 8 chars>")
		os.Exit(1)
	}

	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is not set (see .env.example)")
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)

	if _, err := userRepo.GetByEmail(ctx, *email); err == nil {
		log.Fatalf("a user with email %q already exists", *email)
	} else if !errors.Is(err, repository.ErrNotFound) {
		log.Fatalf("lookup failed: %v", err)
	}

	hash, err := service.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	user, err := userRepo.Create(ctx, *email, hash, model.RoleAdmin, "")
	if err != nil {
		log.Fatalf("create admin: %v", err)
	}

	log.Printf("admin account created: %s (id=%s)", user.Email, user.ID)
}
