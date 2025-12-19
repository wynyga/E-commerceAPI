package main

import (
	"database/sql"
	"log"
	"os"
	"strings"

	// Driver MySQL
	_ "github.com/go-sql-driver/mysql"

	// Library untuk Migrasi
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func initDB() *sql.DB {
	dbURL := os.Getenv("DB_SOURCE")
	if dbURL == "" {
		log.Fatal("DB_SOURCE environment variable is not set")
	}

	// Gunakan strings.TrimPrefix agar lebih aman daripada [8:]
	dsn := strings.TrimPrefix(dbURL, "mysql://")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Database connection established (standard sql)")

	// Jalankan migrasi otomatis
	runMigrations(db)

	return db
}

func runMigrations(db *sql.DB) {
	// Buat driver migrasi khusus MySQL
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		log.Fatalf("Could not create migration driver: %v", err)
	}

	// Inisialisasi instance migrasi (mencari folder migrations di root)
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"mysql",
		driver,
	)
	if err != nil {
		log.Fatalf("Could not initialize migration instance: %v", err)
	}

	// Jalankan migrasi UP
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("Auth-service database schema is up to date")
	} else {
		log.Println("Auth-service database migrated successfully!")
	}
}
