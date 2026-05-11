package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}
	if err = migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Println("[DB] SQLite initialized:", dbPath)
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
