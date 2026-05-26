package gameloop

import (
	"errors"
	"fmt"

	"github.com/notnil/chess"

	"unochess/models"
)

// Errors returned by the combo-turn orchestration.
var (
	ErrNoActiveCombo   = errors.New("no active combo: play a number card to start one")
	ErrComboInProgress = errors.New("a combo is already in progress")
	ErrNoChessEngine   = errors.New("game has no chess engine")
	ErrIllegalSubMove  = errors.New("submitted move is not legal in the current position")
)

// SubMoveOutcome reports the post-state after PlaySubMove applies one move, giving a
// transport layer everything it needs to push an update to clients without reaching
// back into the game struct.
type SubMoveOutcome struct {
	Move        *chess.Move   // the move just played, or nil if the combo ended without one
	FEN         string        // resulting board position
	MovesLeft   int           // sub-moves still owed in this combo (0 once ComboDone)
	ComboDone   bool          // true when the combo has ended and the turn should pass
	GameOutcome chess.Outcome // chess.NoOutcome unless this move ended the game
	GameMethod  chess.Method
}

// StartChessCombo begins a number-card turn for the active color: a run of `count`
// consecutive chess moves from the current board position.
func StartChessCombo(g *models.UnoChessGame, card models.UnoCard, count int) error {
	if g.Pending != nil {
		return ErrComboInProgress
	}
	if g.ChessEngine == nil {
		return ErrNoChessEngine
	}
	if count < 1 {
		return fmt.Errorf("combo move count must be >= 1, got %d", count)
	}

	g.Pending = &models.ActiveCombo{
		Color:          g.ActiveColor,
		Card:           card, // Uno Card that was played by the user
		WorkingFEN:     g.ChessEngine.Position().String(),
		MovesRemaining: count,
	}
	return nil
}

// PlaySubMove validates and applies one sub-move — a UCI string supplied by the
// frontend — against the active combo. The move is never trusted: ApplyChessSubMove
// regenerates the legal moves and rejects anything not among them.
//
// On a legal move it advances the combo in place. When the combo ends — the owed
// moves run out, the player has no legal continuation, or a move ends the game
//
// Advancing the Uno seat/turn (Skip, Reverse, direction, dealing)
// is deliberately left to the Uno turn manager; this function is only for the chess portion
func PlaySubMove(g *models.UnoChessGame, uci string) (SubMoveOutcome, error) {
	combo := g.Pending
	if combo == nil {
		return SubMoveOutcome{}, ErrNoActiveCombo
	}

	res, err := ApplyChessSubMove(combo.WorkingFEN, combo.Color, combo.MovesPlayed == 0, MoveByUCI(uci))
	if err != nil {
		return SubMoveOutcome{}, err
	}

	switch res.Status {
	case SubMoveDeclined:
		// The UCI matched no legal move — reject it without mutating any state.
		return SubMoveOutcome{}, fmt.Errorf("%w: %q", ErrIllegalSubMove, uci)
	case SubMoveNoLegalMoves:
		// The player has no legal continuation (stalemate/checkmate against them);
		// the combo ends here with no move played.
		commitCombo(g, combo)
		return SubMoveOutcome{ComboDone: true, FEN: combo.WorkingFEN}, nil
	}

	// res.Status == SubMovePlayed. Record the sub-move.
	// Bank any opponent piece this move captured before WorkingFEN moves on, so it
	// becomes available for a later +2 / +4 resurrection.
	recordCaptures(g, combo.Color, combo.WorkingFEN, res.FEN)
	combo.WorkingFEN = res.FEN
	combo.BoardStates = append(combo.BoardStates, res.FEN)
	combo.MovesPlayed++
	combo.MovesRemaining--

	out := SubMoveOutcome{
		Move:        res.Move,
		FEN:         res.FEN,
		MovesLeft:   combo.MovesRemaining,
		GameOutcome: res.Outcome,
		GameMethod:  res.Method,
	}

	// A game-ending move (checkmate intercept) or the owed moves running out both
	// finish the combo.
	if res.Outcome != chess.NoOutcome || combo.MovesRemaining == 0 {
		commitCombo(g, combo)
		out.ComboDone = true
	}
	return out, nil
}

// commitCombo finalizes a finished combo: it appends a TurnRecord capturing the card
// and every intermediate board, re-syncs the authoritative chess engine to the final
// position, and clears g.Pending.
//
// Each sub-move's resulting FEN already hands the turn to the opponent, so the rebuilt
// engine resumes normal alternating play. Rebuilding from FEN does drop the engine's
// move history (PGN, threefold/50-move counters) — an accepted consequence of the
// FEN-rewriting combo mechanic; the per-turn board history lives in g.History instead.
func commitCombo(g *models.UnoChessGame, combo *models.ActiveCombo) {
	g.History = append(g.History, models.TurnRecord{
		Player:      combo.Color,
		CardPlayed:  combo.Card,
		BoardStates: combo.BoardStates,
	})

	if fen, err := chess.FEN(combo.WorkingFEN); err == nil {
		g.ChessEngine = chess.NewGame(fen)
	}

	g.Pending = nil
}
