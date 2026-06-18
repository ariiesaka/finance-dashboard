package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./dashboard <command>")
		fmt.Println("Commands:")
		fmt.Println("  setup              Interactive CLI to set up the database and password")
		fmt.Println("  serve              Start the HTTP server")
		fmt.Println("  serve --port N     Start on a specific port (default: 8080)")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "setup":
		cmdSetup()
	case "serve":
		cmdServe()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available: setup, serve")
		os.Exit(1)
	}
}

func cmdSetup() {
	if dbExists() {
		// Prompt to overwrite
		fmt.Print("Database already exists. Overwrite? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Setup cancelled.")
			return
		}
		// Remove old DB
		os.Remove("./data/finances.db")
	}

	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer closeDB()

	fmt.Print("Enter password for finance dashboard: ")
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	password = strings.TrimSpace(password)

	if len(password) < 4 {
		log.Fatalf("Password must be at least 4 characters")
	}

	hash, err := hashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	if err := storePasswordHash(hash); err != nil {
		log.Fatalf("Failed to store password: %v", err)
	}

	fmt.Println("Setup complete! Database created and password stored.")
	fmt.Println("You can now run: ./dashboard serve")
}

func cmdServe() {
	// Parse port from args (it's after "serve")
	port := 8080
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	serveCmd.IntVar(&port, "port", 8080, "HTTP server port")
	serveCmd.Parse(os.Args[2:])

	if !dbExists() {
		log.Fatal("No database found. Run './dashboard setup' first.")
	}

	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer closeDB()

	// Clean up expired sessions on startup
	cleanupExpiredSessions()

	mux := http.NewServeMux()
	registerRoutes(mux)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Finance Dashboard starting on http://localhost%s", addr)
	log.Printf("Open http://localhost%s/login.html to get started", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
