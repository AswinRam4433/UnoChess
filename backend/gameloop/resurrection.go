package gameloop

import (
	"errors"
	"fmt"
	"strings"

	"github.com/notnil/chess"

	"unochess/models"
)

// Errors returned by the resurrection mechanic. They are sentinel values so a
// transport layer can map them onto status codes with errors.Is.
var (
	ErrNotResurrectionCard  = errors.New("card does not grant a resurrection")
	ErrTooManyResurrections = errors.New("more pieces requested than the card allows")
	ErrCannotResurrectKing  = errors.New("the king cannot be resurrected")
	ErrSquareNotOwnHalf     = errors.New("resurrected pieces must be placed on your own half of the board")
	ErrSquareOccupied       = errors.New("target square is not empty")
	ErrPieceNotCaptured     = errors.New("requested piece is not in the captured pool")
	ErrInvalidPieceType     = errors.New("invalid piece type for resurrection")
)

// Resurrection names one captured piece to bring back and the empty square — on the
// player's own half — to place it on.
type Resurrection struct {
	Piece  chess.PieceType
	Square chess.Square
}

// resurrectablePiece maps a color and piece type to the concrete board piece. The
// king is deliberately absent: it can never be resurrected, so a king lookup misses.
var resurrectablePiece = map[chess.Color]map[chess.PieceType]chess.Piece{
	chess.White: {
		chess.Queen:  chess.WhiteQueen,
		chess.Rook:   chess.WhiteRook,
		chess.Bishop: chess.WhiteBishop,
		chess.Knight: chess.WhiteKnight,
		chess.Pawn:   chess.WhitePawn,
	},
	chess.Black: {
		chess.Queen:  chess.BlackQueen,
		chess.Rook:   chess.BlackRook,
		chess.Bishop: chess.BlackBishop,
		chess.Knight: chess.BlackKnight,
		chess.Pawn:   chess.BlackPawn,
	},
}

// ResurrectionCount reports how many pieces a card lets its player bring back:
// 2 for a Draw Two (+2), 4 for a Wild Draw Four (+4), and 0 for any other card.
func ResurrectionCount(card models.UnoCard) int {
	switch card.Value {
	case models.Pl2:
		return 2
	case models.Pl4:
		return 4
	default:
		return 0
	}
}

// PlayResurrection applies the +2 / +4 "Resurrection" mechanic (rulebook §3B/§3C) for
// the active color: in place of taking chess moves this turn, it brings the named
// captured pieces back onto empty squares on the player's own half of the board
// (White: ranks 1–4, Black: ranks 5–8).
//
// Every placement is validated against the rules — own half only, empty target, a
// piece actually sitting in the captured pool, never a king, no two placements on the
// same square, and no more pieces than the card allows. The request is all-or-nothing:
// if any placement is invalid, the game is left completely untouched.
//
// On success the pieces are added to the board, removed from the captured pool, the
// turn passes to the opponent, and a TurnRecord is appended. Resurrecting fewer than
// the card's maximum is allowed — when the captured pool or the empty squares run
// short the effect is simply "wasted", exactly as the rulebook notes. Recoloring
// active Uno play on a +4 and advancing the seat remain the Uno turn manager's job,
// as with PlaySubMove.
func PlayResurrection(g *models.UnoChessGame, card models.UnoCard, placements []Resurrection) error {
	if g.ChessEngine == nil {
		return ErrNoChessEngine
	}
	if g.Pending != nil {
		return ErrComboInProgress
	}

	max := ResurrectionCount(card)
	if max == 0 {
		return fmt.Errorf("%w: %s", ErrNotResurrectionCard, card.Value)
	}
	if len(placements) > max {
		return fmt.Errorf("%w: %d requested, %s allows %d", ErrTooManyResurrections, len(placements), card.Value, max)
	}

	color := g.ActiveColor
	board := g.ChessEngine.Position().Board()

	// Validate every placement up front so a single bad one changes nothing. Tally
	// demand per piece type and the squares being filled in this same request.
	demand := map[chess.PieceType]int{}
	filling := map[chess.Square]bool{}
	for _, p := range placements {
		if p.Piece == chess.King {
			return ErrCannotResurrectKing
		}
		if _, ok := resurrectablePiece[color][p.Piece]; !ok {
			return fmt.Errorf("%w: %v", ErrInvalidPieceType, p.Piece)
		}
		if !onOwnHalf(color, p.Square) {
			return fmt.Errorf("%w: %s", ErrSquareNotOwnHalf, p.Square)
		}
		if filling[p.Square] || board.Piece(p.Square) != chess.NoPiece {
			return fmt.Errorf("%w: %s", ErrSquareOccupied, p.Square)
		}
		filling[p.Square] = true
		demand[p.Piece]++
	}

	// The captured pool must hold at least as many of each type as requested.
	available := map[chess.PieceType]int{}
	for _, pt := range g.Captured[color] {
		available[pt]++
	}
	for pt, n := range demand {
		if available[pt] < n {
			return fmt.Errorf("%w: requested %d %v but %d captured", ErrPieceNotCaptured, n, pt, available[pt])
		}
	}

	// All placements are valid — commit. Add the pieces to a copy of the board map.
	squares := board.SquareMap()
	for _, p := range placements {
		squares[p.Square] = resurrectablePiece[color][p.Piece]
	}

	// Reassemble the FEN: new piece placement, opponent to move (the turn passes),
	// and en passant cleared since no pawn just double-stepped.
	fen := g.ChessEngine.Position().String()
	parts := strings.Split(fen, " ")
	if len(parts) < 6 {
		return fmt.Errorf("unexpected FEN %q from engine", fen)
	}
	parts[0] = chess.NewBoard(squares).String()
	parts[1] = colorFENField(color.Other())
	parts[3] = "-"
	newFEN := strings.Join(parts, " ")

	cfg, err := chess.FEN(newFEN)
	if err != nil {
		return fmt.Errorf("resurrection produced invalid FEN %q: %w", newFEN, err)
	}

	// Mutate game state only after every fallible step has succeeded.
	g.ChessEngine = chess.NewGame(cfg)
	if len(placements) > 0 {
		g.Captured[color] = removeFromPool(g.Captured[color], demand)
	}
	g.History = append(g.History, models.TurnRecord{
		Player:      color,
		CardPlayed:  card,
		BoardStates: []string{newFEN},
	})
	g.Phase = models.PhaseTurnComplete
	return nil
}

