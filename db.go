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

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		time TEXT,
		category TEXT NOT NULL DEFAULT 'Other',
		amount REAL NOT NULL,
		merchant TEXT NOT NULL,
		account TEXT DEFAULT 'Cash',
		method TEXT,
		notes TEXT
	)`); err != nil {
		return nil, fmt.Errorf("create transactions: %w", err)
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

// ─── Expense Queries ─────────────────────────────────────────

func createExpense(db *sql.DB, txn Transaction) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO transactions (date, time, category, amount, merchant, account, method, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		txn.Date, txn.Time, txn.Category, txn.Amount, txn.Merchant,
		txn.Account, txn.Method, txn.Notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func listExpensesForMonth(db *sql.DB, yearMonth string) ([]Transaction, error) {
	rows, err := db.Query(
		`SELECT id, date, COALESCE(time,''), category, amount, merchant,
		        COALESCE(account,'Cash'), COALESCE(method,''), COALESCE(notes,'')
		 FROM transactions
		 WHERE date LIKE ?
		 ORDER BY date DESC, id DESC`,
		yearMonth+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Date, &t.Time, &t.Category, &t.Amount,
			&t.Merchant, &t.Account, &t.Method, &t.Notes); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	if txns == nil {
		txns = []Transaction{}
	}
	return txns, rows.Err()
}

func expenseCategoryBreakdown(db *sql.DB, yearMonth string) ([]CategoryBreakdown, error) {
	rows, err := db.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count
		 FROM transactions
		 WHERE date LIKE ?
		 GROUP BY category
		 ORDER BY total DESC`,
		yearMonth+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []CategoryBreakdown
	for rows.Next() {
		var c CategoryBreakdown
		if err := rows.Scan(&c.Category, &c.Total, &c.Count); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []CategoryBreakdown{}
	}
	return cats, rows.Err()
}

func totalExpensesForMonth(db *sql.DB, yearMonth string) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE date LIKE ?",
		yearMonth+"%",
	).Scan(&total)
	return total, err
}
