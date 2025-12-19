package main

import (
	"log"
	"os"
	"strings"

	// Library untuk Migrasi
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// Library GORM
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	dsn := os.Getenv("DB_SOURCE")
	if dsn == "" {
		log.Fatal("DB_SOURCE not set in environment")
	}

	// Menghapus prefix jika ada (agar kompatibel dengan kode Anda)
	gormDSN := strings.TrimPrefix(dsn, "mysql://")

	db, err := gorm.Open(gormMysql.Open(gormDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	log.Println("Database connection established via GORM")

	// Panggil fungsi migrasi
	runMigrations(db)

	return db
}

func runMigrations(db *gorm.DB) {
	// 1. GORM menggunakan *gorm.DB, tapi migrate butuh *sql.DB (standar Go)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Could not get sql.DB from gorm.DB:", err)
	}

	// 2. Buat driver migrasi untuk MySQL
	driver, err := mysql.WithInstance(sqlDB, &mysql.Config{})
	if err != nil {
		log.Fatal("Could not create migration driver:", err)
	}

	// 3. Inisialisasi instance migrasi
	// Pastikan folder 'migrations' ada di root project container
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"mysql",
		driver,
	)
	if err != nil {
		log.Fatal("Could not initialize migration instance:", err)
	}

	// 4. Jalankan migrasi UP
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to run migrations:", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("Database schema is up to date (no changes)")
	} else {
		log.Println("Database migrated successfully!")
	}
}