// onOwnHalf reports whether sq lies on color's half of the board: ranks 1–4 for
// White, ranks 5–8 for Black.
func onOwnHalf(color chess.Color, sq chess.Square) bool {
	r := sq.Rank()
	if color == chess.White {
		return r >= chess.Rank1 && r <= chess.Rank4
	}
	return r >= chess.Rank5 && r <= chess.Rank8
}

// removeFromPool returns pool with `demand` occurrences of each piece type removed.
// Availability is assumed to have been validated by the caller.
func removeFromPool(pool []chess.PieceType, demand map[chess.PieceType]int) []chess.PieceType {
	left := make(map[chess.PieceType]int, len(demand))
	for pt, n := range demand {
		left[pt] = n
	}
	out := make([]chess.PieceType, 0, len(pool))
	for _, pt := range pool {
		if left[pt] > 0 {
			left[pt]--
			continue
		}
		out = append(out, pt)
	}
	return out
}

// recordCaptures appends to the opponent's captured pool any pieces that left the
// board between beforeFEN and afterFEN. On any single move only the side that did NOT
// move can lose a piece, so we diff just that color's material — which conveniently
// ignores the mover's own pawn→queen promotion (a same-color type swap, not a loss).
func recordCaptures(g *models.UnoChessGame, mover chess.Color, beforeFEN, afterFEN string) {
	victim := mover.Other()
	before := materialByType(beforeFEN, victim)
	after := materialByType(afterFEN, victim)

	for pt, n := range before {
		for lost := n - after[pt]; lost > 0; lost-- {
			if g.Captured == nil {
				g.Captured = map[chess.Color][]chess.PieceType{}
			}
			g.Captured[victim] = append(g.Captured[victim], pt)
		}
	}
}

// materialByType counts, by piece type, how many pieces of the given color stand on
// the board described by fen. A malformed FEN yields an empty count.
func materialByType(fen string, color chess.Color) map[chess.PieceType]int {
	cfg, err := chess.FEN(fen)
	if err != nil {
		return nil
	}
	counts := map[chess.PieceType]int{}
	for _, p := range chess.NewGame(cfg).Position().Board().SquareMap() {
		if p.Color() == color {
			counts[p.Type()]++
		}
	}
	return counts
}

// colorFENField returns the side-to-move letter ("w"/"b") for the FEN active-color field.
func colorFENField(color chess.Color) string {
	if color == chess.White {
		return "w"
	}
	return "b"
}
