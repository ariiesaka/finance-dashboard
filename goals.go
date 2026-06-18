package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// ─── Goal CRUD ──────────────────────────────────────────────

func (h *authHandler) listGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	goals, err := listGoals(h.db)
	if err != nil {
		log.Printf("list goals: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to fetch"})
		return
	}

	// Enrich with progress percentage
	type enriched struct {
		Goal     Goal `json:"goal"`
		Progress int  `json:"progress"`
	}
	results := make([]enriched, 0, len(goals))
	for _, g := range goals {
		progress := 0
		if g.TargetAmount > 0 {
			progress = int((g.CurrentAmount / g.TargetAmount) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		results = append(results, enriched{Goal: g, Progress: progress})
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *authHandler) createGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if req.Name == "" || req.TargetAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "name and target_amount required"})
		return
	}

	priority := req.Priority
	if priority != "high" && priority != "medium" && priority != "low" {
		priority = "medium"
	}

	g := Goal{
		Name:          req.Name,
		TargetAmount:  req.TargetAmount,
		CurrentAmount: 0,
		Priority:      priority,
		Notes:         req.Notes,
	}

	id, err := createGoal(h.db, g)
	if err != nil {
		log.Printf("create goal: %v", err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to save"})
		return
	}

	g.ID = int(id)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok":   true,
		"id":   id,
		"goal": g,
	})
}

func (h *authHandler) updateGoal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid id"})
		return
	}

	var req UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, APIResponse{OK: false, Error: "invalid request"})
		return
	}

	if err := updateGoalProgress(h.db, id, req.CurrentAmount); err != nil {
		log.Printf("update goal %d: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to update"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{OK: true})
}

func (h *authHandler) deleteGoal(w http.ResponseWriter, r *http.Request) {
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

	if err := deleteGoal(h.db, id); err != nil {
		log.Printf("delete goal %d: %v", id, err)
		writeJSON(w, http.StatusInternalServerError, APIResponse{OK: false, Error: "failed to delete"})
		return
	}

	writeJSON(w, http.StatusOK, APIResponse{OK: true})
}
