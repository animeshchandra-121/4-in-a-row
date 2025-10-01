const WS_URL = 'ws://localhost:8080/ws/game';
const currentUser = localStorage.getItem('currentUser');
const welcomeArea = document.getElementById('welcome-area');
const welcomeMessage = document.getElementById('welcome-message');
const playButton = document.getElementById('play-button');
const statusMessage = document.getElementById('status-message');
const waitingArea = document.getElementById('waiting-area');
const gameArea = document.getElementById('game-area');
const gameGrid = document.getElementById('game-grid');
const currentPlayerInfo = document.getElementById('current-player-info');
const gameInfo = document.getElementById('game-info');

let ws = null;
let gameActive = false;
let myTurn = false;
let playerNumber = 0; // 1 = Yellow, 2 = Red
let board = [];       // 6x7 matrix
let gameEnded = false; // Track if game ended normally
let intentionalClose = false; // Track if we're closing on purpose
let gameTimer = 0; // Timer in seconds

// --- Persistent Logging Setup ---
function persistLog(msg) {
    const logs = JSON.parse(localStorage.getItem("clientLogs") || "[]");
    logs.push({ msg, time: new Date().toISOString() });
    localStorage.setItem("clientLogs", JSON.stringify(logs));
}

const originalLog = console.log;
console.log = function (...args) {
    originalLog.apply(console, args);
    persistLog(args.join(" "));
};

const originalError = console.error;
console.error = function (...args) {
    originalError.apply(console, args);
    persistLog("ERROR: " + args.join(" "));
};

const originalWarn = console.warn;
console.warn = function (...args) {
    originalWarn.apply(console, args);
    persistLog("WARN: " + args.join(" "));
};

