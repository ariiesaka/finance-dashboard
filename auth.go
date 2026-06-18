package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenLength = 64
	sessionExpiryDays  = 7
	cookieName         = "session_token"
)

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

func checkPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, sessionTokenLength/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func createSession() (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(sessionExpiryDays * 24 * time.Hour)
	if err := storeSession(token, expiresAt.Format("2006-01-02 15:04:05")); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}

	return token, nil
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionExpiryDays * 24 * 3600,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// AuthMiddleware checks for a valid session cookie.
// For API routes (prefix /api/), returns 401 JSON.
// For static file routes, redirects to /login.html.
func AuthMiddleware(next http.HandlerFunc, isAPI bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow login endpoint without auth
		if r.URL.Path == "/api/login" {
			next(w, r)
			return
		}

		cookie, err := r.Cookie(cookieName)
		if err != nil {
			if isAPI {
				writeJSON(w, http.StatusUnauthorized, map[string]bool{"ok": false})
			} else {
				http.Redirect(w, r, "/login.html", http.StatusFound)
			}
			return
		}

		session, err := getSession(cookie.Value)
		if err != nil {
			clearSessionCookie(w)
			if isAPI {
				writeJSON(w, http.StatusUnauthorized, map[string]bool{"ok": false})
			} else {
				http.Redirect(w, r, "/login.html", http.StatusFound)
			}
			return
		}

		if time.Now().After(session.ExpiresAt) {
			deleteSession(cookie.Value)
			clearSessionCookie(w)
			if isAPI {
				writeJSON(w, http.StatusUnauthorized, map[string]bool{"ok": false})
			} else {
				http.Redirect(w, r, "/login.html", http.StatusFound)
			}
			return
		}

		next(w, r)
	}
}

// StaticFileAuthMiddleware wraps a file server handler to check auth for HTML files
func StaticFileAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow login.html without auth
		if r.URL.Path == "/login.html" || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}

		// Only check auth for .html files
		if strings.HasSuffix(r.URL.Path, ".html") || r.URL.Path == "/dashboard" || r.URL.Path == "/admin" {
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}

			session, err := getSession(cookie.Value)
			if err != nil || time.Now().After(session.ExpiresAt) {
				if err == nil {
					deleteSession(cookie.Value)
				}
				clearSessionCookie(w)
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}

			// Redirect /dashboard and /admin to their .html pages
			if r.URL.Path == "/dashboard" {
				http.Redirect(w, r, "/dashboard.html", http.StatusFound)
				return
			}
			if r.URL.Path == "/admin" {
				http.Redirect(w, r, "/admin.html", http.StatusFound)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]bool{"ok": false})
		return
	}

	hash, err := getPasswordHash()
	if err != nil {
		log.Printf("Login error: %v", err)
		writeJSON(w, http.StatusUnauthorized, map[string]bool{"ok": false})
		return
	}

	if !checkPassword(hash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]bool{"ok": false})
		return
	}

	token, err := createSession()
	if err != nil {
		log.Printf("Session creation error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]bool{"ok": false})
		return
	}

	setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cookie, err := r.Cookie(cookieName)
	if err == nil {
		deleteSession(cookie.Value)
	}

	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleCheckAuth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
