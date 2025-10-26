# Llama Tic-Tac-Toe

An AI-powered Tic-Tac-Toe game where two LLMs play against each other using Ollama, LM Studio, or Llama.

## Features

- Two LLMs play Tic-Tac-Toe against each other
- Visual board display in the console
- Complete move history tracking
- Each move prompt includes all previous plays
- Smart retry logic for invalid moves
- Win and draw detection
- **Multi-game support** with statistics tracking
- **Unlimited games mode** for continuous play
- Intelligent threat detection (win/block analysis)

## Prerequisites

1. **Go**: Install Go 1.20 or later
2. **Ollama** (or LM Studio/Llama): Have one of these running locally
   - For Ollama: Install from https://ollama.ai
   - Run: `ollama pull llama3.2` (or your preferred model)
   - Make sure Ollama is running on `http://localhost:11434`

## Installation

```bash
go mod download
```

## Usage

Basic usage:
```bash
go run main.go
```

With options:
```bash
# Use a different model
go run main.go -model llama3.1:70b

# Use a different API endpoint (LM Studio)
go run main.go -url http://localhost:1234

# Enable debug mode to see prompts
go run main.go -debug

# Play multiple games and see statistics
go run main.go -games 10

# Play unlimited games (Ctrl+C to stop)
go run main.go -games 0

# Combine options for advanced usage
go run main.go -model llama3.1:8b-instruct-q4_1 -games 5
```

## Configuration Options

Use command-line flags to configure the game:

- `-url` : API URL (default: `http://localhost:11434`)
- `-model` : Model name (default: `llama3.2`)
  - Try: `llama3.1:70b`, `qwen2.5`, `mistral`, `llama3.1:8b-instruct-q4_1`
- `-retries` : Max retry attempts for invalid moves (default: `3`)
- `-debug` : Show full prompts sent to LLM (default: `false`)
- `-games` : Number of games to play (default: `1`, use `0` for unlimited)

### Using LM Studio or Llama

Use the `-url` flag to point to your LM Studio or other compatible API endpoint:
```bash
go run main.go -url http://localhost:1234 -model your-model-name
```

## How It Works

1. The game initializes an empty 3x3 board
2. Player X starts first
3. For each turn:
   - The LLM receives a prompt with:
     - Complete move history (all previous plays by both players)
     - Current board state
     - Instructions on how to respond
   - The LLM responds with a position (0-8)
   - The move is validated and applied to the board
   - The board is displayed
   - Win/draw conditions are checked
4. Players alternate until the game ends

## Position Mapping

```
0 | 1 | 2
---------
3 | 4 | 5
---------
6 | 7 | 8
```

## Example Output

Single game:
```
=== Tic-Tac-Toe: LLM vs LLM ===
Using model: llama3.2
Ollama URL: http://localhost:11434

  0 | 1 | 2
 -----------
0   |   |
 -----------
1   |   |
 -----------
2   |   |

--- Player X's turn ---
Requesting move from LLM (attempt 1/3)...
LLM response: 4
Player X plays position 4 (row 1, col 1)

  0 | 1 | 2
 -----------
0   |   |
 -----------
1   | X |
 -----------
2   |   |
```

Multiple games with statistics:
```
=== Tic-Tac-Toe: LLM vs LLM ===
Using model: llama3.1:8b-instruct-q4_1
Ollama URL: http://localhost:11434
Max retries: 3
Games to play: 10

... games play ...

==================================================
FINAL STATISTICS
==================================================
Total games played: 10
Player X wins:      6 (60.0%)
Player O wins:      3 (30.0%)
Draws:              1 (10.0%)
==================================================
```

## License

MIT
