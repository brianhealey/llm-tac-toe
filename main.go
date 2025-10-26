package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Board [3][3]string

type Move struct {
	Player   string
	Position int
}

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

const (
	PlayerX = "X"
	PlayerO = "O"
	Empty   = " "
)

// DisplayBoard prints the current board state to the console
func DisplayBoard(board Board) {
	fmt.Println("\n  0 | 1 | 2")
	fmt.Println(" -----------")
	for i := 0; i < 3; i++ {
		fmt.Printf("%d %s | %s | %s\n", i, board[i][0], board[i][1], board[i][2])
		if i < 2 {
			fmt.Println(" -----------")
		}
	}
	fmt.Println()
}

// InitBoard creates a new empty board
func InitBoard() Board {
	var board Board
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			board[i][j] = Empty
		}
	}
	return board
}

// CheckWinner checks if there's a winner
func CheckWinner(board Board) string {
	// Check rows
	for i := 0; i < 3; i++ {
		if board[i][0] != Empty && board[i][0] == board[i][1] && board[i][1] == board[i][2] {
			return board[i][0]
		}
	}

	// Check columns
	for j := 0; j < 3; j++ {
		if board[0][j] != Empty && board[0][j] == board[1][j] && board[1][j] == board[2][j] {
			return board[0][j]
		}
	}

	// Check diagonals
	if board[0][0] != Empty && board[0][0] == board[1][1] && board[1][1] == board[2][2] {
		return board[0][0]
	}
	if board[0][2] != Empty && board[0][2] == board[1][1] && board[1][1] == board[2][0] {
		return board[0][2]
	}

	return ""
}

// IsBoardFull checks if the board is full (draw)
func IsBoardFull(board Board) bool {
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == Empty {
				return false
			}
		}
	}
	return true
}

// IsValidMove checks if a move is valid
func IsValidMove(board Board, row, col int) bool {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return false
	}
	return board[row][col] == Empty
}

// MakeMove places a player's mark on the board
func MakeMove(board *Board, player string, row, col int) bool {
	if IsValidMove(*board, row, col) {
		board[row][col] = player
		return true
	}
	return false
}

// DetectThreats analyzes the board for winning and blocking opportunities
func DetectThreats(board Board, player string) (winningMoves []int, blockingMoves []int) {
	opponent := PlayerO
	if player == PlayerO {
		opponent = PlayerX
	}

	// All winning combinations: [3]int{pos1, pos2, pos3}
	winningCombinations := [][3]int{
		// Rows
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		// Columns
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		// Diagonals
		{0, 4, 8}, {2, 4, 6},
	}

	for _, combo := range winningCombinations {
		pos1, pos2, pos3 := combo[0], combo[1], combo[2]
		row1, col1 := pos1/3, pos1%3
		row2, col2 := pos2/3, pos2%3
		row3, col3 := pos3/3, pos3%3

		cell1 := board[row1][col1]
		cell2 := board[row2][col2]
		cell3 := board[row3][col3]

		// Check if player can win (has 2 marks and 1 empty)
		playerCount := 0
		emptyCount := 0
		emptyPos := -1

		if cell1 == player {
			playerCount++
		} else if cell1 == Empty {
			emptyCount++
			emptyPos = pos1
		}

		if cell2 == player {
			playerCount++
		} else if cell2 == Empty {
			emptyCount++
			emptyPos = pos2
		}

		if cell3 == player {
			playerCount++
		} else if cell3 == Empty {
			emptyCount++
			emptyPos = pos3
		}

		if playerCount == 2 && emptyCount == 1 {
			winningMoves = append(winningMoves, emptyPos)
		}

		// Check if opponent can win (needs blocking)
		opponentCount := 0
		emptyCount = 0
		emptyPos = -1

		if cell1 == opponent {
			opponentCount++
		} else if cell1 == Empty {
			emptyCount++
			emptyPos = pos1
		}

		if cell2 == opponent {
			opponentCount++
		} else if cell2 == Empty {
			emptyCount++
			emptyPos = pos2
		}

		if cell3 == opponent {
			opponentCount++
		} else if cell3 == Empty {
			emptyCount++
			emptyPos = pos3
		}

		if opponentCount == 2 && emptyCount == 1 {
			blockingMoves = append(blockingMoves, emptyPos)
		}
	}

	return winningMoves, blockingMoves
}

