package database

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"auth-app/migrations"
)

func Migrate(db *sql.DB, command string) error {
	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}

	switch command {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".") // rollback 1 migration terakhir
	case "reset":
		return goose.DownTo(db, ".", 0) // rollback semuanya
	case "status":
		return goose.Status(db, ".")
	default:
		return fmt.Errorf("command tidak dikenal: %q (pakai: up, down, reset, status)", command)
	}
}