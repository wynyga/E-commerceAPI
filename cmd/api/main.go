package main

import (
	"e-commerce/internal/user"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Muat .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Inisialisasi Database
	db := initDB()
	defer db.Close()

	// Inisialisasi Lapis demi Lapis (Dependency Injection)
	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// Setup Gin Router
	router := gin.Default()

	// Grup routing untuk API
	api := router.Group("/api/v1")
	{
		// Endpoint registrasi
		api.POST("/register", userHandler.RegisterUser)
	}

	// Jalankan server
	log.Println("Starting server on :8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