// BuildPrompt creates the prompt for the LLM with game history
func BuildPrompt(board Board, player string, moveHistory []Move) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("You are playing Tic-Tac-Toe as player %s.\n\n", player))

	// Show move history
	if len(moveHistory) > 0 {
		prompt.WriteString("Move history:\n")
		for i, move := range moveHistory {
			prompt.WriteString(fmt.Sprintf("%d. Player %s played position %d\n",
				i+1, move.Player, move.Position))
		}
		prompt.WriteString("\n")
	}

	// Show current board state with position numbers for empty spaces
	prompt.WriteString("Current board (empty spaces show their position number):\n")
	prompt.WriteString("-------------\n")
	for i := 0; i < 3; i++ {
		prompt.WriteString("| ")
		for j := 0; j < 3; j++ {
			if board[i][j] == Empty {
				prompt.WriteString(fmt.Sprintf("%d ", i*3+j))
			} else {
				prompt.WriteString(fmt.Sprintf("%s ", board[i][j]))
			}
			prompt.WriteString("| ")
		}
		prompt.WriteString("\n-------------\n")
	}

	// List available positions explicitly
	var availablePositions []int
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] == Empty {
				availablePositions = append(availablePositions, i*3+j)
			}
		}
	}

	// List taken positions
	var takenPositions []int
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if board[i][j] != Empty {
				takenPositions = append(takenPositions, i*3+j)
			}
		}
	}

	if len(takenPositions) > 0 {
		prompt.WriteString("\n⛔ POSITIONS ALREADY TAKEN (DO NOT USE): ")
		for i, pos := range takenPositions {
			if i > 0 {
				prompt.WriteString(", ")
			}
			prompt.WriteString(fmt.Sprintf("%d", pos))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("\n✅ AVAILABLE POSITIONS (CHOOSE ONE OF THESE): ")
	for i, pos := range availablePositions {
		if i > 0 {
			prompt.WriteString(", ")
		}
		prompt.WriteString(fmt.Sprintf("%d", pos))
	}
	prompt.WriteString("\n")

	// Detect threats on the board
	winningMoves, blockingMoves := DetectThreats(board, player)

	// Determine opponent
	opponent := PlayerO
	if player == PlayerO {
		opponent = PlayerX
	}

	// Explicitly tell the LLM about threats
	prompt.WriteString("\n*** CRITICAL ANALYSIS ***\n")
	if len(winningMoves) > 0 {
		prompt.WriteString(fmt.Sprintf("🎯 YOU CAN WIN NOW! Play position %d to win immediately!\n", winningMoves[0]))
		prompt.WriteString(fmt.Sprintf("WINNING MOVE DETECTED: Position %d will give you three in a row!\n", winningMoves[0]))
	} else if len(blockingMoves) > 0 {
		prompt.WriteString(fmt.Sprintf("⚠️  DANGER! %s can win with position %d! You MUST BLOCK IT!\n", opponent, blockingMoves[0]))
		prompt.WriteString(fmt.Sprintf("BLOCKING REQUIRED: If you don't play position %d, %s will win next turn!\n", blockingMoves[0], opponent))
	} else {
		prompt.WriteString("No immediate wins or threats detected. Play strategically.\n")
		prompt.WriteString("Best strategy: Take center (4) if available, then corners (0,2,6,8), then edges (1,3,5,7)\n")
	}
	prompt.WriteString("*** END ANALYSIS ***\n")

	prompt.WriteString("\nSTRATEGY PRIORITY:\n")
	prompt.WriteString("1. WIN: Play winning moves immediately\n")
	prompt.WriteString(fmt.Sprintf("2. BLOCK: Block %s's winning moves immediately\n", opponent))
	prompt.WriteString("3. STRATEGIC: Otherwise, prefer center (4), then corners (0,2,6,8), then edges (1,3,5,7)\n")

	prompt.WriteString("\n⚠️  CRITICAL INSTRUCTIONS:\n")
	prompt.WriteString("1. You MUST choose ONLY from the AVAILABLE POSITIONS list above\n")
	if len(takenPositions) > 0 {
		prompt.WriteString(fmt.Sprintf("2. NEVER choose positions that are taken: %v\n", takenPositions))
	}
	prompt.WriteString(fmt.Sprintf("3. ONLY respond with ONE number from: %v\n", availablePositions))
	prompt.WriteString("4. Do NOT include any other text, explanation, or formatting\n")
	prompt.WriteString("5. Your response should be a SINGLE digit only\n")

	return prompt.String()
}

// CallLLM makes a request to Ollama API
func CallLLM(prompt string, ollamaURL string, model string) (string, error) {
	reqBody := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(ollamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	err = json.Unmarshal(body, &ollamaResp)
	if err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

// ParseMove extracts the position from LLM response
func ParseMove(response string) (int, error) {
	// Clean the response
	response = strings.TrimSpace(response)

	// Try to find a single digit 0-8
	re := regexp.MustCompile(`[0-8]`)
	match := re.FindString(response)

	if match == "" {
		return -1, fmt.Errorf("no valid position found in response: %s", response)
	}

	position, err := strconv.Atoi(match)
	if err != nil {
		return -1, err
	}

	return position, nil
}

type GameStats struct {
	XWins  int
	OWins  int
	Draws  int
	Errors int
	Total  int
}

// PlayGame runs a single game and returns the winner ("X", "O", "draw", or "error")
func PlayGame(ollamaURL, model string, maxRetries int, debug bool, gameNumber int) string {
	// Initialize game
	board := InitBoard()
	var moveHistory []Move
	currentPlayer := PlayerX

	if gameNumber > 0 {
		fmt.Printf("\n=== Game %d ===\n", gameNumber)
	}

	DisplayBoard(board)

	// Game loop
	for {
		fmt.Printf("\n--- Player %s's turn ---\n", currentPlayer)

		// Build prompt with move history
		prompt := BuildPrompt(board, currentPlayer, moveHistory)

		if debug {
			fmt.Println("\n========== PROMPT DEBUG ==========")
			fmt.Println(prompt)
			fmt.Println("==================================\n")
		}

		var position int
		validMove := false

		// Try to get a valid move from LLM
		for retry := 0; retry < maxRetries; retry++ {
			fmt.Printf("Requesting move from LLM (attempt %d/%d)...\n", retry+1, maxRetries)

			response, err := CallLLM(prompt, ollamaURL, model)
			if err != nil {
				fmt.Printf("Error calling LLM: %v\n", err)
				continue
			}

			fmt.Printf("LLM response: %s\n", strings.TrimSpace(response))

			position, err = ParseMove(response)
			if err != nil {
				fmt.Printf("Error parsing move: %v\n", err)
				continue
			}

			row := position / 3
			col := position % 3

			if MakeMove(&board, currentPlayer, row, col) {
				validMove = true
				moveHistory = append(moveHistory, Move{Player: currentPlayer, Position: position})
				fmt.Printf("Player %s plays position %d (row %d, col %d)\n", currentPlayer, position, row, col)
				break
			} else {
				fmt.Printf("Invalid move: position %d is already taken or out of bounds\n", position)
			}
		}

		if !validMove {
			fmt.Printf("Player %s failed to make a valid move after %d attempts. Game over.\n", currentPlayer, maxRetries)
			fmt.Printf("Total moves played: %d\n", len(moveHistory))
			return "error"
		}

		// Display updated board
		DisplayBoard(board)

		// Check for winner
		winner := CheckWinner(board)
		if winner != "" {
			fmt.Printf("🎉 Player %s wins!\n", winner)
			fmt.Printf("Total moves played: %d\n", len(moveHistory))
			return winner
		}

		// Check for draw
		if IsBoardFull(board) {
			fmt.Println("🤝 It's a draw!")
			fmt.Printf("Total moves played: %d\n", len(moveHistory))
			return "draw"
		}

		// Switch player
		if currentPlayer == PlayerX {
			currentPlayer = PlayerO
		} else {
			currentPlayer = PlayerX
		}
	}
}

func main() {
	// Configuration flags
	ollamaURL := flag.String("url", "http://localhost:11434", "Ollama/LMStudio API URL")
	model := flag.String("model", "llama3.2", "Model to use (e.g., llama3.2, llama3.1:70b, qwen2.5, mistral)")
	maxRetries := flag.Int("retries", 3, "Maximum retries for invalid moves")
	debug := flag.Bool("debug", false, "Show full prompts sent to LLM")
	games := flag.Int("games", 1, "Number of games to play (0 for unlimited)")
	flag.Parse()

	fmt.Println("=== Tic-Tac-Toe: LLM vs LLM ===")
	fmt.Printf("Using model: %s\n", *model)
	fmt.Printf("Ollama URL: %s\n", *ollamaURL)
	fmt.Printf("Max retries: %d\n", *maxRetries)
	if *games == 0 {
		fmt.Println("Games to play: Unlimited")
	} else {
		fmt.Printf("Games to play: %d\n", *games)
	}

	stats := GameStats{}
	gameNumber := 1

	// Game loop
	for {
		// Check if we've reached the game limit (unless unlimited)
		if *games > 0 && gameNumber > *games {
			break
		}

		result := PlayGame(*ollamaURL, *model, *maxRetries, *debug, gameNumber)

		// Update statistics
		stats.Total++
		switch result {
		case PlayerX:
			stats.XWins++
		case PlayerO:
			stats.OWins++
		case "draw":
			stats.Draws++
		case "error":
			stats.Errors++
		}

		gameNumber++

		// For unlimited games, allow graceful exit
		if *games == 0 {
			fmt.Println("\nPress Ctrl+C to stop, or the next game will start in 2 seconds...")
			time.Sleep(2 * time.Second)
		}
	}

	// Print final statistics
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("FINAL STATISTICS")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("Total games played: %d\n", stats.Total)
	fmt.Printf("Player X wins:      %d (%.1f%%)\n", stats.XWins, float64(stats.XWins)/float64(stats.Total)*100)
	fmt.Printf("Player O wins:      %d (%.1f%%)\n", stats.OWins, float64(stats.OWins)/float64(stats.Total)*100)
	fmt.Printf("Draws:              %d (%.1f%%)\n", stats.Draws, float64(stats.Draws)/float64(stats.Total)*100)
	if stats.Errors > 0 {
		fmt.Printf("Errors:             %d (%.1f%%)\n", stats.Errors, float64(stats.Errors)/float64(stats.Total)*100)
	}
	fmt.Println(strings.Repeat("=", 50))
}
