package gameloop

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/notnil/chess"

	"unochess/core"
	"unochess/models"
)

// GameWinningReason classifies how a finished UnoChess game ended.
type GameWinningReason string

const (
	UnoWin       GameWinningReason = "Uno"
	CheckmateWin GameWinningReason = "Checkmate"
	Draw         GameWinningReason = "Draw"
	// TurnCapHit signals that the game ran past RunOptions.TurnCap without anyone
	// winning. Winner is chess.NoColor; treat it as a draw — but the distinct label
	// lets callers tell a genuine no-progress draw apart from a safety-valve trip.
	TurnCapHit GameWinningReason = "TurnCap"
)

// GameResult describes the terminal state of a finished game.
type GameResult struct {
	Winner chess.Color       // chess.NoColor on a draw
	Reason GameWinningReason // why the game ended
	Turns  int               // number of turns that actually executed before the game ended
}

// RunOptions configures a RunGame invocation.
type RunOptions struct {
	// ChessMoveChooser is the chess chooser used during number-card combos. When nil
	// the driver falls back to PreferCapturesAndChecks.
	ChessMoveChooser ChessMoveChooser

	// TurnCap is the safety limit on the number of turns played. Hitting it ends the
	// game with Reason=TurnCapHit. Zero means use defaultTurnCap (10,000).
	TurnCap int

	// RandomSourcer is reserved for future bot-level randomness (e.g. a stochastic
	// chess chooser). Currently unused: determinism flows in via NewUnoChessGameWith
	// at construction, since the deck shuffle/deal are the only random ops and they
	// happen before RunGame is called.
	RandomSourcer rand.Source

	// OnTurnEnd, when non-nil, is invoked after every turn that actually executed
	// (including the final, game-ending one) with the game state and the 1-based
	// turn index. It is the seam the integration test uses to assert per-turn
	// invariants without polluting the main loop with assertion logic. The hook is
	// not called on the turn-cap exit path, since that turn never executed.
	OnTurnEnd func(g *models.UnoChessGame, turn int)
}

const defaultTurnCap = 10_000

// RunGame plays the game to a terminal state by repeatedly calling the Phase-2
// handlers (PlayCard, PlaySubMove, PlayResurrection, DrawForTurn, AdvanceTurn). It
// is pure orchestration — no new mutation logic lives here — so the integrated
// state machine is exercised exactly the way a transport layer would exercise it.
//
// Termination is guaranteed by three mechanisms, in order:
//   - PhaseGameOver set by a Uno-out (PlayCard) or checkmate intercept (PlaySubMove).
//   - A full round with no progress (no card played and none drawn) — a draw.
//   - TurnCap as the last-resort safety valve, also reported as a draw.
func RunGame(g *models.UnoChessGame, opts RunOptions) (GameResult, error) {
	if g == nil {
		return GameResult{}, errors.New("game is nil")
	}
	if g.ChessEngine == nil {
		return GameResult{}, ErrNoChessEngine
	}

	chooser := opts.ChessMoveChooser
	if chooser == nil {
		chooser = PreferCapturesAndChecks
	}
	turnCap := opts.TurnCap
	if turnCap <= 0 {
		turnCap = defaultTurnCap
	}

	// A wild opening discard has no color — resolve it before turn 1 so all
	// matching downstream is unambiguous.
	if topOfDiscard(g).Color == models.Wild {
		chosen := ChooseWildColor(g.Hands[g.ActiveColor])
		if err := DeclareStartingColor(g, chosen); err != nil {
			return GameResult{}, fmt.Errorf("declaring opening color: %w", err)
		}
	}

	turn := 0
	stalled := 0
	for g.Phase != models.PhaseGameOver {
		turn++
		if turn > turnCap {
			g.Phase = models.PhaseGameOver
			g.Winner = chess.NoColor
			return GameResult{Winner: chess.NoColor, Reason: TurnCapHit, Turns: turn - 1}, nil
		}
		progressed, err := runOneTurn(g, chooser)
		if err != nil {
			return GameResult{}, fmt.Errorf("turn %d: %w", turn, err)
		}
		if opts.OnTurnEnd != nil {
			opts.OnTurnEnd(g, turn)
		}
		if progressed {
			stalled = 0
		} else {
			stalled++
			// A full two-player round with zero progress means the deck is exhausted
			// and neither player can move — call it a draw rather than spin to TurnCap.
			if stalled >= 2 {
				g.Phase = models.PhaseGameOver
				g.Winner = chess.NoColor
				return GameResult{Winner: chess.NoColor, Reason: Draw, Turns: turn}, nil
			}
		}
	}

	return GameResult{
		Winner: g.Winner,
		Reason: classifyEnd(g),
		Turns:  turn,
	}, nil
}

// runOneTurn plays exactly one turn for the active player and returns whether the
// player made any progress (played or drew a card) so the outer loop can detect a
// dead-round draw.
func runOneTurn(g *models.UnoChessGame, chooser ChessMoveChooser) (bool, error) {
	active := g.ActiveColor
	top := topOfDiscard(g)
	hand := g.Hands[active]

	valid := core.GetValidUnoMoves(top, []models.UnoCard(hand))

	var toPlay *models.UnoCard
	progressed := false

	if len(valid) > 0 {
		c := ChooseMove(valid)
		toPlay = &c
		progressed = true
	} else {
		drew, err := DrawForTurn(g)
		if err != nil {
			return false, fmt.Errorf("draw: %w", err)
		}
		if drew.Drew {
			progressed = true
			if drew.Playable {
				toPlay = &drew.Card
			}
		}
	}

	if toPlay != nil {
		declared := models.CardColor("")
		if isWildCard(*toPlay) {
			declared = ChooseWildColor(g.Hands[active])
		}
		res, err := PlayCard(g, *toPlay, declared)
		if err != nil {
			return progressed, fmt.Errorf("play %s %s: %w", toPlay.Color, toPlay.Value, err)
		}
		if res.UnoWin {
			return true, nil
		}

		switch g.Phase {
		case models.PhaseInCombo:
			if err := runCombo(g, chooser); err != nil {
				return progressed, err
			}
		case models.PhaseAwaitingResurrection:
			if err := runResurrection(g, res.Card); err != nil {
				return progressed, err
			}
		}
	}

	if g.Phase == models.PhaseGameOver {
		return progressed, nil
	}
	return progressed, AdvanceTurn(g)
}

