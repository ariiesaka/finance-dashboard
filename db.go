package main

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func initDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return nil, fmt.Errorf("create users: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	)`); err != nil {
		return nil, fmt.Errorf("create sessions: %w", err)
	}

	return db, nil
}

func hasUser(db *sql.DB) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count > 0, err
}

func getPasswordHash(db *sql.DB) (string, error) {
	var hash string
	err := db.QueryRow("SELECT password_hash FROM users LIMIT 1").Scan(&hash)
	return hash, err
}

func storePassword(db *sql.DB, hash string) error {
	_, err := db.Exec("INSERT INTO users (password_hash) VALUES (?)", hash)
	return err
}
