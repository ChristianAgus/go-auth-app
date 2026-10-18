package main

import (
	"log"
	"os"

	"auth-app/internal/config"
	"auth-app/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("pemakaian: go run ./cmd/migrate <up|down|reset|status>")
	}

	cfg := config.Load()

	db, err := database.NewMySQL(cfg.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Migrate(db, os.Args[1]); err != nil {
		log.Fatal(err)
	}
}