package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func getDBPath() string {
	dataDir := filepath.Join(".", "data")
	return filepath.Join(dataDir, "finances.db")
}

func initDB() error {
	dataDir := filepath.Join(".", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	dbPath := getDBPath()
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrent reads
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}

	if err := runMigrations(); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	return nil
}

func dbExists() bool {
	dbPath := getDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		return true
	}
	return false
}

func runMigrations() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		date TEXT NOT NULL,
		time TEXT,
		type TEXT NOT NULL DEFAULT 'expense',
		category TEXT DEFAULT 'Other',
		amount REAL NOT NULL,
		currency TEXT DEFAULT 'IDR',
		merchant TEXT,
		account TEXT,
		method TEXT,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS income (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		amount REAL NOT NULL,
		source TEXT,
		notes TEXT
	);

	CREATE TABLE IF NOT EXISTS debts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		total_amount REAL NOT NULL,
		remaining REAL NOT NULL,
		paid_off INTEGER DEFAULT 0,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS debt_payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		debt_id INTEGER REFERENCES debts(id),
		amount REAL NOT NULL,
		date TEXT NOT NULL,
		notes TEXT
	);

	CREATE TABLE IF NOT EXISTS wishlist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		target_amount REAL NOT NULL,
		saved_amount REAL DEFAULT 0,
		priority INTEGER DEFAULT 0,
		achieved INTEGER DEFAULT 0,
		notes TEXT
	);
	`

	_, err := db.Exec(schema)
	return err
}

func getPasswordHash() (string, error) {
	var hash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE id = 1").Scan(&hash)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no user found")
	}
	return hash, err
}

func storePasswordHash(hash string) error {
	_, err := db.Exec("INSERT INTO users (id, password_hash) VALUES (1, ?)", hash)
	return err
}

func storeSession(id string, expiresAt string) error {
	_, err := db.Exec("INSERT INTO sessions (id, expires_at) VALUES (?, ?)", id, expiresAt)
	return err
}

func getSession(id string) (*Session, error) {
	s := &Session{}
	err := db.QueryRow("SELECT id, created_at, expires_at FROM sessions WHERE id = ?", id).Scan(&s.ID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func deleteSession(id string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func cleanupExpiredSessions() {
	db.Exec("DELETE FROM sessions WHERE expires_at < datetime('now')")
}

func getSummary(month string, year string) (*SummaryResponse, error) {
	resp := &SummaryResponse{}

	// Income this month
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM income WHERE strftime('%m', date) = ? AND strftime('%Y', date) = ?",
		month, year).Scan(&resp.IncomeThisMonth)

	// Expenses this month
	db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense' AND strftime('%m', date) = ? AND strftime('%Y', date) = ?",
		month, year).Scan(&resp.ExpensesThisMonth)

	resp.Balance = resp.IncomeThisMonth - resp.ExpensesThisMonth

	// Category breakdown
	rows, err := db.Query(`
		SELECT COALESCE(category, 'Other'), COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE type = 'expense' AND strftime('%m', date) = ? AND strftime('%Y', date) = ?
		GROUP BY category
		ORDER BY SUM(amount) DESC
	`, month, year)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c CategoryBreakdown
			rows.Scan(&c.Category, &c.Total)
			resp.Categories = append(resp.Categories, c)
		}
	}

	// Debts
	debtRows, err := db.Query("SELECT id, name, total_amount, remaining FROM debts WHERE paid_off = 0 ORDER BY remaining DESC")
	if err == nil {
		defer debtRows.Close()
		for debtRows.Next() {
			var d DebtSummary
			debtRows.Scan(&d.ID, &d.Name, &d.TotalAmount, &d.Remaining)
			if d.TotalAmount > 0 {
				d.Percentage = (d.TotalAmount - d.Remaining) / d.TotalAmount * 100
			}
			resp.Debts = append(resp.Debts, d)
		}
	}

	// Wishlist
	wishRows, err := db.Query("SELECT id, name, target_amount, saved_amount, priority, achieved FROM wishlist ORDER BY priority ASC")
	if err == nil {
		defer wishRows.Close()
		for wishRows.Next() {
			var w WishlistItem
			wishRows.Scan(&w.ID, &w.Name, &w.TargetAmount, &w.SavedAmount, &w.Priority, &w.Achieved)
			resp.Wishlist = append(resp.Wishlist, w)
		}
	}

	return resp, nil
}

func getTransactions(month string, limit int) ([]Transaction, error) {
	query := "SELECT id, date, COALESCE(time,''), type, COALESCE(category,'Other'), amount, COALESCE(currency,'IDR'), COALESCE(merchant,''), COALESCE(account,''), COALESCE(method,'') FROM transactions"
	args := []interface{}{}

	if month != "" {
		query += " WHERE strftime('%Y-%m', date) = ?"
		args = append(args, month)
	}

	query += " ORDER BY date DESC, id DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		rows.Scan(&t.ID, &t.Date, &t.Time, &t.Type, &t.Category, &t.Amount, &t.Currency, &t.Merchant, &t.Account, &t.Method)
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func getIncome(year, month string) ([]Income, error) {
	rows, err := db.Query("SELECT id, date, amount, COALESCE(source,''), COALESCE(notes,'') FROM income WHERE strftime('%Y', date) = ? AND strftime('%m', date) = ? ORDER BY date DESC",
		year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incomes []Income
	for rows.Next() {
		var i Income
		rows.Scan(&i.ID, &i.Date, &i.Amount, &i.Source, &i.Notes)
		incomes = append(incomes, i)
	}
	return incomes, nil
}

func getDebts() ([]Debt, error) {
	rows, err := db.Query("SELECT id, name, total_amount, remaining, paid_off, COALESCE(notes,'') FROM debts ORDER BY paid_off ASC, created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []Debt
	for rows.Next() {
		var d Debt
		rows.Scan(&d.ID, &d.Name, &d.TotalAmount, &d.Remaining, &d.PaidOff, &d.Notes)
		debts = append(debts, d)
	}
	return debts, nil
}

func getWishlist() ([]WishlistItem, error) {
	rows, err := db.Query("SELECT id, name, target_amount, saved_amount, priority, achieved FROM wishlist ORDER BY priority ASC, id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WishlistItem
	for rows.Next() {
		var w WishlistItem
		rows.Scan(&w.ID, &w.Name, &w.TargetAmount, &w.SavedAmount, &w.Priority, &w.Achieved)
		items = append(items, w)
	}
	return items, nil
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}

// initDBLogger is called from main to set up logging
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
