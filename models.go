package main

import "time"

type User struct {
	ID           int       `json:"id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type LoginRequest struct {
	Password string `json:"password"`
}

type APIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ─── Expense Models ─────────────────────────────────────────

type Transaction struct {
	ID          int     `json:"id"`
	Date        string  `json:"date"`
	Time        string  `json:"time,omitempty"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Merchant    string  `json:"merchant"`
	Account     string  `json:"account,omitempty"`
	Method      string  `json:"method,omitempty"`
	Notes       string  `json:"notes,omitempty"`
}

type CreateExpenseRequest struct {
	Date     string  `json:"date"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Merchant string  `json:"merchant"`
	Account  string  `json:"account,omitempty"`
	Method   string  `json:"method,omitempty"`
	Notes    string  `json:"notes,omitempty"`
}

type CategoryBreakdown struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

type MonthSummary struct {
	TotalExpenses float64            `json:"total_expenses"`
	Categories    []CategoryBreakdown `json:"categories"`
	Transactions  []Transaction      `json:"transactions"`
}
