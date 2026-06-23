package gameloop

import (
	"testing"

	"github.com/notnil/chess"
)

// TestKingCaptureEndsGame guards the UnoChess winning condition: during a
// multi-move combo, forceColorClearEnPassant keeps the attacker on move after a
// check, so the engine lists "capture the king" as a legal move. When that move
// is played the game must end immediately with the capturing side as the winner,
// regardless of how many combo moves remain.
func TestKingCaptureEndsGame(t *testing.T) {
	// White pawn on d7 can promote-capture the black king on e8.
	fen := "rnbqkbnr/pppPpppp/8/p7/3P4/8/PPP2PPP/RNBQKBNR w KQkq - 0 1"

	res, err := ApplyChessSubMove(fen, chess.White, true, MoveByUCI("d7e8q"))
	if err != nil {
		t.Fatalf("ApplyChessSubMove: %v", err)
	}
	if res.Status != SubMovePlayed {
		t.Fatalf("status = %d, want SubMovePlayed", res.Status)
	}
	if res.Outcome != chess.WhiteWon {
		t.Errorf("outcome = %q, want WhiteWon (king capture must end the game)", res.Outcome)
	}
}
