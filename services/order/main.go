// Command order turns a cart into an order and keeps stock consistent while doing so.
package main

import (
	"context"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/wynyga/ecommerce/services/order/internal/clients"
	"github.com/wynyga/ecommerce/services/order/internal/order"
	"github.com/wynyga/ecommerce/services/order/migrations"
	"github.com/wynyga/ecommerce/shared/config"
	"github.com/wynyga/ecommerce/shared/db"
	"github.com/wynyga/ecommerce/shared/httpx"
)

func main() {
	config.LoadDotEnv()
	dsn := config.Required("DB_SOURCE")
	jwtKey := []byte(config.Secret("JWT_SECRET"))
	internalKey := config.Secret("INTERNAL_KEY")
	productURL := config.Required("PRODUCT_URL")
	cartURL := config.Required("CART_URL")
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

	svc := order.NewService(order.NewRepository(gdb), clients.NewCatalog(productURL, internalKey), clients.NewCarts(cartURL))

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Logger, middleware.Recoverer)
	r.Get("/health", httpx.Health("order"))
	order.NewHandler(svc).Routes(r, jwtKey)

	log.Printf("order service listening on :%s", port)
	log.Fatal(httpx.Serve(":"+port, r))
}
