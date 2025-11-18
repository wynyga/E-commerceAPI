package main

import (
	"log"
	"net/http"
	"os"

	// Impor paket-paket kita
	"cart-service/internal/auth"
	"cart-service/internal/cart"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables from docker-compose")
	}

	//Inisialisasi Database
	db = InitDB()

	//Inisialisasi arsitektur layanan
	cartRepo := cart.NewRepository(db)
	cartService := cart.NewService(cartRepo)
	cartHandler := cart.NewHandler(cartService)

	//Inisialisasi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger) // Middleware untuk logging request

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK Pats"))
	})

	// --- Grup Rute API v1 ---
	// Kita buat grup rute baru untuk /api/v1
	r.Route("/api/v1", func(r chi.Router) {

		// Grup rute untuk keranjang (/api/v1/cart)
		// Kita terapkan AuthMiddleware di sini!
		r.Route("/cart", func(r chi.Router) {
			r.Use(auth.AuthMiddleware) // Penjaga Gerbang!

			// Daftarkan rute-rute dari handler kita
			cartHandler.RegisterRoutes(r)
		})

		// (Nanti jika ada service 'product', bisa ditambahkan di sini)
		// r.Route("/products", func(r chi.Router) { ... })
	})

	// Muali server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
