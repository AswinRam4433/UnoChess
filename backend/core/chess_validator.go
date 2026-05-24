// Package core has the logic and validations for UnoChess
package core

import (
	"fmt"
	"strings"

	"github.com/notnil/chess"
)

// ForceTurnInFEN is a utility function that rewrites the active turn indicator
// in a FEN string to match the current player's color.
func ForceTurnInFEN(fen string, color chess.Color) string {
	parts := strings.Split(fen, " ")
	if len(parts) < 2 {
		return fen
	}
	if color == chess.White {
		parts[1] = "w"
	} else {
		parts[1] = "b"
	}
	return strings.Join(parts, " ")
}

// GetValidChessMovesForSubMove returns all valid chess moves for the active player
// given a specific intermediate FEN board state.
func GetValidChessMovesForSubMove(currentFEN string, activeColor chess.Color) ([]*chess.Move, error) {
	// 1. Force the engine to look at the board from the active player's perspective
	forcedFEN := ForceTurnInFEN(currentFEN, activeColor)

	fenFunc, err := chess.FEN(forcedFEN)
	if err != nil {
		return nil, fmt.Errorf("invalid FEN string: %w", err)
	}

	// 2. Initialize a temporary validation engine
	tempEngine := chess.NewGame(fenFunc)

	// 3. Return the legal moves computed by the package for this position
	return tempEngine.ValidMoves(), nil
}

// ValidateChessMoveChain takes a starting FEN string and an array of algebraic moves
// (e.g., ["e2e4", "e4e5"]) and verifies if the entire sequence can be executed
// sequentially by the active player. It returns the final FEN string if successful.
func ValidateChessMoveChain(startingFEN string, activeColor chess.Color, movesToValidate []string) (string, error) {
	currentFEN := startingFEN

	for i, moveStr := range movesToValidate {
		// 1. Ensure the board turn flag matches our active player before calculating valid moves
		forcedFEN := ForceTurnInFEN(currentFEN, activeColor)
		fenFunc, err := chess.FEN(forcedFEN)
		if err != nil {
			return "", fmt.Errorf("error parsing FEN at step %d: %w", i, err)
		}

		tempEngine := chess.NewGame(fenFunc)
		validMoves := tempEngine.ValidMoves()

		// 2. Look for the user's move inside the generated valid moves
		var matchedMove *chess.Move
		for _, vm := range validMoves {
			// Comparing against UCI / algebraic notation (e.g., "e2e4")
			if vm.String() == moveStr {
				matchedMove = vm
				break
			}
		}

		if matchedMove == nil {
			return "", fmt.Errorf("move %d (%s) is invalid from the current position", i+1, moveStr)
		}

		// 3. Execute the move on our temporary engine to generate the next intermediate FEN state
		err = tempEngine.Move(matchedMove)
		if err != nil {
			return "", fmt.Errorf("failed to execute move %d: %w", i+1, err)
		}

		// Update the FEN for the next step of the loop
		currentFEN = tempEngine.Position().String()
	}

	// Return the final board state after all moves are successfully calculated
	return currentFEN, nil
}
