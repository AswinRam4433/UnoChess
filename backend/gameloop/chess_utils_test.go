package gameloop

import (
	"testing"

	"github.com/notnil/chess"
)

func legalMovesFor(t *testing.T, fen string) []*chess.Move {
	t.Helper()
	cfg, err := chess.FEN(fen)
	if err != nil {
		t.Fatalf("invalid FEN %q: %v", fen, err)
	}
	return chess.NewGame(cfg).ValidMoves()
}

func TestPreferCapturesAndChecks_NilOnEmptyMoveList(t *testing.T) {
	if got := PreferCapturesAndChecks(nil); got != nil {
		t.Errorf("expected nil chooser result on empty list, got %v", got)
	}
}

func TestPreferCapturesAndChecks_PicksCapture(t *testing.T) {
	// White pawn e4 captures Black pawn d5. Capture is e4xd5 (UCI "e4d5").
	moves := legalMovesFor(t, "4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1")
	pick := PreferCapturesAndChecks(moves)
	if pick == nil {
		t.Fatal("chooser returned nil for non-empty list")
	}
	if !pick.HasTag(chess.Capture) {
		t.Errorf("expected a capturing move, got %v (tags missing Capture)", pick)
	}
}

func TestPreferCapturesAndChecks_PicksCheckWhenNoCaptureAvailable(t *testing.T) {
	// White queen on d3 can play Qd3-d8+ (check, no capture). No black pieces are
	// adjacent to any white piece, so no capture exists in the move set.
	moves := legalMovesFor(t, "4k3/8/8/8/8/3Q4/8/4K3 w - - 0 1")

	// Sanity check: precondition is "no captures available".
	for _, m := range moves {
		if m.HasTag(chess.Capture) {
			t.Fatalf("test precondition violated: capture %v was generated", m)
		}
	}

	pick := PreferCapturesAndChecks(moves)
	if pick == nil {
		t.Fatal("chooser returned nil for non-empty list")
	}
	if !pick.HasTag(chess.Check) {
		t.Errorf("expected a checking move, got %v (tags missing Check)", pick)
	}
}

func TestPreferCapturesAndChecks_FallsBackToFirstLegal(t *testing.T) {
	// Standard starting position: no captures, no checks → the chooser falls back
	// to moves[0].
	moves := legalMovesFor(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	pick := PreferCapturesAndChecks(moves)
	if pick != moves[0] {
		t.Errorf("expected fallback to moves[0]=%v, got %v", moves[0], pick)
	}
}
