// Cara Jalankan server: go run ./cmd/api
package main

import (
	"log"

	"auth-service/internal/auth"
	"auth-service/internal/user" //Perhaitkan go.mod untuk path module yang benar

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Muat .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables from docker-compose")
	}

	// Inisialisasi Database
	db := initDB()

	// Inisialisasi Lapis demi Lapis (Dependency Injection)
	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// Setup Gin Router
	router := gin.Default()

	// Grup untuk endpoint publik (tidak perlu login)
	api := router.Group("/api/v1")
	{
		api.POST("/register", userHandler.RegisterUser)
		api.POST("/login", userHandler.LoginUser)
	}

	// Grup untuk endpoint terproteksi (wajib login/bawa token)
	protected := router.Group("/api/v1")
	protected.Use(auth.AuthMiddleware()) // Terapkan middleware di sini!
	{
		protected.GET("/profile", userHandler.GetProfile)
	}

	// Jalankan server
	log.Println("Starting server on :8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
