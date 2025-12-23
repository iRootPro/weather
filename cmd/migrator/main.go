package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/iRootPro/weather/internal/config"
)

const migrationsDir = "migrations"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := args[0]

	db, err := sql.Open("pgx", cfg.DB.DSN())
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set dialect: %v", err)
	}

	switch command {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}
	case "down":
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("failed to rollback migration: %v", err)
		}
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("failed to get status: %v", err)
		}
	case "reset":
		if err := goose.Reset(db, migrationsDir); err != nil {
			log.Fatalf("failed to reset migrations: %v", err)
		}
	case "version":
		if err := goose.Version(db, migrationsDir); err != nil {
			log.Fatalf("failed to get version: %v", err)
		}
	default:
		fmt.Printf("unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrator <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up       Apply all available migrations")
	fmt.Println("  down     Rollback the last migration")
	fmt.Println("  status   Show migration status")
	fmt.Println("  reset    Rollback all migrations")
	fmt.Println("  version  Show current migration version")
}
