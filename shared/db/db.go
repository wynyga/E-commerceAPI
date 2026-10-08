// Package db opens MySQL connections and applies embedded SQL migrations.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// prepare parses a DSN ("user:pass@tcp(host:3306)/name", an optional mysql:// prefix is
// tolerated) and applies the settings every service needs.
func prepare(dsn string, multiStatements bool) (string, error) {
	cfg, err := mysql.ParseDSN(strings.TrimPrefix(dsn, "mysql://"))
	if err != nil {
		return "", fmt.Errorf("invalid DB_SOURCE: %w", err)
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.MultiStatements = multiStatements
	return cfg.FormatDSN(), nil
}

// Open connects to MySQL and waits up to 60 seconds for it to accept connections, so
// services can start in any order under docker compose.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	full, err := prepare(dsn, false)
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("mysql", full)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	deadline := time.Now().Add(60 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = conn.PingContext(pingCtx)
		cancel()
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			conn.Close()
			return nil, fmt.Errorf("database not reachable: %w", err)
		}
		log.Printf("waiting for database: %v", err)
		time.Sleep(time.Second)
	}
}

// Migrate applies every pending migration under dir in fsys. It uses its own short-lived
// connection (multi-statement files need multiStatements=true), so the app pool is untouched.
func Migrate(dsn string, fsys fs.FS, dir string) error {
	full, err := prepare(dsn, true)
	if err != nil {
		return err
	}
	conn, err := sql.Open("mysql", full)
	if err != nil {
		return err
	}
	driver, err := migratemysql.WithInstance(conn, &migratemysql.Config{})
	if err != nil {
		conn.Close()
		return err
	}
	src, err := iofs.New(fsys, dir)
	if err != nil {
		conn.Close()
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "mysql", driver)
	if err != nil {
		conn.Close()
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// IsDuplicate reports whether err is MySQL's "duplicate entry" violation (error 1062).
func IsDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}
