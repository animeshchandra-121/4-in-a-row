package matchmaking

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db        *sql.DB
	rankMutex sync.Mutex
)

// InitRankingDB initializes the database
func InitRankingDB(database *sql.DB) {
	db = database
}
func SaveGame(player1, player2, winner string, moves []string) {
	rankMutex.Lock()
	defer rankMutex.Unlock()

	movesStr := strings.Join(moves, ",") // store moves as comma-separated string

	_, err := db.Exec(`
		INSERT INTO games (player1, player2, winner, moves)
		VALUES (?, ?, ?, ?)
	`, player1, player2, winner, movesStr)

	if err != nil {
		log.Printf("Error saving game: %v", err)
	}
}

// AddWin increases a player's score in DB
func AddWin(username string) {
	rankMutex.Lock()
	defer rankMutex.Unlock()

	_, err := db.Exec(`
		INSERT INTO rankings (username, score)
		VALUES (?, 1)
		ON CONFLICT(username) DO UPDATE SET score = score + 1
	`, username)

	if err != nil {
		log.Printf("Error updating score for %s: %v", username, err)
	}
}

// GetRanking returns a sorted list of players by score
func GetRanking() []struct {
	Username string
	Score    int
} {
	rankMutex.Lock()
	defer rankMutex.Unlock()

	rows, err := db.Query(`SELECT username, score FROM rankings`)
	if err != nil {
		log.Printf("Error fetching rankings: %v", err)
		return nil
	}
	defer rows.Close()

	var ranking []struct {
		Username string
		Score    int
	}
	for rows.Next() {
		var user string
		var score int
		if err := rows.Scan(&user, &score); err != nil {
			log.Println("Error scanning row:", err)
			continue
		}
		ranking = append(ranking, struct {
			Username string
			Score    int
		}{user, score})
	}

	// Sort by score desc, then username
	sort.Slice(ranking, func(i, j int) bool {
		if ranking[i].Score == ranking[j].Score {
			return ranking[i].Username < ranking[j].Username
		}
		return ranking[i].Score > ranking[j].Score
	})

	return ranking
}
func HandleRanking(w http.ResponseWriter, r *http.Request) {
	ranking := GetRanking()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ranking)
}
