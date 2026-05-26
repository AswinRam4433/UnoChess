package gameloop

import (
	"errors"
	"testing"

	"github.com/notnil/chess"

	"unochess/models"
)

func drawCard(v models.CardValue) models.UnoCard {
	return models.UnoCard{Value: v, Color: models.Red}
}

func sq(f chess.File, r chess.Rank) chess.Square {
	return chess.NewSquare(f, r)
}

func TestResurrectionCount(t *testing.T) {
	cases := []struct {
		card models.CardValue
		want int
	}{
		{models.Pl2, 2},
		{models.Pl4, 4},
		{models.WildCard, 0},
		{models.Skip, 0},
		{"3", 0},
	}
	for _, c := range cases {
		if got := ResurrectionCount(drawCard(c.card)); got != c.want {
			t.Errorf("ResurrectionCount(%s) = %d, want %d", c.card, got, c.want)
		}
	}
}

func TestRecordCapturesDuringCombo(t *testing.T) {
	// White pawn e4 captures black pawn d5; the lost black pawn must land in the pool.
	const fen = "4k3/8/8/3p4/4P3/8/8/4K3 w - - 0 1"
	g := newGameAt(t, fen)
	if err := StartChessCombo(g, drawCard("1"), 1); err != nil {
		t.Fatalf("StartChessCombo: %v", err)
	}
	if _, err := PlaySubMove(g, "e4d5"); err != nil {
		t.Fatalf("PlaySubMove: %v", err)
	}

	got := g.Captured[chess.Black]
	if len(got) != 1 || got[0] != chess.Pawn {
		t.Fatalf("Captured[Black] = %v, want [Pawn]", got)
	}
	if len(g.Captured[chess.White]) != 0 {
		t.Errorf("White lost nothing, but Captured[White] = %v", g.Captured[chess.White])
	}
}

func TestResurrectPlacesPieceAndConsumesPool(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Queen}}

	// e3 is empty and on White's half (rank 3).
	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Queen, Square: sq(chess.FileE, chess.Rank3)},
	})
	if err != nil {
		t.Fatalf("PlayResurrection: %v", err)
	}

	pos := g.ChessEngine.Position()
	if got := pos.Board().Piece(chess.E3); got != chess.WhiteQueen {
		t.Errorf("expected White queen on e3, got %v", got)
	}
	if len(g.Captured[chess.White]) != 0 {
		t.Errorf("captured queen should be consumed, pool = %v", g.Captured[chess.White])
	}
	if pos.Turn() != chess.Black {
		t.Errorf("turn should pass to Black, got %v", pos.Turn())
	}
	if len(g.History) != 1 {
		t.Errorf("expected 1 TurnRecord, got %d", len(g.History))
	}
}

func TestResurrectRejectsOpponentHalf(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Rook}}

	// e5 is rank 5 — Black's half.
	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Rook, Square: sq(chess.FileE, chess.Rank5)},
	})
	if !errors.Is(err, ErrSquareNotOwnHalf) {
		t.Fatalf("expected ErrSquareNotOwnHalf, got %v", err)
	}
	assertUntouched(t, g, chess.Rook)
}

func TestResurrectRejectsOccupiedSquare(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Queen}}

	// e2 holds a White pawn in the start position.
	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Queen, Square: sq(chess.FileE, chess.Rank2)},
	})
	if !errors.Is(err, ErrSquareOccupied) {
		t.Fatalf("expected ErrSquareOccupied, got %v", err)
	}
	assertUntouched(t, g, chess.Queen)
}

func TestResurrectRejectsUncapturedPiece(t *testing.T) {
	g := newGameAt(t, startFEN) // empty captured pool
	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Queen, Square: sq(chess.FileE, chess.Rank3)},
	})
	if !errors.Is(err, ErrPieceNotCaptured) {
		t.Fatalf("expected ErrPieceNotCaptured, got %v", err)
	}
}

func TestResurrectRejectsKing(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.King}}
	err := PlayResurrection(g, drawCard(models.Pl4), []Resurrection{
		{Piece: chess.King, Square: sq(chess.FileE, chess.Rank3)},
	})
	if !errors.Is(err, ErrCannotResurrectKing) {
		t.Fatalf("expected ErrCannotResurrectKing, got %v", err)
	}
}

func TestResurrectRejectsTooMany(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Pawn, chess.Pawn, chess.Pawn}}
	// A +2 allows only 2 resurrections.
	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Pawn, Square: sq(chess.FileA, chess.Rank3)},
		{Piece: chess.Pawn, Square: sq(chess.FileB, chess.Rank3)},
		{Piece: chess.Pawn, Square: sq(chess.FileC, chess.Rank3)},
	})
	if !errors.Is(err, ErrTooManyResurrections) {
		t.Fatalf("expected ErrTooManyResurrections, got %v", err)
	}
}

func TestResurrectIsAllOrNothing(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Queen, chess.Rook}}

	// First placement is fine (e3), second is illegal (Black's half) — the whole
	// request must fail and leave the board and pool untouched.
	err := PlayResurrection(g, drawCard(models.Pl4), []Resurrection{
		{Piece: chess.Queen, Square: sq(chess.FileE, chess.Rank3)},
		{Piece: chess.Rook, Square: sq(chess.FileE, chess.Rank6)},
	})
	if !errors.Is(err, ErrSquareNotOwnHalf) {
		t.Fatalf("expected ErrSquareNotOwnHalf, got %v", err)
	}
	if got := g.ChessEngine.Position().Board().Piece(chess.E3); got != chess.NoPiece {
		t.Errorf("e3 should be empty after a rejected request, got %v", got)
	}
	if len(g.Captured[chess.White]) != 2 {
		t.Errorf("pool should be untouched (2 pieces), got %v", g.Captured[chess.White])
	}
	if len(g.History) != 0 {
		t.Errorf("a rejected resurrection should record no TurnRecord, got %d", len(g.History))
	}
}

func TestResurrectMultiplePieces(t *testing.T) {
	g := newGameAt(t, startFEN)
	g.Captured = map[chess.Color][]chess.PieceType{chess.White: {chess.Knight, chess.Bishop}}

	err := PlayResurrection(g, drawCard(models.Pl2), []Resurrection{
		{Piece: chess.Knight, Square: sq(chess.FileA, chess.Rank3)},
		{Piece: chess.Bishop, Square: sq(chess.FileH, chess.Rank4)},
	})
	if err != nil {
		t.Fatalf("PlayResurrection: %v", err)
	}
	board := g.ChessEngine.Position().Board()
	if board.Piece(chess.A3) != chess.WhiteKnight {
		t.Errorf("expected White knight on a3, got %v", board.Piece(chess.A3))
	}
	if board.Piece(chess.H4) != chess.WhiteBishop {
		t.Errorf("expected White bishop on h4, got %v", board.Piece(chess.H4))
	}
	if len(g.Captured[chess.White]) != 0 {
		t.Errorf("both pieces should be consumed, pool = %v", g.Captured[chess.White])
	}
}

// assertUntouched checks that a rejected single-piece resurrection left the pool
// holding exactly that one piece and recorded no history.
func assertUntouched(t *testing.T, g *models.UnoChessGame, want chess.PieceType) {
	t.Helper()
	pool := g.Captured[chess.White]
	if len(pool) != 1 || pool[0] != want {
		t.Errorf("captured pool changed on a rejected request: %v", pool)
	}
	if len(g.History) != 0 {
		t.Errorf("rejected request recorded a TurnRecord: %d", len(g.History))
	}
}
