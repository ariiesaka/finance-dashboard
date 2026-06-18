package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// ─── Debt CRUD ──────────────────────────────────────────────

func (h *authHandler) listDebts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debts, err := listDebts(h.db)
	if err != nil {
		log.Printf("list debts: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to fetch"})
		return
	}

	// Enrich each debt with total paid + progress
	type enriched struct {
		Debt      Debt    `json:"debt"`
		TotalPaid float64 `json:"total_paid"`
		Progress  int     `json:"progress"`
	}
	results := make([]enriched, 0, len(debts))
	for _, d := range debts {
		paid, err := totalPaidForDebt(h.db, d.ID)
		if err != nil {
			log.Printf("total paid for debt %d: %v", d.ID, err)
			paid = 0
		}
		progress := 0
		if d.TotalAmount > 0 {
			progress = int((paid / d.TotalAmount) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		results = append(results, enriched{Debt: d, TotalPaid: paid, Progress: progress})
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *authHandler) createDebt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.Name == "" || req.TotalAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "name and total_amount required"})
		return
	}

	priority := req.Priority
	if priority != "high" && priority != "medium" && priority != "low" {
		priority = "medium"
	}

	d := Debt{
		Name:         req.Name,
		Description:  req.Description,
		TotalAmount:  req.TotalAmount,
		Remaining:    req.TotalAmount,
		InterestRate: req.InterestRate,
		Priority:     priority,
		Notes:        req.Notes,
	}

	id, err := createDebt(h.db, d)
	if err != nil {
		log.Printf("create debt: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to save"})
		return
	}

	d.ID = int(id)
	d.Remaining = req.TotalAmount
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":   true,
		"id":   id,
		"debt": d,
	})
}

func (h *authHandler) deleteDebt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid id"})
		return
	}

	if err := deleteDebt(h.db, id); err != nil {
		log.Printf("delete debt %d: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to delete"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{OK: true})
}

// ─── Debt Payments ──────────────────────────────────────────

func (h *authHandler) createPayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.DebtID <= 0 || req.Amount <= 0 || req.Date == "" {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "debt_id, amount, and date required"})
		return
	}

	p := DebtPayment{
		DebtID: req.DebtID,
		Amount: req.Amount,
		Date:   req.Date,
		Notes:  req.Notes,
	}

	id, err := createDebtPayment(h.db, p)
	if err != nil {
		log.Printf("create payment: %v", err)
		code := http.StatusInternalServerError
		msg := "failed to save"
		if err.Error() == "payment exceeds remaining balance" {
			code = http.StatusBadRequest
			msg = "payment exceeds remaining balance"
		}
		writeJSON(w, code, APIResponse{OK: false, Error: msg})
		return
	}

	p.ID = int(id)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":      true,
		"id":      id,
		"payment": p,
	})
}

func (h *authHandler) listPayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debtIDStr := r.URL.Query().Get("debt_id")
	debtID, err := strconv.Atoi(debtIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid debt_id"})
		return
	}

	payments, err := listDebtPayments(h.db, debtID)
	if err != nil {
		log.Printf("list payments: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to fetch"})
		return
	}

	writeJSON(w, http.StatusOK, payments)
}
