package main

import (
	"log"
	"net/http"
	"os"

	// Impor paket-paket kita
	"product-service/internal/product"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// Kode inisialisasi dan server di sini
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables from docker-compose")
	}

	db = InitDB()

	productRepo := product.NewRepository(db)
	productService := product.NewService(productRepo)
	productHandler := product.NewHandler(productService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK Pats"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/products", func(r chi.Router) {
			productHandler.RegisterRoutes(r)
		})
	})

	port := os.Getenv("API_PORT")
	log.Printf("Starting Product Service on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
