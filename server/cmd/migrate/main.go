package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var migrationsPath string
	var databaseURL string
	var command string

	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
	flag.StringVar(&databaseURL, "database", os.Getenv("DATABASE_URL"), "Database URL")
	flag.StringVar(&command, "command", "up", "Migration command: up, down, force, version")
	flag.Parse()

	if databaseURL == "" {
		databaseURL = "postgresql://postgres:postgres@localhost:5432/keycabinet?sslmode=disable"
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d, Dirty: %t\n", version, dirty)
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}
