package main

import "time"

type Transaction struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Time        string  `json:"time,omitempty"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Merchant    string  `json:"merchant,omitempty"`
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
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	TotalAmount float64 `json:"total_amount"`
	Remaining   float64 `json:"remaining"`
	PaidOff     int     `json:"paid_off"`
	Notes       string  `json:"notes,omitempty"`
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
	Achieved     int     `json:"achieved"`
	Notes        string  `json:"notes,omitempty"`
}

type SummaryResponse struct {
	IncomeThisMonth  float64            `json:"income_this_month"`
	ExpensesThisMonth float64           `json:"expenses_this_month"`
	Balance          float64            `json:"balance"`
	Categories       []CategoryBreakdown `json:"categories"`
	Debts            []DebtSummary      `json:"debts"`
	Wishlist         []WishlistItem     `json:"wishlist"`
}

type CategoryBreakdown struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}

type DebtSummary struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	TotalAmount float64 `json:"total_amount"`
	Remaining   float64 `json:"remaining"`
	Percentage  float64 `json:"percentage"`
}

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Ok bool `json:"ok"`
}

type AuthCheckResponse struct {
	Ok bool `json:"ok"`
}

type CategoryUpdate struct {
	Category string `json:"category"`
}

type IncomeRequest struct {
	Amount float64 `json:"amount"`
	Source string  `json:"source"`
	Date   string  `json:"date"`
}

type DebtRequest struct {
	Name        string  `json:"name"`
	TotalAmount float64 `json:"total_amount"`
	Remaining   float64 `json:"remaining"`
}

type DebtPayRequest struct {
	DebtID int     `json:"debt_id"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
	Notes  string  `json:"notes,omitempty"`
}

type WishlistRequest struct {
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	Priority     int     `json:"priority,omitempty"`
}

type WishlistUpdate struct {
	TargetAmount *float64 `json:"target_amount,omitempty"`
	SavedAmount  *float64 `json:"saved_amount,omitempty"`
	Priority     *int     `json:"priority,omitempty"`
	Achieved     *int     `json:"achieved,omitempty"`
}

type WishlistSaveRequest struct {
	WishID int     `json:"wish_id"`
	Amount float64 `json:"amount"`
}

type TransactionRequest struct {
	Date        string  `json:"date"`
	Time        string  `json:"time,omitempty"`
	Type        string  `json:"type"`
	Category    string  `json:"category"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency,omitempty"`
	Merchant    string  `json:"merchant,omitempty"`
	Account     string  `json:"account,omitempty"`
	Method      string  `json:"method,omitempty"`
	Description string  `json:"description,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
