package users

import (
	"Connect-4/internals/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// You should pass your *sql.DB connection to these handlers in real usage

// SignupHandler handles user registration
func SignupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		if req.Username == "" || req.Password == "" {
			http.Error(w, "Username and password required", http.StatusBadRequest)
			return
		}
		// Check if user exists
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&exists)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if exists > 0 {
			http.Error(w, "Username already taken", http.StatusConflict)
			return
		}
		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		// Insert user
		_, err = db.Exec("INSERT INTO users (username, password, email) VALUES (?, ?, ?)", req.Username, string(hash), req.Email)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		// inserting into rankings table
		_, err = db.Exec(`INSERT OR IGNORE INTO rankings (username, score) VALUES (?, 0)`, req.Username)
		if err != nil {
			log.Printf("Failed to insert into rankings: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Signup successful"))
	}
}

// LoginHandler handles user login
func LoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		var user models.User
		err := db.QueryRow("SELECT id, username, password, email FROM users WHERE username = ?", req.Username).Scan(&user.Id, &user.Username, &user.Password, &user.Email)
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		w.Write([]byte("Login successful"))
	}
}
