package gameloop

import (
	"errors"
	"testing"

	"github.com/notnil/chess"

	"unochess/models"
)

// newGameAt builds a UnoChessGame whose chess engine starts from the given FEN, with
// White to move. Uno-side fields (hands, deck) are left zero — these tests exercise
// only the chess-combo state machine.
func newGameAt(t *testing.T, fen string) *models.UnoChessGame {
	t.Helper()
	cfg, err := chess.FEN(fen)
	if err != nil {
		t.Fatalf("invalid FEN %q: %v", fen, err)
	}
	return &models.UnoChessGame{
		ChessEngine: chess.NewGame(cfg),
		ActiveColor: chess.White,
	}
}

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func numberCard(v models.CardValue) models.UnoCard {
	return models.UnoCard{Value: v, Color: models.Red}
}

func TestPlaySubMoveDrivesComboToCompletion(t *testing.T) {
	g := newGameAt(t, startFEN)
	if err := StartChessCombo(g, numberCard("2"), 2); err != nil {
		t.Fatalf("StartChessCombo: %v", err)
	}

	// First sub-move: knight b1->c3. Combo is not done yet (one move owed).
	out, err := PlaySubMove(g, "b1c3")
	if err != nil {
		t.Fatalf("first PlaySubMove: %v", err)
	}
	if out.ComboDone {
		t.Fatal("combo should not be done after 1 of 2 moves")
	}
	if out.MovesLeft != 1 {
		t.Errorf("MovesLeft = %d, want 1", out.MovesLeft)
	}
	if g.Pending == nil {
		t.Fatal("Pending should still be set mid-combo")
	}

	// Second sub-move ends the combo: c3->d5.
	out, err = PlaySubMove(g, "c3d5")
	if err != nil {
		t.Fatalf("second PlaySubMove: %v", err)
	}
	if !out.ComboDone {
		t.Error("combo should be done after both moves")
	}
	if out.MovesLeft != 0 {
		t.Errorf("MovesLeft = %d, want 0", out.MovesLeft)
	}

	// Combo cleared, and exactly one TurnRecord recorded with both board states.
	if g.Pending != nil {
		t.Error("Pending should be nil once the combo commits")
	}
	if len(g.History) != 1 {
		t.Fatalf("expected 1 TurnRecord, got %d", len(g.History))
	}
	rec := g.History[0]
	if rec.Player != chess.White {
		t.Errorf("TurnRecord.Player = %v, want White", rec.Player)
	}
	if len(rec.BoardStates) != 2 {
		t.Errorf("TurnRecord.BoardStates len = %d, want 2", len(rec.BoardStates))
	}

	// Engine was re-synced: the knight sits on d5 and it is Black to move.
	board := g.ChessEngine.Position().Board()
	if got := board.Piece(chess.D5); got != chess.WhiteKnight {
		t.Errorf("expected White knight on d5, got %v", got)
	}
	if g.ChessEngine.Position().Turn() != chess.Black {
		t.Errorf("after White's combo it should be Black to move, got %v", g.ChessEngine.Position().Turn())
	}
}

func TestPlaySubMoveRejectsIllegalMove(t *testing.T) {
	g := newGameAt(t, startFEN)
	if err := StartChessCombo(g, numberCard("1"), 1); err != nil {
		t.Fatalf("StartChessCombo: %v", err)
	}

	// e2e5 is not a legal opening move (pawn can't jump three ranks).
	_, err := PlaySubMove(g, "e2e5")
	if !errors.Is(err, ErrIllegalSubMove) {
		t.Fatalf("expected ErrIllegalSubMove, got %v", err)
	}
	// State must be untouched: still mid-combo, nothing recorded.
	if g.Pending == nil {
		t.Error("Pending should survive a rejected move")
	}
	if g.Pending.MovesPlayed != 0 || g.Pending.MovesRemaining != 1 {
		t.Errorf("combo counters changed on a rejected move: played=%d remaining=%d",
			g.Pending.MovesPlayed, g.Pending.MovesRemaining)
	}
	if len(g.History) != 0 {
		t.Errorf("a rejected move should record no TurnRecord, got %d", len(g.History))
	}
}

func TestPlaySubMoveWithoutComboFails(t *testing.T) {
	g := newGameAt(t, startFEN)
	if _, err := PlaySubMove(g, "b1c3"); !errors.Is(err, ErrNoActiveCombo) {
		t.Fatalf("expected ErrNoActiveCombo, got %v", err)
	}
}

func TestStartComboRejectsConcurrentCombo(t *testing.T) {
	g := newGameAt(t, startFEN)
	if err := StartChessCombo(g, numberCard("3"), 3); err != nil {
		t.Fatalf("first StartChessCombo: %v", err)
	}
	if err := StartChessCombo(g, numberCard("3"), 3); !errors.Is(err, ErrComboInProgress) {
		t.Fatalf("expected ErrComboInProgress, got %v", err)
	}
}

func TestPlaySubMoveCheckmateInterceptEndsComboEarly(t *testing.T) {
	// White to move; Ra1-a8 is mate. Even with 3 moves owed, the combo must end at once.
	const mateIn1 = "6k1/5ppp/8/8/8/8/8/R6K w - - 0 1"
	g := newGameAt(t, mateIn1)
	if err := StartChessCombo(g, numberCard("3"), 3); err != nil {
		t.Fatalf("StartChessCombo: %v", err)
	}

	out, err := PlaySubMove(g, "a1a8")
	if err != nil {
		t.Fatalf("PlaySubMove: %v", err)
	}
	if !out.ComboDone {
		t.Error("checkmate should end the combo immediately")
	}
	if out.GameOutcome != chess.WhiteWon {
		t.Errorf("GameOutcome = %v, want WhiteWon", out.GameOutcome)
	}
	if out.GameMethod != chess.Checkmate {
		t.Errorf("GameMethod = %v, want Checkmate", out.GameMethod)
	}
	if g.Pending != nil {
		t.Error("combo should be committed after the mating move")
	}
	if len(g.History) != 1 || len(g.History[0].BoardStates) != 1 {
		t.Errorf("expected 1 TurnRecord with 1 board state, got %d records", len(g.History))
	}
}
