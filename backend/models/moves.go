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

// ActiveCombo is the in-progress state of a single number-card turn: the run of
// consecutive chess moves one player is owed but has not yet finished playing. It
// lives on UnoChessGame between requests so a staggered, frontend-driven turn can
// be resumed one sub-move at a time. It is nil whenever no combo is underway.
//
// WorkingFEN — not ChessEngine — is the source of truth mid-combo: a chess engine
// refuses two moves by the same color in a row, so the combo advances by rewriting
// the FEN. The engine is only re-synced from WorkingFEN once the combo commits.
type ActiveCombo struct {
	Color          chess.Color // the player owed the moves
	Card           UnoCard     // the number card that granted them
	WorkingFEN     string      // board position after the sub-moves played so far
	MovesPlayed    int         // sub-moves completed in this combo
	MovesRemaining int         // sub-moves still owed
	BoardStates    []string    // FEN after each sub-move played so far
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

	// Pending is the number-card combo currently being played out, or nil when the
	// active player has not started (or has already finished) their chess moves.
	Pending *ActiveCombo

	// Captured holds, per color, the piece types that color has lost and may bring
	// back via a +2 / +4 resurrection (rulebook §3B/§3C). Kings never enter this pool:
	// a captured king ends the game, and a king can never be resurrected.
	Captured map[chess.Color][]chess.PieceType
}
