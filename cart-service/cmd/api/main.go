package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	//Inisialisasi Database
	db = InitDB()

	//Inisialisasi Router
	r := chi.NewRouter()
	r.Use(middleware.Logger) // Middleware untuk logging request

	// Endpoint sederhana untuk Health check
	// Health Check berfungsi untuk memastikan service berjalan dengan baik
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK Pats"))
	})

	// Muali server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8081" //Default port
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
