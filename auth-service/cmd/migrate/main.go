package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	dbURL := os.Getenv("DB_SOURCE")
	if dbURL == "" {
		log.Fatal("DB_SOURCE environment variable is required")
	}

	//Cek argumen terminal (up/down)
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go [up|down]")
	}
	direction := os.Args[1]

	// Path ke folder migrasi
	migrationsPath := "file://migrations"

	m, err := migrate.New(
		migrationsPath,
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	var migrationErr error
	switch direction {
	case "up":
		log.Println("Running migration up...")
		migrationErr = m.Up()
	case "down":
		log.Println("Running migration down...")
		migrationErr = m.Down()
	default:
		log.Fatalf("Invalid direction: %s. Use 'up' or 'down'", direction)
	}

	if migrationErr != nil && migrationErr != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", migrationErr)
	}

	if migrationErr == migrate.ErrNoChange {
		log.Println("No new migrations to apply.")
		return
	}

	log.Printf("Migration %s completed successfully!", direction)
}
