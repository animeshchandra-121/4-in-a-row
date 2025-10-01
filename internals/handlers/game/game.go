package game

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Game struct {
	ID        string
	Board     [][]int
	Player1   string
	Player2   string
	Turn      int // 1 or 2 (whose turn)
	Mutex     sync.Mutex
	Over      bool
	Moves     []string
	StartTime time.Time
}

func NewGame(id, p1, p2 string) *Game {
	board := make([][]int, 7) // 7 rows
	for i := range board {
		board[i] = make([]int, 6) // 6 columns
	}
	return &Game{
		ID:        id,
		Board:     board,
		Player1:   p1,
		Player2:   p2,
		Turn:      1, // player1 starts
		Over:      false,
		Moves:     make([]string, 0),
		StartTime: time.Now(),
	}
}

// PlaceDisc tries to drop a disc in a column
func (g *Game) PlaceDisc(player int, col int) (int, int, error) {
	if col < 0 || col >= 6 {
		return -1, -1, errors.New("invalid column")
	}
	if player != g.Turn {
		return -1, -1, errors.New("not your turn")
	}

	// find lowest empty row
	for row := len(g.Board) - 1; row >= 0; row-- {
		if g.Board[row][col] == 0 {
			g.Board[row][col] = player
			g.Moves = append(g.Moves, fmt.Sprintf("%d:%d", col, player))
			// switch turn
			if g.Turn == 1 {
				g.Turn = 2
			} else {
				g.Turn = 1
			}
			return row, col, nil
		}
	}
	return -1, -1, errors.New("column is full")
}

// CheckWin checks if the last move caused a win
func (g *Game) CheckWin(row, col, player int) bool {
	directions := [][]int{
		{0, 1},  // →
		{1, 0},  // ↓
		{1, 1},  // ↘
		{1, -1}, // ↙
	}
	for _, d := range directions {
		count := 1
		// forward
		r, c := row+d[0], col+d[1]
		for r >= 0 && r < 7 && c >= 0 && c < 6 && g.Board[r][c] == player {
			count++
			r += d[0]
			c += d[1]
		}
		// backward
		r, c = row-d[0], col-d[1]
		for r >= 0 && r < 7 && c >= 0 && c < 6 && g.Board[r][c] == player {
			count++
			r -= d[0]
			c -= d[1]
		}
		if count >= 4 {
			return true
		}
	}
	return false
}

// CheckDraw returns true if the board is full
func (g *Game) CheckDraw() bool {
	for c := 0; c < 6; c++ {
		if g.Board[0][c] == 0 {
			return false
		}
	}
	return true
}
