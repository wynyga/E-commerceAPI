// Command product is the catalogue service: products, prices and stock.
package main

import (
	"context"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/wynyga/ecommerce/services/product/internal/product"
	"github.com/wynyga/ecommerce/services/product/migrations"
	"github.com/wynyga/ecommerce/shared/config"
	"github.com/wynyga/ecommerce/shared/db"
	"github.com/wynyga/ecommerce/shared/httpx"
)

func main() {
	config.LoadDotEnv()
	dsn := config.Required("DB_SOURCE")
	jwtKey := []byte(config.Secret("JWT_SECRET"))
	internalKey := config.Secret("INTERNAL_KEY")
	port := config.String("PORT", "8080")

	conn, err := db.Open(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(dsn, migrations.FS, "."); err != nil {
		log.Fatal(err)
	}
	// TranslateError turns driver errors (duplicates, ...) into gorm's own error values.
	gdb, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: conn}), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)
	r.Get("/health", httpx.Health("product"))
	product.NewHandler(product.NewRepository(gdb)).Routes(r, jwtKey, internalKey)

	log.Printf("product service listening on :%s", port)
	log.Fatal(httpx.Serve(":"+port, r))
}
