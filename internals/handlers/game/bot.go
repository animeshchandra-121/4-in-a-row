package game

import (
	"math"
)

const (
	HumanPlayer = 1
	BotPlayer   = 2
)

// Get valid columns where a move can be played
func getValidLocations(board [][]int) []int {
	cols := []int{}
	for c := 0; c < len(board[0]); c++ {
		if board[0][c] == 0 {
			cols = append(cols, c)
		}
	}
	return cols
}

// Copy the game state to simulate moves
func copyGame(g *Game) *Game {
	newBoard := make([][]int, len(g.Board))
	for i := range g.Board {
		newBoard[i] = make([]int, len(g.Board[i]))
		copy(newBoard[i], g.Board[i])
	}
	return &Game{
		ID:      g.ID,
		Board:   newBoard,
		Player1: g.Player1,
		Player2: g.Player2,
		Turn:    g.Turn,
		Over:    g.Over,
		Moves:   append([]string{}, g.Moves...),
	}
}

// Heuristic scoring for the board
func scorePosition(board [][]int, piece int) int {
	score := 0
	rows := len(board)
	cols := len(board[0])

	// Center column preference
	centerCol := cols / 2
	centerCount := 0
	for r := 0; r < rows; r++ {
		if board[r][centerCol] == piece {
			centerCount++
		}
	}
	score += centerCount * 3

	// Horizontal
	for r := 0; r < rows; r++ {
		for c := 0; c < cols-3; c++ {
			window := []int{board[r][c], board[r][c+1], board[r][c+2], board[r][c+3]}
			score += evaluateWindow(window, piece)
		}
	}

	// Vertical
	for c := 0; c < cols; c++ {
		for r := 0; r < rows-3; r++ {
			window := []int{board[r][c], board[r+1][c], board[r+2][c], board[r+3][c]}
			score += evaluateWindow(window, piece)
		}
	}

	// Positive diagonal
	for r := 0; r < rows-3; r++ {
		for c := 0; c < cols-3; c++ {
			window := []int{board[r][c], board[r+1][c+1], board[r+2][c+2], board[r+3][c+3]}
			score += evaluateWindow(window, piece)
		}
	}

	// Negative diagonal
	for r := 3; r < rows; r++ {
		for c := 0; c < cols-3; c++ {
			window := []int{board[r][c], board[r-1][c+1], board[r-2][c+2], board[r-3][c+3]}
			score += evaluateWindow(window, piece)
		}
	}

	return score
}

// Evaluate 4-slot window
func evaluateWindow(window []int, piece int) int {
	score := 0
	opp := HumanPlayer
	if piece == HumanPlayer {
		opp = BotPlayer
	}

	countPiece, countOpp, countEmpty := 0, 0, 0
	for _, v := range window {
		if v == piece {
			countPiece++
		} else if v == opp {
			countOpp++
		} else {
			countEmpty++
		}
	}

	if countPiece == 4 {
		score += 100
	} else if countPiece == 3 && countEmpty == 1 {
		score += 5
	} else if countPiece == 2 && countEmpty == 2 {
		score += 2
	}

	if countOpp == 3 && countEmpty == 1 {
		score -= 4
	}

	return score
}

// Minimax with alpha-beta pruning
func minimax(g *Game, depth int, alpha, beta float64, maximizingPlayer bool) (int, float64) {
	validLocations := getValidLocations(g.Board)
	isDraw := g.CheckDraw()

	// Base case
	if depth == 0 || isDraw {
		if isDraw {
			return -1, 0 // Neutral score for draw
		}
		netScore := float64(scorePosition(g.Board, BotPlayer) - scorePosition(g.Board, HumanPlayer))
		return -1, netScore
	}

	if maximizingPlayer {
		value := math.Inf(-1)
		bestCol := validLocations[0]
		for _, col := range validLocations {
			temp := copyGame(g)
			row, _, err := temp.PlaceDisc(BotPlayer, col)
			if err != nil {
				continue
			}
			if temp.CheckWin(row, col, BotPlayer) {
				return col, 10000 // Winning move
			}
			_, newScore := minimax(temp, depth-1, alpha, beta, false)
			if newScore > value {
				value = newScore
				bestCol = col
			}
			alpha = math.Max(alpha, value)
			if alpha >= beta {
				break
			}
		}
		return bestCol, value
	} else {
		value := math.Inf(1)
		bestCol := validLocations[0]
		for _, col := range validLocations {
			temp := copyGame(g)
			row, _, err := temp.PlaceDisc(HumanPlayer, col)
			if err != nil {
				continue
			}
			if temp.CheckWin(row, col, HumanPlayer) {
				return col, -10000 // Opponent wins
			}
			_, newScore := minimax(temp, depth-1, alpha, beta, true)
			if newScore < value {
				value = newScore
				bestCol = col
			}
			beta = math.Min(beta, value)
			if alpha >= beta {
				break
			}
		}
		return bestCol, value
	}
}

// Public function to find best move for bot
func FindBestMove(g *Game, depth int) int {
	col, _ := minimax(g, depth, math.Inf(-1), math.Inf(1), true)
	return col
}