// runCombo plays every sub-move the active player owes, using chooser to pick from
// the legal set PlaySubMove will accept. The loop exits when PlaySubMove flips Phase
// out of PhaseInCombo (either via completion, a stalemate against the active color,
// or the checkmate intercept).
func runCombo(g *models.UnoChessGame, chooser ChessMoveChooser) error {
	for g.Phase == models.PhaseInCombo {
		moves := legalMovesForCombo(g.Pending)
		if len(moves) == 0 {
			// No legal continuation — ApplyChessSubMove reports SubMoveNoLegalMoves
			// regardless of the chooser, and PlaySubMove ends the combo cleanly.
			if _, err := PlaySubMove(g, ""); err != nil {
				return fmt.Errorf("ending stalled combo: %w", err)
			}
			return nil
		}
		move := chooser(moves)
		if move == nil {
			return fmt.Errorf("chooser declined with %d legal moves available", len(moves))
		}
		if _, err := PlaySubMove(g, move.String()); err != nil {
			return fmt.Errorf("sub-move %s: %w", move.String(), err)
		}
	}
	return nil
}

// runResurrection places up to the card's allowance of captured pieces back on the
// active player's own half. The bot uses a deliberately dumb heuristic — pop pieces
// in pool order onto the first empty squares — because Phase 3 is about proving the
// plumbing, not optimizing piece placement.
func runResurrection(g *models.UnoChessGame, card models.UnoCard) error {
	placements := chooseResurrections(g, ResurrectionCount(card))
	return PlayResurrection(g, card, placements)
}

func chooseResurrections(g *models.UnoChessGame, n int) []Resurrection {
	color := g.ActiveColor
	pool := g.Captured[color]
	if len(pool) == 0 || n == 0 {
		return nil
	}
	board := g.ChessEngine.Position().Board()
	empties := emptySquaresOnHalf(board, color)

	placements := make([]Resurrection, 0, n)
	for i, piece := range pool {
		if i >= n || i >= len(empties) {
			break
		}
		placements = append(placements, Resurrection{Piece: piece, Square: empties[i]})
	}
	return placements
}

func emptySquaresOnHalf(board *chess.Board, color chess.Color) []chess.Square {
	var ranks []chess.Rank
	if color == chess.White {
		ranks = []chess.Rank{chess.Rank1, chess.Rank2, chess.Rank3, chess.Rank4}
	} else {
		ranks = []chess.Rank{chess.Rank5, chess.Rank6, chess.Rank7, chess.Rank8}
	}
	files := []chess.File{chess.FileA, chess.FileB, chess.FileC, chess.FileD, chess.FileE, chess.FileF, chess.FileG, chess.FileH}

	var out []chess.Square
	for _, r := range ranks {
		for _, f := range files {
			sq := chess.NewSquare(f, r)
			if board.Piece(sq) == chess.NoPiece {
				out = append(out, sq)
			}
		}
	}
	return out
}

// legalMovesForCombo mirrors ApplyChessSubMove's FEN preprocessing so the bot
// chooser sees exactly the move set PlaySubMove will validate against — including
// the en-passant clearing on second-and-later sub-moves that core's standalone
// GetValidChessMovesForSubMove does not perform.
func legalMovesForCombo(combo *models.ActiveCombo) []*chess.Move {
	posFEN := combo.WorkingFEN
	if combo.MovesPlayed == 0 {
		posFEN = core.ForceTurnInFEN(posFEN, combo.Color)
	} else {
		posFEN = forceColorClearEnPassant(posFEN, combo.Color)
	}
	cfg, err := chess.FEN(posFEN)
	if err != nil {
		return nil
	}
	return chess.NewGame(cfg).ValidMoves()
}

// RunBotTurn drives exactly one complete turn for the active player using chooser
// for chess sub-moves. It is the transport layer's entry point for server-side
// bot opponents. The progressed bool mirrors runOneTurn's semantics: false means
// neither a card was played nor drawn (both piles exhausted).
func RunBotTurn(g *models.UnoChessGame, chooser ChessMoveChooser) (bool, error) {
	return runOneTurn(g, chooser)
}

// ClassifyEnd reports why a PhaseGameOver state was reached. Exported so the
// transport layer can format game-over messages without re-running the game loop.
func ClassifyEnd(g *models.UnoChessGame) GameWinningReason {
	return classifyEnd(g)
}

// classifyEnd reports why a PhaseGameOver state was reached. The chess engine is
// authoritative for chess-track wins; everything else (with a Winner set) is a Uno
// victory; otherwise the game was drawn.
func classifyEnd(g *models.UnoChessGame) GameWinningReason {
	if g.ChessEngine != nil {
		switch g.ChessEngine.Outcome() {
		case chess.WhiteWon, chess.BlackWon:
			return CheckmateWin
		case chess.Draw:
			return Draw
		}
	}
	if g.Winner == chess.NoColor {
		return Draw
	}
	return UnoWin
}
