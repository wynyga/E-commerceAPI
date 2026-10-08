// Command cart is the shopping-cart service.
package main

import (
	"context"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/wynyga/ecommerce/services/cart/internal/cart"
	"github.com/wynyga/ecommerce/services/cart/internal/catalog"
	"github.com/wynyga/ecommerce/services/cart/migrations"
	"github.com/wynyga/ecommerce/shared/config"
	"github.com/wynyga/ecommerce/shared/db"
	"github.com/wynyga/ecommerce/shared/httpx"
)

func main() {
	config.LoadDotEnv()
	dsn := config.Required("DB_SOURCE")
	jwtKey := []byte(config.Secret("JWT_SECRET"))
	productURL := config.Required("PRODUCT_URL")
	port := config.String("PORT", "8080")

	conn, err := db.Open(context.Background(), dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if err := db.Migrate(dsn, migrations.FS, "."); err != nil {
		log.Fatal(err)
	}
	gdb, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: conn}), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal(err)
	}

	svc := cart.NewService(cart.NewRepository(gdb), catalog.New(productURL))

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)
	r.Get("/health", httpx.Health("cart"))
	cart.NewHandler(svc).Routes(r, jwtKey)

	log.Printf("cart service listening on :%s", port)
	log.Fatal(httpx.Serve(":"+port, r))
}
