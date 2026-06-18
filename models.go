package main

import "time"

// ─── Domain Models ──────────────────────────────────────────

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

type Transaction struct {
	// ID is a string because transactions come from Gmail with alphanumeric
	// message IDs (e.g. "bca_cc_25062"). Other entities use auto-increment int IDs.
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Time        string  `json:"time,omitempty"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Merchant    string  `json:"merchant"`
	Account     string  `json:"account,omitempty"`
	Method      string  `json:"method,omitempty"`
	Description string  `json:"description,omitempty"`
}

type Income struct {
	ID     int     `json:"id"`
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Source string  `json:"source"`
	Notes  string  `json:"notes,omitempty"`
}

type Debt struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	TotalAmount float64   `json:"total_amount"`
	Remaining   float64   `json:"remaining"`
	PaidOff     bool      `json:"paid_off"`
	Notes       string    `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type DebtPayment struct {
	ID      int     `json:"id"`
	DebtID  int     `json:"debt_id"`
	Amount  float64 `json:"amount"`
	Date    string  `json:"date"`
	Notes   string  `json:"notes,omitempty"`
}

type WishlistItem struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	SavedAmount  float64 `json:"saved_amount"`
	Priority     int     `json:"priority"`
	Achieved     bool    `json:"achieved"`
	Notes        string  `json:"notes,omitempty"`
}

// ─── Request Payloads ───────────────────────────────────────

type LoginRequest struct {
	Password string `json:"password"`
}

type UpdateCategoryRequest struct {
	Category string `json:"category"`
}

type CreateIncomeRequest struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Source string  `json:"source"`
	Notes  string  `json:"notes,omitempty"`
}

type CreateDebtRequest struct {
	Name        string  `json:"name"`
	TotalAmount float64 `json:"total_amount"`
	Remaining   float64 `json:"remaining"`
	Notes       string  `json:"notes,omitempty"`
}

type PayDebtRequest struct {
	DebtID int     `json:"debt_id"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
	Notes  string  `json:"notes,omitempty"`
}

type CreateWishlistRequest struct {
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	Priority     int     `json:"priority,omitempty"`
	Notes        string  `json:"notes,omitempty"`
}

type UpdateWishlistRequest struct {
	TargetAmount *float64 `json:"target_amount,omitempty"`
	SavedAmount  *float64 `json:"saved_amount,omitempty"`
	Priority     *int     `json:"priority,omitempty"`
	Achieved     *bool    `json:"achieved,omitempty"`
	Notes        *string  `json:"notes,omitempty"`
}

type SaveWishlistRequest struct {
	WishID int     `json:"wish_id"`
	Amount float64 `json:"amount"`
}

type CreateTransactionRequest struct {
	Date        string  `json:"date"`
	Time        string  `json:"time,omitempty"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency,omitempty"`
	Merchant    string  `json:"merchant"`
	Account     string  `json:"account,omitempty"`
	Method      string  `json:"method,omitempty"`
	Description string  `json:"description,omitempty"`
}

// ─── Response Payloads ──────────────────────────────────────

type APIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type SummaryResponse struct {
	IncomeThisMonth  float64            `json:"income_this_month"`
	ExpensesThisMonth float64           `json:"expenses_this_month"`
	Balance          float64            `json:"balance"`
	Categories       []CategorySummary  `json:"categories"`
	Debts            []DebtSummary      `json:"debts"`
	Wishlist         []WishlistSummary  `json:"wishlist"`
}

type CategorySummary struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

type DebtSummary struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	TotalAmount float64 `json:"total_amount"`
	Remaining   float64 `json:"remaining"`
	PaidOff     bool    `json:"paid_off"`
	Progress    float64 `json:"progress"`
}

type WishlistSummary struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	SavedAmount  float64 `json:"saved_amount"`
	Achieved     bool    `json:"achieved"`
	Progress     float64 `json:"progress"`
}
