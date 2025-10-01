package matchmaking

import (
	"Connect-4/internals/handlers/game"
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	lru "github.com/hashicorp/golang-lru"
)

type Player struct {
	// UserID   string // A unique identifier for the user (e.g., token, unique username)
	Username string
	Conn     *websocket.Conn
	ID       int // 1 or 2
}

// You'll also need a struct to store in the cache
type CachedGame struct {
	Game        *game.Game
	Player1     *Player
	Player2     *Player
	Timestamp   time.Time
	CancelTimer context.CancelFunc // To track when the game was cached
}

// Move represents the message structure for a player's move
type Move struct {
	Type   string `json:"type"`
	Col    int    `json:"col"`
	Player int    `json:"player"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	// Make queue a channel to simplify waiting logic
	playerQueue            = make(chan *Player, 1)
	games                  = make(map[string]*game.Game)
	mutex                  sync.Mutex // To protect the games map
	botTimeout             = 10 * time.Second
	disconnectedGamesCache *lru.Cache
)

// A single, dedicated goroutine to handle all matchmaking.
// This should be started once when your server starts.
func init() {
	// Initialize the cache. Let's say we want to store up to 100 disconnected games.
	// If the 101st game is added, the least recently used one is automatically removed.
	var err error
	disconnectedGamesCache, err = lru.New(100)
	if err != nil {
		log.Fatalf("Could not initialize LRU cache: %v", err)
	}
	go Matchmaker()
}

func Matchmaker() {
	log.Println("Matchmaker started...")
	for {
		// Wait for a player to enter the queue
		p1 := <-playerQueue
		log.Printf("Player %s is in the queue, waiting for an opponent or timeout.", p1.Username)

		select {
		case p2 := <-playerQueue:
			// An opponent was found!
			log.Printf("Match found: %s vs %s", p1.Username, p2.Username)
			go startGame(p1, p2)
		case <-time.After(botTimeout):
			// Timeout occurred, start a bot game for p1
			log.Printf("No opponent found for %s, starting a bot game.", p1.Username)
			botPlayer := &Player{Username: "Bot", ID: 2} // Conn is nil for bot
			go startGame(p1, botPlayer)
		}
	}
}

func HandleGame(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	// --- RECONNECTION LOGIC ---
	if val, ok := disconnectedGamesCache.Get(username); ok {
		cachedGame := val.(*CachedGame)
		log.Printf("Player %s is reconnecting to game %s", username, cachedGame.Game.ID)

		if cachedGame.CancelTimer != nil {
			cachedGame.CancelTimer()
		}

		// Find which player is reconnecting and update their connection
		var p1, p2 *Player
		if cachedGame.Player1.Username == username {
			cachedGame.Player1.Conn = conn
			p1 = cachedGame.Player1
			p2 = cachedGame.Player2
		} else {
			cachedGame.Player2.Conn = conn
			p1 = cachedGame.Player1
			p2 = cachedGame.Player2
		}

		// Remove from cache and move back to active games
		disconnectedGamesCache.Remove(p1.Username)
		disconnectedGamesCache.Remove(p2.Username)
		mutex.Lock()
		games[cachedGame.Game.ID] = cachedGame.Game
		mutex.Unlock()

		// Notify players and resume the game
		// Send GAME_START to the reconnecting player
		if p1.Conn != nil {
			p1.Conn.WriteJSON(map[string]interface{}{
				"type":            "GAME_START",
				"game_id":         cachedGame.Game.ID,
				"board":           cachedGame.Game.Board,
				"player_number":   p1.ID,
				"player1_name":    cachedGame.Game.Player1,
				"player2_name":    cachedGame.Game.Player2,
				"starting_player": cachedGame.Game.Turn,
			})
		}

		// Notify opponent if they are still connected
		if p2.Conn != nil {
			p2.Conn.WriteJSON(map[string]interface{}{
				"type":          "OPPONENT_RECONNECTED",
				"message":       "Your opponent has reconnected!",
				"game_id":       cachedGame.Game.ID,
				"board":         cachedGame.Game.Board,
				"next_turn":     cachedGame.Game.Turn,
				"player_number": p2.ID,
				"player1_name":  cachedGame.Game.Player1,
				"player2_name":  cachedGame.Game.Player2,
			})
		}
		go handleGamePlay(cachedGame.Game, p1, p2)
		return
	}

	// --- NEW PLAYER LOGIC ---

	player := &Player{Username: username, Conn: conn}
	log.Printf("Player %s connected and is being added to the queue.", username)

	// Simply add the player to the queue. The Matchmaker goroutine will handle the rest.
	playerQueue <- player
}

// The rest of the file (startGame, handleGamePlay) can remain largely the same.
// Just ensure that the access to the shared 'games' map is protected correctly.

func startGame(p1, p2 *Player) {
	isBotGame := p2.Conn == nil

	id := time.Now().Format("150405") + p1.Username
	g := game.NewGame(id, p1.Username, p2.Username)

	mutex.Lock()
	games[id] = g
	mutex.Unlock()

	p1.ID, p2.ID = 1, 2

	// Message for Player 1
	p1.Conn.WriteJSON(map[string]interface{}{
		"type":            "GAME_START",
		"game_id":         g.ID,
		"board":           g.Board,
		"player_number":   p1.ID,
		"player1_name":    g.Player1,
		"player2_name":    g.Player2,
		"starting_player": g.Turn,
	})

	// Message for Player 2 (only if not a bot)
	if !isBotGame {
		p2.Conn.WriteJSON(map[string]interface{}{
			"type":            "GAME_START",
			"game_id":         g.ID,
			"board":           g.Board,
			"player_number":   p2.ID,
			"player1_name":    g.Player1,
			"player2_name":    g.Player2,
			"starting_player": g.Turn,
		})
	}

	go handleGamePlay(g, p1, p2)
}

func handleGamePlay(g *game.Game, p1, p2 *Player) {
	moves := make(chan Move)
	done := make(chan struct{})
	// Timer goroutine (runs parallel to existing goroutines)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				g.Mutex.Lock()
				if g.Over {
					g.Mutex.Unlock()
					return
				}
				elapsed := time.Since(g.StartTime)
				g.Mutex.Unlock()

				// Send timer update to both players
				timerMsg := map[string]interface{}{
					"type":    "TIMER_UPDATE",
					"elapsed": int(elapsed.Seconds()),
				}

				if p1.Conn != nil {
					p1.Conn.WriteJSON(timerMsg)
				}
				if p2.Conn != nil {
					p2.Conn.WriteJSON(timerMsg)
				}
			}
		}
	}()

	// Goroutine to read messages from player 1
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				var move Move
				if err := p1.Conn.ReadJSON(&move); err != nil {
					select {
					case <-done:
						return
					default:
						log.Println("done channel not closed")
						log.Printf("Player 1 (%s) disconnected: %v", p1.Username, err)
						handleDisconnection(g, p1, p2)
						return
					}
				}
				moves <- move
			}
		}
	}()

	// Conditionally start the goroutine for player 2
	if p2.Conn != nil {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					var move Move
					if err := p2.Conn.ReadJSON(&move); err != nil {
						select {
						case <-done:
							return
						default:
							log.Println("done channel not closed")
							log.Printf("Player 2 (%s) disconnected: %v", p2.Username, err)
							handleDisconnection(g, p2, p1)
							return
						}
					}
					moves <- move
				}
			}
		}()
	} else {
		// Goroutine for bot player moves
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					if g.Turn == 2 {
						time.Sleep(1 * time.Second)

						if g.CheckDraw() {
							return
						}

						bestCol := game.FindBestMove(g, 6)

						botMove := Move{
							Type:   "MOVE",
							Col:    bestCol,
							Player: 2,
						}
						moves <- botMove
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	// Main Game Loop
	for move := range moves {
		g.Mutex.Lock()

		if move.Player != g.Turn {
			g.Mutex.Unlock()
			continue
		}

		row, col, err := g.PlaceDisc(move.Player, move.Col)
		if err != nil {
			log.Printf("Invalid move by player %d: %v", move.Player, err)
			g.Mutex.Unlock()
			continue
		}
		log.Printf("=== After move by Player %d at row=%d, col=%d ===", move.Player, row, col)
		for i := 0; i < len(g.Board); i++ {
			log.Printf("Row %d: %v", i, g.Board[i])
		}
		response := map[string]interface{}{
			"type":      "MOVE",
			"col":       col,
			"row":       row,
			"player":    move.Player,
			"next_turn": g.Turn,
		}
		if p1.Conn != nil {
			p1.Conn.WriteJSON(response)
		}
		if p2.Conn != nil {
			p2.Conn.WriteJSON(response)
		}
		if g.CheckWin(row, col, move.Player) {
			log.Printf("*** WIN DETECTED for Player %d ***", move.Player)
			winnerName := g.Player1
			if move.Player == 2 {
				winnerName = g.Player2
			}
			AddWin(winnerName)
			SaveGame(g.Player1, g.Player2, winnerName, g.Moves)
			msg := map[string]interface{}{
				"type":    "GAME_OVER",
				"message": winnerName + " wins!",
			}
			log.Printf("Game %s ended. Winner: %s", g.ID, winnerName)

			g.Over = true
			g.Mutex.Unlock()

			// Send GAME_OVER message
			if p1.Conn != nil {
				p1.Conn.WriteJSON(msg)
			}
			if p2.Conn != nil {
				p2.Conn.WriteJSON(msg)
			}

			// Close the done channel to stop all goroutines
			close(done)

			// Clean up game from map
			mutex.Lock()
			delete(games, g.ID)
			mutex.Unlock()
			log.Printf("Game %s ended and cleaned up.", g.ID)

			// Keep connections open - let clients close when they're ready
			// This prevents unexpected disconnection that might trigger page reloads
			return

		} else if g.CheckDraw() {
			SaveGame(g.Player1, g.Player2, "draw", g.Moves)
			msg := map[string]interface{}{
				"type":    "GAME_OVER",
				"message": "It's a draw!",
			}
			log.Println("Game ended in a draw.")

			g.Over = true
			g.Mutex.Unlock()

			// Send GAME_OVER message
			if p1.Conn != nil {
				p1.Conn.WriteJSON(msg)
			}
			if p2.Conn != nil {
				p2.Conn.WriteJSON(msg)
			}

			// Close the done channel to stop all goroutines
			close(done)

			// Clean up game from map
			mutex.Lock()
			delete(games, g.ID)
			mutex.Unlock()
			log.Printf("Game %s ended and cleaned up.", g.ID)

			// Keep connections open - let clients close when they're ready
			// This prevents unexpected disconnection that might trigger page reloads
			return
		}

		g.Mutex.Unlock()
	}
}

// Add this constant at the top with your other variables
const reconnectionTimeout = 30 * time.Second

// Modified handleDisconnection function with timer
func handleDisconnection(g *game.Game, disconnectedPlayer, otherPlayer *Player) {
	if g.Over {
		return // Game already over, no need to handle disconnection
	}
	disconnectedPlayer.Conn = nil

	// Remove from active games
	mutex.Lock()
	delete(games, g.ID)
	mutex.Unlock()

	// Create a context with cancel to manage the timer
	ctx, cancel := context.WithCancel(context.Background())
	// Build cached game
	cachedGame := &CachedGame{
		Game:        g,
		Player1:     disconnectedPlayer,
		Player2:     otherPlayer,
		Timestamp:   time.Now(),
		CancelTimer: cancel, // Store the cancel function
	}

	// Use player usernames or IDs as cache keys
	disconnectedGamesCache.Add(disconnectedPlayer.Username, cachedGame)
	disconnectedGamesCache.Add(otherPlayer.Username, cachedGame)

	log.Printf("Game %s moved to cache due to disconnection of %s. Timer started for %v.",
		g.ID, disconnectedPlayer.Username, reconnectionTimeout)

	// Notify the remaining player
	if otherPlayer.Conn != nil {
		otherPlayer.Conn.WriteJSON(map[string]string{
			"type":    "OPPONENT_DISCONNECTED",
			"message": "Your opponent has disconnected. Waiting for them to reconnect...",
		})
	}

	// Start a timer goroutine to handle forfeit after 30 seconds
	go func() {
		timer := time.NewTimer(reconnectionTimeout)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Timer expired - check if player reconnected
			if _, stillInCache := disconnectedGamesCache.Get(disconnectedPlayer.Username); stillInCache {
				log.Printf("Reconnection timeout for %s. Game %s is forfeited.",
					disconnectedPlayer.Username, g.ID)

				// Remove from cache
				disconnectedGamesCache.Remove(disconnectedPlayer.Username)
				disconnectedGamesCache.Remove(otherPlayer.Username)

				// Mark game as over
				g.Mutex.Lock()
				g.Over = true
				g.Mutex.Unlock()

				// Determine winner (the player who stayed connected)
				winnerName := otherPlayer.Username

				// Save the game result
				AddWin(winnerName)
				SaveGame(g.Player1, g.Player2, winnerName, g.Moves)

				// Notify the connected player about the forfeit
				if otherPlayer.Conn != nil {
					otherPlayer.Conn.WriteJSON(map[string]interface{}{
						"type":    "GAME_OVER",
						"message": fmt.Sprintf("%s forfeited. You win!", disconnectedPlayer.Username),
						"reason":  "opponent_timeout",
					})
				}

				log.Printf("Game %s ended due to forfeit. Winner: %s", g.ID, winnerName)
			} else {
				log.Printf("Player %s reconnected within timeout period for game %s.",
					disconnectedPlayer.Username, g.ID)
			}

		case <-ctx.Done():
			// Timer was cancelled due to reconnection
			log.Printf("Timer cancelled for %s - player reconnected to game %s",
				disconnectedPlayer.Username, g.ID)
			return
		}
	}()
}
