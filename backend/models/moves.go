// Package models has object definitions used in UnoChess
package models

import (
	chess "github.com/notnil/chess"
)

// TurnRecord represents the entire action a player took on their turn.
type TurnRecord struct {
	Player     chess.Color
	CardPlayed UnoCard
	// We store a slice of FEN strings because a single card
	// can result in multiple intermediate board states.
	BoardStates []string
}

type UnoChessGame struct {
	// The notnil chess game instance for validation
	ChessEngine *chess.Game

	// Game History
	History []TurnRecord

	// Hands & Deck
	Hands       map[chess.Color][]UnoCard
	DrawPile    []UnoCard
	DiscardPile []UnoCard

	// Custom Turn Management State
	ActiveColor   chess.Color
	PlayDirection int // 1 for normal, -1 for reversed
}
