package gameloop

import (
	"strings"

	"github.com/notnil/chess"
)

// ChessMoveChooser selects which legal move to play for a single sub-move of a
// multi-move turn. Returning nil ends the combo early. This is the seam where a
// real strategy, AI, or human input plugs in.
type ChessMoveChooser func(moves []*chess.Move) *chess.Move

// FirstLegalMove is a trivial deterministic chooser that always plays the first
// legal move. Handy as a placeholder bot and in tests.
func FirstLegalMove(moves []*chess.Move) *chess.Move {
	if len(moves) == 0 {
		return nil
	}
	return moves[0]
}

// PreferCapturesAndChecks is the bot driver's chess chooser: it picks the first
// capturing move, then the first checking move, and finally falls back to the first
// legal move. It is intentionally minimal — enough to keep bot games varied and out
// of the trivial rook-shuffle pattern that FirstLegalMove produces, without
// committing to a real engine. Pluggable via gameloop.RunOptions for tests that
// want stricter determinism.
func PreferCapturesAndChecks(moves []*chess.Move) *chess.Move {
	if len(moves) == 0 {
		return nil
	}
	for _, m := range moves {
		if m.HasTag(chess.Capture) {
			return m
		}
	}
	for _, m := range moves {
		if m.HasTag(chess.Check) {
			return m
		}
	}
	return moves[0]
}

// MoveByUCI returns a chooser that plays the legal move matching the given UCI
// string (e.g. "b1a3", "e7e8q"), or declines (returns nil) when no legal move
// matches. It is the human-facing counterpart to FirstLegalMove: the move a player
// picked on the frontend, validated against the authoritative legal-move set rather
// than trusted. A per-request handler treats a nil result as an illegal submission.
func MoveByUCI(uci string) ChessMoveChooser {
	return func(moves []*chess.Move) *chess.Move {
		for _, m := range moves {
			if m.String() == uci {
				return m
			}
		}
		return nil
	}
}

// forceColorClearEnPassant rewrites a FEN so it is `color`'s move and clears any
// en passant target square. Clearing en passant is required when the same color
// moves twice in a row: the square left behind by our own pawn's double-step is
// not something we can capture, so it must not leak into the next sub-move.
func forceColorClearEnPassant(fen string, color chess.Color) string {
	parts := strings.Split(fen, " ")
	if len(parts) < 6 {
		return fen
	}
	if color == chess.White {
		parts[1] = "w"
	} else {
		parts[1] = "b"
	}
	parts[3] = "-"
	return strings.Join(parts, " ")
}
