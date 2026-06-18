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

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS debts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		total_amount REAL NOT NULL,
		remaining REAL NOT NULL,
		interest_rate REAL DEFAULT 0,
		priority TEXT DEFAULT 'medium',
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return nil, fmt.Errorf("create debts: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS debt_payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		debt_id INTEGER NOT NULL,
		amount REAL NOT NULL,
		date TEXT NOT NULL,
		notes TEXT DEFAULT '',
		FOREIGN KEY (debt_id) REFERENCES debts(id)
	)`); err != nil {
		return nil, fmt.Errorf("create debt_payments: %w", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS goals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		target_amount REAL NOT NULL,
		current_amount REAL DEFAULT 0,
		priority TEXT DEFAULT 'medium',
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return nil, fmt.Errorf("create goals: %w", err)
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

func getExpenseDateRange(db *sql.DB) (string, string, error) {
	var minDate, maxDate string
	err := db.QueryRow(
		"SELECT COALESCE(MIN(date), ''), COALESCE(MAX(date), '') FROM transactions",
	).Scan(&minDate, &maxDate)
	return minDate, maxDate, err
}

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

func listExpensesForRange(db *sql.DB, startDate, endDate string) ([]Transaction, error) {
	rows, err := db.Query(
		`SELECT id, date, COALESCE(time,''), category, amount, merchant,
		        COALESCE(account,'Cash'), COALESCE(method,''), COALESCE(notes,'')
		 FROM transactions
		 WHERE date >= ? AND date <= ?
		 ORDER BY date DESC, id DESC`,
		startDate, endDate,
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

func categoryBreakdownForRange(db *sql.DB, startDate, endDate string) ([]CategoryBreakdown, error) {
	rows, err := db.Query(
		`SELECT category, SUM(amount) as total, COUNT(*) as count
		 FROM transactions
		 WHERE date >= ? AND date <= ?
		 GROUP BY category
		 ORDER BY total DESC`,
		startDate, endDate,
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

func totalExpensesForRange(db *sql.DB, startDate, endDate string) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE date >= ? AND date <= ?",
		startDate, endDate,
	).Scan(&total)
	return total, err
}

func updateExpense(db *sql.DB, t Transaction) error {
	res, err := db.Exec(
		`UPDATE transactions SET date=?, category=?, amount=?, merchant=?
		 WHERE id=?`,
		t.Date, t.Category, t.Amount, t.Merchant, t.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("expense not found")
	}
	return nil
}

// ─── Debt Queries ───────────────────────────────────────────

func createDebt(db *sql.DB, d Debt) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO debts (name, description, total_amount, remaining, interest_rate, priority, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		d.Name, d.Description, d.TotalAmount, d.Remaining, d.InterestRate, d.Priority, d.Notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func listDebts(db *sql.DB) ([]Debt, error) {
	rows, err := db.Query(
		`SELECT id, name, COALESCE(description,''), total_amount, remaining,
		        COALESCE(interest_rate,0), COALESCE(priority,'medium'), COALESCE(notes,''),
		        COALESCE(created_at,'')
		 FROM debts ORDER BY priority DESC, remaining DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var debts []Debt
	for rows.Next() {
		var d Debt
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.TotalAmount,
			&d.Remaining, &d.InterestRate, &d.Priority, &d.Notes, &d.CreatedAt); err != nil {
			return nil, err
		}
		debts = append(debts, d)
	}
	if debts == nil {
		debts = []Debt{}
	}
	return debts, rows.Err()
}

func deleteDebt(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM debt_payments WHERE debt_id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM debts WHERE id = ?", id)
	return err
}

func totalPaidForDebt(db *sql.DB, debtID int) (float64, error) {
	var total float64
	err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM debt_payments WHERE debt_id = ?",
		debtID,
	).Scan(&total)
	return total, err
}

func createDebtPayment(db *sql.DB, p DebtPayment) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`INSERT INTO debt_payments (debt_id, amount, date, notes) VALUES (?, ?, ?, ?)`,
		p.DebtID, p.Amount, p.Date, p.Notes,
	)
	if err != nil {
		return 0, err
	}

	r, err := tx.Exec(
		"UPDATE debts SET remaining = remaining - ? WHERE id = ? AND remaining >= ?",
		p.Amount, p.DebtID, p.Amount,
	)
	if err != nil {
		return 0, err
	}

	rows, _ := r.RowsAffected()
	if rows == 0 {
		return 0, fmt.Errorf("payment exceeds remaining balance")
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func listDebtPayments(db *sql.DB, debtID int) ([]DebtPayment, error) {
	rows, err := db.Query(
		`SELECT id, debt_id, amount, date, COALESCE(notes,'') FROM debt_payments
		 WHERE debt_id = ? ORDER BY date DESC, id DESC`, debtID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []DebtPayment
	for rows.Next() {
		var p DebtPayment
		if err := rows.Scan(&p.ID, &p.DebtID, &p.Amount, &p.Date, &p.Notes); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	if payments == nil {
		payments = []DebtPayment{}
	}
	return payments, rows.Err()
}

// ─── Goal Queries ───────────────────────────────────────────

func createGoal(db *sql.DB, g Goal) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO goals (name, target_amount, current_amount, priority, notes)
		 VALUES (?, ?, ?, ?, ?)`,
		g.Name, g.TargetAmount, g.CurrentAmount, g.Priority, g.Notes,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func listGoals(db *sql.DB) ([]Goal, error) {
	rows, err := db.Query(
		`SELECT id, name, target_amount, current_amount,
		        COALESCE(priority,'medium'), COALESCE(notes,''), COALESCE(created_at,'')
		 FROM goals ORDER BY priority DESC, target_amount ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.Name, &g.TargetAmount,
			&g.CurrentAmount, &g.Priority, &g.Notes, &g.CreatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	if goals == nil {
		goals = []Goal{}
	}
	return goals, rows.Err()
}

func updateGoalProgress(db *sql.DB, id int, amount float64) error {
	_, err := db.Exec("UPDATE goals SET current_amount = ? WHERE id = ?", amount, id)
	return err
}

func deleteGoal(db *sql.DB, id int) error {
	_, err := db.Exec("DELETE FROM goals WHERE id = ?", id)
	return err
}
