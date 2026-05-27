package gameloop

import (
	"github.com/notnil/chess"

	"unochess/models"
)

// startingHandSize is the number of Uno cards each player is dealt at setup.
const startingHandSize = 7

// NewUnoChessGame builds a fresh two-player UnoChess game: a shuffled Uno deck dealt
// 7 cards each to White and Black, a discard pile seeded with one card, and a chess
// board in its standard starting position. White moves first.
//
// A wild card flipped as the opening discard is left colorless here; declaring its
// starting color is a turn-flow concern handled by the orchestrator (Phase 2).
func NewUnoChessGame() *models.UnoChessGame {
	drawPile := InitialiseFullUnoDeck()

	hands := map[chess.Color]models.Deck{
		chess.White: drawPile.DealStartingUnoCards(startingHandSize),
		chess.Black: drawPile.DealStartingUnoCards(startingHandSize),
	}

	// Seed the discard pile with the top card of what remains in the draw pile.
	topCard := drawPile[0]
	drawPile = drawPile[1:]

	return &models.UnoChessGame{
		ChessEngine:   chess.NewGame(),
		History:       []models.TurnRecord{},
		Hands:         hands,
		DrawPile:      drawPile,
		DiscardPile:   models.Deck{topCard},
		ActiveColor:   chess.White,
		PlayDirection: 1,
		Captured:      map[chess.Color][]chess.PieceType{},
	}
}
