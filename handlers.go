package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func handleSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	if month == "" {
		month = fmt.Sprintf("%02d", now.Month())
	}
	if year == "" {
		year = fmt.Sprintf("%d", now.Year())
	}

	summary, err := getSummary(month, year)
	if err != nil {
		log.Printf("Summary error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	// Check for path-based params like /api/transactions/:id/category
	path := strings.TrimPrefix(r.URL.Path, "/api/transactions")
	path = strings.TrimPrefix(path, "/")

	if path != "" && path != "transactions" {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 && parts[1] == "category" {
			handleUpdateTransactionCategory(w, r, parts[0])
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	if r.Method == http.MethodPost {
		handleCreateTransaction(w, r)
		return
	}

	month := r.URL.Query().Get("month")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	transactions, err := getTransactions(month, limit)
	if err != nil {
		log.Printf("Transactions error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if transactions == nil {
		transactions = []Transaction{}
	}

	writeJSON(w, http.StatusOK, transactions)
}

func handleUpdateTransactionCategory(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req CategoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec("UPDATE transactions SET category = ? WHERE id = ?", req.Category, id)
	if err != nil {
		log.Printf("Update category error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if req.Type == "" {
		req.Type = "expense"
	}
	if req.Category == "" {
		req.Category = "Other"
	}
	if req.Currency == "" {
		req.Currency = "IDR"
	}

	// Generate UUID-like ID
	id := generateID()

	_, err := db.Exec(
		"INSERT INTO transactions (id, date, time, type, category, amount, currency, merchant, account, method, description) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.Date, req.Time, req.Type, req.Category, req.Amount, req.Currency, req.Merchant, req.Account, req.Method, req.Description,
	)
	if err != nil {
		log.Printf("Create transaction error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"ok": true, "id": id})
}

func handleIncome(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleCreateIncome(w, r)
		return
	}

	// GET /api/income
	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	if year == "" {
		year = fmt.Sprintf("%d", time.Now().Year())
	}
	if month == "" {
		month = fmt.Sprintf("%02d", time.Now().Month())
	}

	incomes, err := getIncome(year, month)
	if err != nil {
		log.Printf("Income error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if incomes == nil {
		incomes = []Income{}
	}

	writeJSON(w, http.StatusOK, incomes)
}

func handleCreateIncome(w http.ResponseWriter, r *http.Request) {
	// Check for bulk
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read error"})
		return
	}

	// Try as array first (bulk)
	var bulk []IncomeRequest
	if err := json.Unmarshal(body, &bulk); err == nil {
		for _, req := range bulk {
			_, err := db.Exec("INSERT INTO income (date, amount, source) VALUES (?, ?, ?)",
				req.Date, req.Amount, req.Source)
			if err != nil {
				log.Printf("Bulk income error: %v", err)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusCreated, map[string]bool{"ok": true, "bulk": true})
		return
	}

	// Single
	var req IncomeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	_, err = db.Exec("INSERT INTO income (date, amount, source) VALUES (?, ?, ?)",
		req.Date, req.Amount, req.Source)
	if err != nil {
		log.Printf("Create income error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

func handleDebts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		handleCreateDebt(w, r)
		return
	}

	debts, err := getDebts()
	if err != nil {
		log.Printf("Debts error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if debts == nil {
		debts = []Debt{}
	}

	writeJSON(w, http.StatusOK, debts)
}

func handleCreateDebt(w http.ResponseWriter, r *http.Request) {
	var req DebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec("INSERT INTO debts (name, total_amount, remaining) VALUES (?, ?, ?)",
		req.Name, req.TotalAmount, req.Remaining)
	if err != nil {
		log.Printf("Create debt error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

func handleDebtPay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req DebtPayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Add payment record
	_, err := db.Exec("INSERT INTO debt_payments (debt_id, amount, date, notes) VALUES (?, ?, ?, ?)",
		req.DebtID, req.Amount, req.Date, req.Notes)
	if err != nil {
		log.Printf("Debt payment error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update remaining
	_, err = db.Exec("UPDATE debts SET remaining = remaining - ? WHERE id = ?", req.Amount, req.DebtID)
	if err != nil {
		log.Printf("Update debt remaining error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Check if paid off
	var remaining float64
	db.QueryRow("SELECT remaining FROM debts WHERE id = ?", req.DebtID).Scan(&remaining)
	if remaining <= 0 {
		db.Exec("UPDATE debts SET paid_off = 1, remaining = 0 WHERE id = ?", req.DebtID)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleWishlist(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/wishlist")
	path = strings.TrimPrefix(path, "/")

	if path != "" && path != "wishlist" {
		parts := strings.Split(path, "/")
		if len(parts) >= 1 && parts[0] == "save" {
			handleWishlistSave(w, r)
			return
		}
		if len(parts) >= 1 {
			handleUpdateWishlist(w, r, parts[0])
			return
		}
	}

	if r.Method == http.MethodPost {
		handleCreateWishlist(w, r)
		return
	}

	items, err := getWishlist()
	if err != nil {
		log.Printf("Wishlist error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if items == nil {
		items = []WishlistItem{}
	}

	writeJSON(w, http.StatusOK, items)
}

func handleCreateWishlist(w http.ResponseWriter, r *http.Request) {
	var req WishlistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec("INSERT INTO wishlist (name, target_amount, priority) VALUES (?, ?, ?)",
		req.Name, req.TargetAmount, req.Priority)
	if err != nil {
		log.Printf("Create wishlist error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

func handleUpdateWishlist(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req WishlistUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Build dynamic update
	setClauses := []string{}
	args := []interface{}{}

	if req.TargetAmount != nil {
		setClauses = append(setClauses, "target_amount = ?")
		args = append(args, *req.TargetAmount)
	}
	if req.SavedAmount != nil {
		setClauses = append(setClauses, "saved_amount = ?")
		args = append(args, *req.SavedAmount)
	}
	if req.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *req.Priority)
	}
	if req.Achieved != nil {
		setClauses = append(setClauses, "achieved = ?")
		args = append(args, *req.Achieved)
	}

	if len(setClauses) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		return
	}

	query := fmt.Sprintf("UPDATE wishlist SET %s WHERE id = ?", strings.Join(setClauses, ", "))
	args = append(args, id)

	_, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Update wishlist error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleWishlistSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req WishlistSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	_, err := db.Exec("UPDATE wishlist SET saved_amount = saved_amount + ? WHERE id = ?", req.Amount, req.WishID)
	if err != nil {
		log.Printf("Wishlist save error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Check if achieved
	var saved, target float64
	db.QueryRow("SELECT saved_amount, target_amount FROM wishlist WHERE id = ?", req.WishID).Scan(&saved, &target)
	if saved >= target {
		db.Exec("UPDATE wishlist SET achieved = 1 WHERE id = ?", req.WishID)
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func registerRoutes(mux *http.ServeMux) {
	// Auth routes
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/logout", AuthMiddleware(handleLogout, true))
	mux.HandleFunc("/api/check-auth", AuthMiddleware(handleCheckAuth, true))

	// Protected API routes
	mux.HandleFunc("/api/summary", AuthMiddleware(handleSummary, true))
	mux.HandleFunc("/api/transactions", AuthMiddleware(handleTransactions, true))
	mux.HandleFunc("/api/transactions/", AuthMiddleware(handleTransactions, true))
	mux.HandleFunc("/api/income", AuthMiddleware(handleIncome, true))
	mux.HandleFunc("/api/income/", AuthMiddleware(handleIncome, true))
	mux.HandleFunc("/api/debts", AuthMiddleware(handleDebts, true))
	mux.HandleFunc("/api/debts/", AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/debts/pay") {
			handleDebtPay(w, r)
			return
		}
		handleDebts(w, r)
	}, true))
	mux.HandleFunc("/api/wishlist", AuthMiddleware(handleWishlist, true))
	mux.HandleFunc("/api/wishlist/", AuthMiddleware(handleWishlist, true))

	// Static file serving with auth middleware
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", StaticFileAuthMiddleware(fs))
}