function downloadLogs() {
    const logs = localStorage.getItem("clientLogs") || "[]";
    const blob = new Blob([logs], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "clientLogs.json";
    a.click();
    URL.revokeObjectURL(url);
}

document.addEventListener("DOMContentLoaded", () => {
    const logBtn = document.createElement("button");
    logBtn.textContent = "Download Logs";
    logBtn.onclick = downloadLogs;
    document.body.appendChild(logBtn);
});

// --- WebSocket Connection ---
function connectWebSocket() {
    welcomeArea.classList.add('hidden');
    waitingArea.classList.remove('hidden');
    gameEnded = false; // Reset on new connection
    intentionalClose = false; // Reset intentional close flag

    ws = new WebSocket(`${WS_URL}?username=${currentUser}`);

    ws.onopen = () => {
        console.log(`[WebSocket] Connected as ${currentUser}`);
        statusMessage.textContent = `Connected as ${currentUser}. Waiting for opponent...`;
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log("Received message:", data);
        console.log("[WS MESSAGE] Type:", data.type, "| gameActive:", gameActive, "| gameEnded:", gameEnded);
        console.log("[WS MESSAGE] Grid children before handling:", gameGrid.children.length);
        handleMessage(data);
        console.log("[WS MESSAGE] Grid children after handling:", gameGrid.children.length);
    };

    ws.onclose = (event) => {
        console.log("[WebSocket] Connection closed", event.code, event.reason);
        console.log("[WS CLOSE] gameActive:", gameActive, "| gameEnded:", gameEnded, "| intentionalClose:", intentionalClose);
        console.log("[WS CLOSE] Grid children count:", gameGrid.children.length);
        console.log("[WS CLOSE] gameArea hidden?", gameArea.classList.contains('hidden'));
        
        // If we closed intentionally (Return to Lobby), do nothing
        if (intentionalClose) {
            console.log("[WS CLOSE] Intentional close, UI already handled");
            return;
        }
        
        // Only clear the board if the game ended unexpectedly (not via GAME_OVER)
        if (gameEnded) {
            // Game ended normally, keep board visible
            console.log("[WS CLOSE] Game ended normally, keeping board visible");
            statusMessage.textContent = "Game finished.";
        } else if (gameActive) {
            // Unexpected disconnect during active game
            console.log("[WS CLOSE] Unexpected disconnect, clearing board");
            statusMessage.textContent = "Connection lost. Please refresh.";
            gameArea.classList.add("hidden");
            waitingArea.classList.remove("hidden");
        } else {
            console.log("[WS CLOSE] No action taken (not active, not ended)");
        }
    };

    ws.onerror = (error) => {
        console.error("[WebSocket] Error:", error);
        statusMessage.textContent = 'WebSocket Error. Check server.';
    };
}

// --- Message Handling ---
function handleMessage(data) {
    switch (data.type) {
        case 'TIMER_UPDATE':
            gameTimer = data.elapsed;
            updateTimerDisplay();
            break;
        case 'GAME_START':
            gameActive = true;
            gameEnded = false;
            waitingArea.classList.add('hidden');
            gameArea.classList.remove('hidden');
            gameInfo.innerHTML = "";
            statusMessage.textContent = `Game ID: ${data.game_id}`;

            board = data.board;
            playerNumber = data.player_number;
            myTurn = data.starting_player === playerNumber;

            console.log(`[GAME_START] You are Player ${playerNumber}, starting turn: ${myTurn}`);
            initializeBoard(board, data.player1_name, data.player2_name);
            updateTurnMessage();
            break;

        case 'MOVE':
            console.log(`[MOVE] Player ${data.player} placed in column ${data.col} at row ${data.row}`);
            applyMove(data.col, data.player, data.row);  // Pass row from server
            myTurn = (data.next_turn === playerNumber);
            updateTurnMessage();
            break;

        case 'GAME_OVER':
            console.log(`[GAME_OVER] Received. Setting flags...`);
            gameActive = false;
            gameEnded = true; // Mark that game ended normally
            myTurn = false;

            console.log(`[GAME_OVER] Result: ${data.message}`);
            console.log(`[GAME_OVER] Board state before showing result:`, board);
            console.log(`[GAME_OVER] Game grid children count:`, gameGrid.children.length);
            console.log(`[GAME_OVER] gameArea is hidden?`, gameArea.classList.contains('hidden'));
            console.log(`[GAME_OVER] gameArea display style:`, window.getComputedStyle(gameArea).display);
            
            gameInfo.innerHTML = `<strong>${data.message}</strong><br/>`;

            const returnBtn = document.createElement('button');
            returnBtn.textContent = "Return to Lobby";
            returnBtn.onclick = () => {
                console.log("[RETURN TO LOBBY] Button clicked");
                intentionalClose = true; // Mark this as intentional before closing
                
                if (ws && ws.readyState === WebSocket.OPEN) {
                    ws.close(1000, "Player returned to lobby");
                }
                
                // Reset UI
                gameArea.classList.add("hidden");
                waitingArea.classList.add("hidden");
                welcomeArea.classList.remove("hidden");
                statusMessage.textContent = "Welcome back! Click Play to start a new match.";
                gameGrid.innerHTML = "";
                gameEnded = false; // Reset for next game
                intentionalClose = false; // Reset the flag
            };
            gameInfo.appendChild(returnBtn);
            
            console.log(`[GAME_OVER] Finished processing. Board should still be visible.`);
            console.log(`[GAME_OVER] gameArea is hidden NOW?`, gameArea.classList.contains('hidden'));
            
            // Set a timer to check state after 1 second
            setTimeout(() => {
                console.log(`[GAME_OVER+1s] Grid children:`, gameGrid.children.length);
                console.log(`[GAME_OVER+1s] gameArea hidden?`, gameArea.classList.contains('hidden'));
                console.log(`[GAME_OVER+1s] gameEnded flag:`, gameEnded);
            }, 1000);
            break;

        case 'OPPONENT_DISCONNECTED':
            statusMessage.textContent = data.message;
            console.log("[OPPONENT_DISCONNECTED]", data.message);
            break;

        case 'OPPONENT_RECONNECTED':
            statusMessage.textContent = data.message;
            console.log("[OPPONENT_RECONNECTED]", data.message);
            // Update board state in case we missed any moves
            if (data.board) {
                board = data.board;
                initializeBoard(board, data.player1_name, data.player2_name);
            }
            myTurn = (data.next_turn === playerNumber);
            updateTurnMessage();
            break;

        default:
            console.warn("Unknown message:", data.type);
    }
}
// Update timer display
function updateTimerDisplay() {
    const minutes = Math.floor(gameTimer / 60);
    const seconds = gameTimer % 60;
    const timerElement = document.getElementById('timer-display');
    
    if (timerElement) {
        timerElement.textContent = `⏱️ ${minutes}:${seconds.toString().padStart(2, '0')}`;
    } else {
        console.error('Timer display element not found!');
    }
}
// --- Initialize Board ---
function initializeBoard(serverBoard, player1Name, player2Name) {
    console.log("[INIT BOARD] Starting board initialization");
    console.log("[INIT BOARD] Server board:", serverBoard);
    
    gameGrid.innerHTML = '';
    board = serverBoard;

    currentPlayerInfo.textContent = `${player1Name} (Yellow) vs ${player2Name} (Red)`;

    for (let row = 0; row < board.length; row++) {
        for (let col = 0; col < board[row].length; col++) {
            const cell = document.createElement('div');
            cell.classList.add('cell');
            cell.dataset.row = row;
            cell.dataset.col = col;
            cell.addEventListener('click', () => handleCellClick(col));
            gameGrid.appendChild(cell);

            if (board[row][col] !== 0) {
                const disc = document.createElement('div');
                disc.classList.add('disc', board[row][col] === 1 ? 'player1' : 'player2');
                disc.style.transform = "translateY(0)";
                cell.appendChild(disc);
            }
        }
    }
    
    console.log("[INIT BOARD] Board initialized with", gameGrid.children.length, "cells");
}

// --- Handle Cell Click ---
function handleCellClick(col) {
    if (!gameActive || !myTurn) return;

    let row = findAvailableRow(col);
    if (row === -1) return;

    console.log(`[UI] Sending MOVE, col=${col}, player=${playerNumber}`);

    ws.send(JSON.stringify({
        type: 'MOVE',
        col: col,
        player: playerNumber
    }));
}

// --- Apply Move from Server ---
function applyMove(col, player, row) {  // Add row parameter
    board[row][col] = player;

    const cellIndex = row * board[0].length + col;
    const cell = gameGrid.children[cellIndex];

    const disc = document.createElement('div');
    disc.classList.add('disc', player === 1 ? 'player1' : 'player2');
    cell.appendChild(disc);

    setTimeout(() => {
        disc.style.transform = "translateY(0)";
    }, 50);
}

// --- Find lowest empty row in a column ---
function findAvailableRow(col) {
    for (let row = board.length - 1; row >= 0; row--) {
        if (board[row][col] === 0) return row;
    }
    return -1;
}

// --- Update Turn Message ---
function updateTurnMessage() {
    if (!gameActive) return;
    if (myTurn) {
        currentPlayerInfo.innerHTML = `<span class="${playerNumber === 1 ? 'turn-yellow' : 'turn-red'}">Your Turn</span>`;
    } else {
        currentPlayerInfo.innerHTML = `<span class="${playerNumber === 1 ? 'turn-red' : 'turn-yellow'}">Opponent's Turn</span>`;
    }
}

// --- Init ---
document.addEventListener('DOMContentLoaded', () => {
    if (!currentUser) {
        const username = prompt("Enter your username:");
        if (username) {
            localStorage.setItem("currentUser", username);
            window.location.reload();
        } else {
            alert("Username is required!");
            return;
        }
    }
    welcomeMessage.textContent = `Hello, ${currentUser}!`;
    playButton.addEventListener('click', connectWebSocket);
    
    // Add ranking button handler
    const rankingButton = document.getElementById('ranking-button');
    console.log('Ranking button:', rankingButton); // Debug log
    if (rankingButton) {
        rankingButton.addEventListener('click', () => {
            console.log('Ranking button clicked!'); // Debug log
            window.location.href = 'ranking.html';
        });
    } else {
        console.error('Ranking button not found!');
    }
});