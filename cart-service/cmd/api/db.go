package main

import (
	"log"
	"os"
	"strings" // <-- 1. Import package "strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	// 2. Ambil DSN (ini akan berisi "mysql://root:@...")
	dsn := os.Getenv("DB_SOURCE")

	if dsn == "" {
		// Kita ubah dari log.Fatal menjadi log.Println
		// agar tidak crash di Docker jika .env tidak ada
		log.Println("DB_SOURCE not set in .env file, relying on docker-compose")
	}

	// 3. --- PERBAIKAN KUNCI ---
	//    Secara eksplisit hapus awalan "mysql://" yang
	//    dibutuhkan oleh 'migrate' tapi dibenci oleh 'GORM'.
	gormDSN := strings.TrimPrefix(dsn, "mysql://")
	//    Sekarang gormDSN = "root:@tcp..." (format yang benar untuk GORM)

	// 4. Gunakan DSN yang sudah bersih untuk GORM
	db, err := gorm.Open(mysql.Open(gormDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established")
	return db
}
