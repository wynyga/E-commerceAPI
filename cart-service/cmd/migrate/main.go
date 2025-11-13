package main

import (
	"log"
	"os"

	// --- IMPORT KUNCI ---
	// 1. Driver database/sql standar untuk MySQL
	// Ini adalah bagian yang hilang dari skrip pertama Anda.
	_ "github.com/go-sql-driver/mysql"

	// 2. Library migrasi
	"github.com/golang-migrate/migrate/v4"
	// 3. Adapter "database" untuk migrate (agar tahu cara bicara "mysql")
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	// 4. Adapter "source" untuk migrate (agar tahu cara baca "file://")
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// 5. Library untuk memuat .env
	"github.com/joho/godotenv"
)

func main() {
	// Muat file .env dari direktori yang sama dengan tempat skrip dijalankan
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Gunakan variabel yang sama dengan skrip Anda yang berhasil (DB_SOURCE)
	dbURL := os.Getenv("DB_SOURCE")
	if dbURL == "" {
		log.Fatal("DB_SOURCE environment variable is required")
	}

	// Tentukan path ke folder migrasi
	// Ini berasumsi Anda menjalankan skrip dari root proyek
	// dan ada folder "migrations" di root tersebut.
	migrationsPath := "file://migrations"

	// Buat instance migrate
	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	// Cek argumen terminal (up/down)
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./path/to/main.go [up|down]")
	}
	direction := os.Args[1]

	// Jalankan migrasi berdasarkan argumen
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

	// Penanganan error yang lebih baik
	if migrationErr != nil && migrationErr != migrate.ErrNoChange {
		log.Fatalf("Migration failed: %v", migrationErr)
	}

	if migrationErr == migrate.ErrNoChange {
		log.Println("No new migrations to apply.")
		return
	}

	log.Printf("Migration %s completed successfully!", direction)
}
