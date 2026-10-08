// Package config reads service configuration from environment variables.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads a local .env file when present. Variables already set in the
// environment win, so containers and CI are never overridden.
func LoadDotEnv() {
	_ = godotenv.Load()
}

func String(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Required exits the process when the variable is missing.
func Required(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("config: %s is required", key)
	}
	return v
}

// Secret is Required plus a minimum length, for signing keys.
func Secret(key string) string {
	v := Required(key)
	if len(v) < 32 {
		log.Fatalf("config: %s must be at least 32 characters", key)
	}
	return v
}

func Int(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		log.Fatalf("config: %s must be an integer, got %q", key, v)
	}
	return n
}

func Duration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(v))
	if err != nil {
		log.Fatalf("config: %s must be a duration like 15m, got %q", key, v)
	}
	return d
}
