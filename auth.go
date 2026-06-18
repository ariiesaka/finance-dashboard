package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "session_token"
	sessionDuration   = 7 * 24 * time.Hour
	bcryptCost        = 12
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func createSession(db *sql.DB) (string, error) {
	id, err := generateSessionID()
	if err != nil {
		return "", err
	}
	_, err = db.Exec(
		"INSERT INTO sessions (id, expires_at) VALUES (?, ?)",
		id, time.Now().UTC().Add(sessionDuration),
	)
	return id, err
}

func deleteSession(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

func cleanExpiredSessions(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC())
	return err
}

func validateSession(db *sql.DB, token string) bool {
	var expiresAt time.Time
	err := db.QueryRow(
		"SELECT expires_at FROM sessions WHERE id = ?", token,
	).Scan(&expiresAt)
	if err != nil {
		return false
	}
	if time.Now().UTC().After(expiresAt) {
		deleteSession(db, token)
		return false
	}
	return true
}

func authMiddleware(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !validateSession(db, cookie.Value) {
			// API requests get JSON 401
			if r.Header.Get("Accept") == "application/json" ||
				r.URL.Path == "/api/check-auth" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(APIResponse{OK: false, Error: "unauthorized"})
				return
			}
			// Page requests get redirect
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		next(w, r)
	}
}
