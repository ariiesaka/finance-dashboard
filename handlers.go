package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type authHandler struct {
	db          *sql.DB
	secureCookie bool
}

func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.Password == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "password required"})
		return
	}

	hash, err := getPasswordHash(h.db)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "wrong password"})
		return
	}

	if !checkPassword(hash, req.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "wrong password"})
		return
	}

	sessionID, err := createSession(h.db)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "internal error"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{OK: true})
}

func (h *authHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		deleteSession(h.db, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{OK: true})
}

func (h *authHandler) checkAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{OK: true})
}

// ─── Expense Handlers ────────────────────────────────────────

func (h *authHandler) createExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.Merchant == "" || req.Amount <= 0 || req.Date == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "merchant, amount, and date required"})
		return
	}

	txn := Transaction{
		Date:     req.Date,
		Category: req.Category,
		Amount:   req.Amount,
		Merchant: req.Merchant,
		Account:  req.Account,
		Method:   req.Method,
		Notes:    req.Notes,
	}

	id, err := createExpense(h.db, txn)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to save"})
		return
	}

	txn.ID = int(id)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":    true,
		"id":    id,
		"transaction": txn,
	})
}
func (h *authHandler) listExpenses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	month := r.URL.Query().Get("month")

	var txns []Transaction
	var cats []CategoryBreakdown
	var total float64

	if startDate != "" && endDate != "" {
		var err error
		txns, err = listExpensesForRange(h.db, startDate, endDate)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		cats, err = categoryBreakdownForRange(h.db, startDate, endDate)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		total, err = totalExpensesForRange(h.db, startDate, endDate)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
	} else if month != "" {
		var err error
		txns, err = listExpensesForMonth(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		cats, err = expenseCategoryBreakdown(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		total, err = totalExpensesForMonth(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
	} else {
		// Default: current month
		month = time.Now().UTC().Format("2006-01")
		var err error
		txns, err = listExpensesForMonth(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		cats, err = expenseCategoryBreakdown(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
		total, err = totalExpensesForMonth(h.db, month)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to list"})
			return
		}
	}

	writeJSON(w, http.StatusOK, MonthSummary{
		TotalExpenses: total,
		Categories:    cats,
		Transactions:  txns,
	})
}

func (h *authHandler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "id required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid id"})
		return
	}

	_, err = h.db.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to delete"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{OK: true})
}

func (h *authHandler) updateExpense(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.ID <= 0 || req.Merchant == "" || req.Amount <= 0 || req.Date == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "id, merchant, amount, and date required"})
		return
	}

	txn := Transaction{
		ID:       req.ID,
		Date:     req.Date,
		Category: req.Category,
		Amount:   req.Amount,
		Merchant: req.Merchant,
		Account:  req.Account,
		Method:   req.Method,
		Notes:    req.Notes,
	}

	if err := updateExpense(h.db, txn); err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":    true,
		"transaction": txn,
	})
}

func (h *authHandler) expenseDateRange(w http.ResponseWriter, r *http.Request) {
	minDate, maxDate, err := getExpenseDateRange(h.db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to get date range"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"min_date": minDate,
		"max_date": maxDate,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
