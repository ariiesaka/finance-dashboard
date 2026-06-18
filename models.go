package main

type User struct {
	ID           int    `json:"id"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

type Session struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
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
	ID       int     `json:"id"`
	Date     string  `json:"date"`
	Time     string  `json:"time,omitempty"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Merchant string  `json:"merchant"`
	Account  string  `json:"account,omitempty"`
	Method   string  `json:"method,omitempty"`
	Notes    string  `json:"notes,omitempty"`
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

// ─── Debt Models ────────────────────────────────────────────

type Debt struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description,omitempty"`
	TotalAmount     float64 `json:"total_amount"`
	Remaining       float64 `json:"remaining"`
	InterestRate    float64 `json:"interest_rate,omitempty"`
	Priority        string  `json:"priority"` // high, medium, low
	Notes           string  `json:"notes,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type CreateDebtRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	TotalAmount  float64 `json:"total_amount"`
	InterestRate float64 `json:"interest_rate,omitempty"`
	Priority     string  `json:"priority"`
	Notes        string  `json:"notes,omitempty"`
}

type DebtPayment struct {
	ID      int     `json:"id"`
	DebtID  int     `json:"debt_id"`
	Amount  float64 `json:"amount"`
	Date    string  `json:"date"`
	Notes   string  `json:"notes,omitempty"`
}

type CreatePaymentRequest struct {
	DebtID int     `json:"debt_id"`
	Amount float64 `json:"amount"`
	Date   string  `json:"date"`
	Notes  string  `json:"notes,omitempty"`
}

type DebtDetail struct {
	Debt     Debt          `json:"debt"`
	TotalPaid float64      `json:"total_paid"`
	Progress  int          `json:"progress"` // percentage 0-100
}

// ─── Goal Models ────────────────────────────────────────────

type Goal struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	TargetAmount  float64 `json:"target_amount"`
	CurrentAmount float64 `json:"current_amount"`
	Priority      string  `json:"priority"` // high, medium, low
	Notes         string  `json:"notes,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type CreateGoalRequest struct {
	Name          string  `json:"name"`
	TargetAmount  float64 `json:"target_amount"`
	Priority      string  `json:"priority"`
	Notes         string  `json:"notes,omitempty"`
}

type UpdateGoalRequest struct {
	CurrentAmount float64 `json:"current_amount"`
}
