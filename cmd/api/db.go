package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql" // Driver
)

func initDB() *sql.DB {
	dbURL := os.Getenv("DB_SOURCE")
	if dbURL == "" {
		log.Fatal("DB_SOURCE environment variable is not set")
	}

	db, err := sql.Open("mysql", dbURL[8:])
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}
	return db
}
