package main

import (
	"Connect-4/internals/config"
	"Connect-4/internals/handlers/matchmaking"
	"Connect-4/internals/handlers/users"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors" // Import the cors package
)

func main() {
	fmt.Println("Starting main...")
	cfg := config.MustLoad()
	fmt.Println("Config loaded")

	db, err := sql.Open("sqlite3", cfg.Database.SQLitePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("Database connected")

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);`

	createRankingsTableSQL := `
	CREATE TABLE IF NOT EXISTS rankings (
		username TEXT PRIMARY KEY,
		score INTEGER NOT NULL DEFAULT 0
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Users table created or already exists")

	_, err = db.Exec(createRankingsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create rankings table: %v", err)
	}
	fmt.Println("Rankings table created or already exists")

	_, err = db.Exec(`
    INSERT INTO rankings (username, score)
    SELECT username, 0
    FROM users
    WHERE username NOT IN (SELECT username FROM rankings)
`)
	if err != nil {
		log.Fatalf("Failed to backfill rankings for existing users: %v", err)
	}
	fmt.Println("Rankings table backfilled for existing users")
	// Initialize matchmaking with DB
	matchmaking.InitRankingDB(db)

	createGamesTableSQL := `
	CREATE TABLE IF NOT EXISTS games (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		player1 TEXT NOT NULL,
		player2 TEXT NOT NULL,
		winner TEXT,
		moves TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createGamesTableSQL)
	if err != nil {
		log.Fatalf("Failed to create games table: %v", err)
	}
	fmt.Println("Games table created or already exists")

	router := http.NewServeMux()
	router.HandleFunc("/api/signup", users.SignupHandler(db))     // api for signup
	router.HandleFunc("/api/login", users.LoginHandler(db))       // api for login
	router.HandleFunc("/ws/game", matchmaking.HandleGame)         // WebSocket endpoint for games
	router.HandleFunc("/api/rankings", matchmaking.HandleRanking) // api for rankings

	fmt.Println("Router setup complete")

	// Create a CORS handler
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},                                                // Allow all origins. For production, specify your frontend's domain.
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "PATCH", "PUT", "DELETE"}, // Allow all methods
		AllowedHeaders: []string{"Content-Type"},                                     // Allow only the Content-Type header
	})

	// Wrap the router with the CORS handler
	handler := c.Handler(router)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: handler, // Use the wrapped handler here
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("Server Started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-done
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to gracefully shutdown server: %v", err)
	}
	fmt.Println("Server stopped")
}
