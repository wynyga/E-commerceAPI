// Command auth is the account service: registration, login and token issuing.
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/wynyga/ecommerce/services/auth/internal/user"
	"github.com/wynyga/ecommerce/services/auth/migrations"
	"github.com/wynyga/ecommerce/shared/config"
	"github.com/wynyga/ecommerce/shared/db"
	"github.com/wynyga/ecommerce/shared/httpx"
)

func main() {
	config.LoadDotEnv()
	dsn := config.Required("DB_SOURCE")
	jwtKey := []byte(config.Secret("JWT_SECRET"))
	port := config.String("PORT", "8080")
	ttl := config.Duration("JWT_TTL", 24*time.Hour)
	loginLimit := config.Int("AUTH_RATE_PER_MINUTE", 10)

	ctx := context.Background()
	conn, err := db.Open(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(dsn, migrations.FS, "."); err != nil {
		log.Fatal(err)
	}

	svc := user.NewService(user.NewRepository(conn), jwtKey, ttl)

	// Bootstrap the first admin so a fresh deployment can manage the catalogue.
	if email := config.String("ADMIN_EMAIL", ""); email != "" {
		if err := svc.EnsureAdmin(ctx, email, config.Required("ADMIN_PASSWORD")); err != nil {
			log.Fatalf("seed admin: %v", err)
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)
	r.Get("/health", httpx.Health("auth"))
	r.Route("/api/v1", func(r chi.Router) {
		user.NewHandler(svc).Routes(r, jwtKey, loginLimit)
	})

	log.Printf("auth service listening on :%s", port)
	log.Fatal(httpx.Serve(":"+port, r))
}
