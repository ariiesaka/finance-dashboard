package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const dbDir = "data"
const dbFile = "finances.db"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: dashboard <setup|serve> [--port PORT] [--secure]")
		os.Exit(1)
	}

	absDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatalf("failed to get directory: %v", err)
	}

	dbPath := filepath.Join(absDir, dbDir, dbFile)
	if err := os.MkdirAll(filepath.Join(absDir, dbDir), 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	db, err := initDB(dbPath)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}
	defer db.Close()

	switch os.Args[1] {
	case "setup":
		cmdSetup(db)
	case "serve":
		port := "8080"
		secure := false
		for i, arg := range os.Args {
			if arg == "--port" && i+1 < len(os.Args) {
				port = os.Args[i+1]
			}
			if arg == "--secure" {
				secure = true
			}
		}
		cmdServe(db, port, secure)
	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		fmt.Println("Usage: dashboard <setup|serve> [--port PORT] [--secure]")
		os.Exit(1)
	}
}

func cmdSetup(db *sql.DB) {
	hasUser, err := hasUser(db)
	if err != nil {
		log.Fatalf("check user: %v", err)
	}
	if hasUser {
		fmt.Print("Password already set. Overwrite? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		reply, _ := reader.ReadString('\n')
		reply = strings.TrimSpace(strings.ToLower(reply))
		if reply != "y" && reply != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	fmt.Print("Enter password: ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if len(password) < 4 {
		log.Fatal("password must be at least 4 characters")
	}

	hash, err := hashPassword(password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	if hasUser {
		_, err = db.Exec("UPDATE users SET password_hash = ? WHERE id = 1", hash)
	} else {
		err = storePassword(db, hash)
	}
	if err != nil {
		log.Fatalf("store password: %v", err)
	}
	fmt.Println("Password set successfully.")
}

func cmdServe(db *sql.DB, port string, secure bool) {
	hasUser, err := hasUser(db)
	if err != nil {
		log.Fatalf("check user: %v", err)
	}
	if !hasUser {
		log.Fatal("no password set. Run 'dashboard setup' first.")
	}

	cleanExpiredSessions(db)

	handler := &authHandler{db: db, secureCookie: secure}

	mux := http.NewServeMux()

	mux.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/login.html")
	})
	mux.HandleFunc("/api/login", handler.login)
	// Protected routes
	mux.Handle("/api/logout", authMiddleware(db, handler.logout))
	mux.Handle("/api/check-auth", authMiddleware(db, handler.checkAuth))

	// Expense routes
	mux.Handle("/api/expenses", authMiddleware(db, handler.createExpense))
	mux.Handle("/api/expenses/list", authMiddleware(db, handler.listExpenses))
	mux.Handle("/api/expenses/delete", authMiddleware(db, handler.deleteExpense))

	// Debt routes
	mux.Handle("/api/debts", authMiddleware(db, handler.createDebt))
	mux.Handle("/api/debts/list", authMiddleware(db, handler.listDebts))
	mux.Handle("/api/debts/delete", authMiddleware(db, handler.deleteDebt))
	mux.Handle("/api/debts/payments", authMiddleware(db, handler.createPayment))
	mux.Handle("/api/debts/payments/list", authMiddleware(db, handler.listPayments))

	// Goal routes
	mux.Handle("/api/goals", authMiddleware(db, handler.createGoal))
	mux.Handle("/api/goals/list", authMiddleware(db, handler.listGoals))
	mux.Handle("/api/goals/update", authMiddleware(db, handler.updateGoal))
	mux.Handle("/api/goals/delete", authMiddleware(db, handler.deleteGoal))

	mux.Handle("/dashboard.html", authMiddleware(db, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/dashboard.html")
	}))

	protocol := "http"
	if secure {
		protocol = "https"
	}
	fmt.Printf("Server starting on :%s\n", port)
	fmt.Printf("Open %s://localhost:%s/login.html\n", protocol, port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
