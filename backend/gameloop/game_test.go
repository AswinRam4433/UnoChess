package gameloop

import (
	"testing"

	"github.com/notnil/chess"
)

func TestNewUnoChessGameSetup(t *testing.T) {
	g := NewUnoChessGame()

	// Each player is dealt a hand of 7.
	if got := len(g.Hands[chess.White]); got != startingHandSize {
		t.Errorf("White hand = %d cards, want %d", got, startingHandSize)
	}
	if got := len(g.Hands[chess.Black]); got != startingHandSize {
		t.Errorf("Black hand = %d cards, want %d", got, startingHandSize)
	}

	// The discard pile is seeded with exactly one card.
	if len(g.DiscardPile) != 1 {
		t.Fatalf("discard pile = %d cards, want 1", len(g.DiscardPile))
	}

	// Card conservation: the full 104-card house deck is fully accounted for across
	// hands, discard, and the draw pile — none lost, none duplicated.
	const fullDeck = 104
	total := len(g.Hands[chess.White]) + len(g.Hands[chess.Black]) + len(g.DiscardPile) + len(g.DrawPile)
	if total != fullDeck {
		t.Errorf("cards not conserved: %d on table, want %d", total, fullDeck)
	}
	if want := fullDeck - 2*startingHandSize - 1; len(g.DrawPile) != want {
		t.Errorf("draw pile = %d, want %d", len(g.DrawPile), want)
	}

	// The chess board starts in the standard position with White to move.
	if g.ChessEngine == nil {
		t.Fatal("ChessEngine is nil")
	}
	if turn := g.ChessEngine.Position().Turn(); turn != chess.White {
		t.Errorf("chess side to move = %v, want White", turn)
	}

	// Turn-management defaults.
	if g.ActiveColor != chess.White {
		t.Errorf("ActiveColor = %v, want White", g.ActiveColor)
	}
	if g.PlayDirection != 1 {
		t.Errorf("PlayDirection = %d, want 1", g.PlayDirection)
	}

	// Combo and capture state start clean.
	if g.Pending != nil {
		t.Error("Pending should be nil at game start")
	}
	if len(g.Captured[chess.White]) != 0 || len(g.Captured[chess.Black]) != 0 {
		t.Error("captured pools should start empty")
	}
}
