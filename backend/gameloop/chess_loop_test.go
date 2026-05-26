package gameloop

import (
	"fmt"
	"strings"
	"testing"

	"github.com/notnil/chess"
)

func InitialiseSampleStarterChessBoard() *chess.Game {
	game := chess.NewGame()
	return game
}

// scriptedChooser plays the given UCI moves in order, one per sub-move. It returns
// nil once the script is exhausted (or if a scripted move isn't legal), which ends
// the combo.
func scriptedChooser(ucis ...string) ChessMoveChooser {
	i := 0
	return func(moves []*chess.Move) *chess.Move {
		if i >= len(ucis) {
			return nil
		}
		want := ucis[i]
		i++
		for _, m := range moves {
			if m.String() == want {
				return m
			}
		}
		return nil
	}
}

// sideToMove pulls the active-color field out of a FEN string.
func sideToMove(fen string) string {
	parts := strings.Split(fen, " ")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func TestChessMultipleMovesProgress(t *testing.T) {
	// The heart of the number-card mechanic: White moves twice in succession with
	// the SAME knight (Nb1-c3 then Nc3-d5) — Black never moves in between.
	start := InitialiseSampleStarterChessBoard().Position().String()

	result, err := PlayConsecutiveChessMoves(start, chess.White, 2, scriptedChooser("b1c3", "c3d5"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Moves) != 2 {
		t.Fatalf("expected 2 sub-moves, got %d", len(result.Moves))
	}
	if len(result.BoardStates) != 2 {
		t.Fatalf("expected 2 recorded board states, got %d", len(result.BoardStates))
	}

	// Every sub-move was White's, so each resulting board shows Black to move.
	// If the turn had alternated normally, the second board would show White.
	for i, fen := range result.BoardStates {
		if stm := sideToMove(fen); stm != "b" {
			t.Errorf("board state %d: side to move = %q, want %q (White should have just moved)", i+1, stm, "b")
		}
	}

	// The knight should now sit on d5, having vacated both b1 and c3.
	board := boardFromFEN(t, result.FinalFEN)
	if got := board.Piece(chess.D5); got != chess.WhiteKnight {
		t.Errorf("expected White knight on d5, got %v", got)
	}
	if got := board.Piece(chess.B1); got != chess.NoPiece {
		t.Errorf("expected b1 to be empty, got %v", got)
	}
	if got := board.Piece(chess.C3); got != chess.NoPiece {
		t.Errorf("expected c3 to be empty, got %v", got)
	}

	if result.Outcome != chess.NoOutcome {
		t.Errorf("game should still be in progress, got outcome %v", result.Outcome)
	}

	t.Logf("final board after White's two-move combo:\n%s", board.Draw())
}

func TestConsecutiveMovesAllSameColor(t *testing.T) {
	// Generalize to N=3 with the default chooser and assert every sub-move was
	// White's (each resulting board hands the move to Black).
	startBoard := InitialiseSampleStarterChessBoard()
	start := startBoard.Position().String()

	result, err := PlayConsecutiveChessMoves(start, chess.White, 3, FirstLegalMove)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Moves) != 3 {
		t.Fatalf("expected 3 sub-moves, got %d", len(result.Moves))
	}
	for i, fen := range result.BoardStates {
		if stm := sideToMove(fen); stm != "b" {
			t.Errorf("board state %d: side to move = %q, want %q", i+1, stm, "b")
		}
	}

	for i, fen := range result.BoardStates {
		board := boardFromFEN(t, fen)
		fmt.Printf("after sub-move %d (%s):\n%s\n", i+1, result.Moves[i], board.Draw())
	}
}

func TestChessCheckmateInterceptEndsCombo(t *testing.T) {
	// White to move; Ra1-a8 is checkmate. Even though we request 3 moves, the combo
	// must stop the instant the game ends.
	const mateIn1 = "6k1/5ppp/8/8/8/8/8/R6K w - - 0 1"

	result, err := PlayConsecutiveChessMoves(mateIn1, chess.White, 3, scriptedChooser("a1a8"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Moves) != 1 {
		t.Fatalf("combo should stop after the mating move, got %d moves", len(result.Moves))
	}
	if result.Outcome != chess.WhiteWon {
		t.Errorf("expected WhiteWon, got %v", result.Outcome)
	}
	if result.Method != chess.Checkmate {
		t.Errorf("expected Checkmate, got %v", result.Method)
	}
}

func boardFromFEN(t *testing.T, fen string) *chess.Board {
	t.Helper()
	cfg, err := chess.FEN(fen)
	if err != nil {
		t.Fatalf("invalid FEN %q: %v", fen, err)
	}
	return chess.NewGame(cfg).Position().Board()
}
