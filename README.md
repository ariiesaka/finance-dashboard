# Finance Dashboard

Personal finance dashboard built with Go, SQLite, and vanilla HTML/JS.

Track expenses, manage debts, save toward wishlists — all from one dashboard hosted on your own server.

## Tech Stack

- **Backend:** Go (stdlib)
- **Database:** SQLite
- **Frontend:** Vanilla HTML + JS + Chart.js
- **Auth:** bcrypt + session cookies

## Quick Start

```bash
# Set your password (first time only)
./dashboard setup

# Start server
./dashboard serve --port 8080

# Open in browser
# http://localhost:8080/login.html
```

## Project Structure

```
├── main.go          — CLI entry point (setup | serve)
├── auth.go          — password hashing, sessions, auth middleware
├── db.go            — SQLite schema + queries
├── handlers.go      — HTTP API endpoints
├── models.go        — data structures
├── static/
│   ├── login.html
│   ├── dashboard.html
│   └── admin.html
└── data/
    └── finances.db  — SQLite database (created at runtime)
```
