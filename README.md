# Connect 4 Game

A web-based implementation of the classic Connect 4 game with a Go backend and Python frontend server.

## Prerequisites

- Go 1.16 or higher
- Git

## Project Structure

```
Connect-4/
├── cmd/                  # Main application entry points
│   └── Connect_4-api/    # Backend API server
├── config/               # Configuration files
├── frontend/             # Frontend web application
├── internal/             # Internal application code
├── go.mod                # Go module file
└── README.md             # This file
```

## Setup Instructions

### Backend (Go)

1. Navigate to the project root directory:
   ```bash
   cd Connect-4
   ```

2. Install Go dependencies:
   ```bash
   go mod download
   ```

3. Run the backend server:
   ```bash
   go run cmd/Connect_4-api/main.go -config config/config.yaml
   ```

   The backend server will start on the port specified in your config file (default is usually 8080).

### Frontend (Python)

1. Open a new terminal window and navigate to the frontend directory:
   ```bash
   cd frontend
   ```

2. Start the Python HTTP server:
   ```bash
   python -m http.server 5500
   ```

3. Open your web browser and navigate to:
   ```
   http://localhost:5500
   ```

## Features

- **User Authentication:** Secure user signup and login system with password hashing.
- **Real-time Multiplayer:** Play against other players in real-time using WebSockets.
- **Automatic Matchmaking:** Players are automatically placed in a queue and matched with the next available opponent.
- **Intelligent Bot Opponent:** If no human opponent is found within 10 seconds, you can play against a challenging AI that uses a minimax algorithm with alpha-beta pruning.
- **Disconnection & Reconnection:** If a player disconnects, they have a 30-second window to rejoin the game before they forfeit.
- **Core Game Logic:** Includes robust win detection for horizontal, vertical, and diagonal lines, as well as draw detection.
- **Game State Management:** The backend tracks the game board, player turns, and game-over status.

## Game Rules

1. Two players take turns dropping their colored discs into a vertical grid.
2. The objective is to be the first to form a horizontal, vertical, or diagonal line of four of one's own discs.
3. The game ends when either player achieves four in a row or the board is full (resulting in a draw).

## Configuration

Edit the `config/config.yaml` file to modify server settings, such as:
- Server port
- Database connection details (if applicable)
- Game settings

## Development

### Backend Dependencies
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Viper](https://github.com/spf13/viper) for configuration

### Frontend Dependencies
- HTML5
- CSS3
- JavaScript (ES6+)

## License

[Specify your license here]

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request
