package gameloop

import (
	"fmt"

	"github.com/notnil/chess"

	"unochess/core"
)

// ChessTurnResult captures everything that happened during a single UnoChess turn
// of one or more consecutive chess moves by the same player. BoardStates maps
// directly onto models.TurnRecord.BoardStates (one FEN per intermediate state).
type ChessTurnResult struct {
	Moves       []*chess.Move // the sub-moves actually played, in order
	BoardStates []string      // FEN after each sub-move
	FinalFEN    string        // board after the last sub-move (or the start FEN if none)
	Outcome     chess.Outcome // chess.NoOutcome unless a sub-move ended the game
	Method      chess.Method  // how the game ended (e.g. Checkmate), if it did
}

// SubMoveStatus reports what happened when ApplyChessSubMove tried to play one move.
type SubMoveStatus int

const (
	// SubMovePlayed: a legal move was chosen and applied.
	SubMovePlayed SubMoveStatus = iota
	// SubMoveNoLegalMoves: the side to move has no legal move (checkmate/stalemate).
	SubMoveNoLegalMoves
	// SubMoveDeclined: the chooser returned nil. For an in-process bot this means
	// "stop the combo"; for a per-request handler it means the submitted move was
	// not in the legal set — i.e. an illegal submission to reject.
	SubMoveDeclined
)

// ChessSubMoveResult is the outcome of a single sub-move.
type ChessSubMoveResult struct {
	Status  SubMoveStatus
	Move    *chess.Move   // the move played, when Status == SubMovePlayed
	FEN     string        // resulting position when played; the unchanged input FEN otherwise
	Outcome chess.Outcome // chess.NoOutcome unless this move ended the game
	Method  chess.Method
}

// ApplyChessSubMove validates and applies exactly one sub-move for `color`,
// starting from startFEN and using choose to pick among the legal moves. It is the
// single source of truth for one step of the number-card combo, shared by the
// in-process loop (PlayConsecutiveChessMoves) and any per-request handler driving a
// staggered, frontend-supplied turn.
//
// isFirst mirrors the combo's en-passant rule: the first sub-move keeps any en
// passant target the opponent left behind, while every later sub-move forces the
// turn back to `color` and drops the stale target our own pawn may have just
// created. A handler therefore passes isFirst = (movesPlayed == 0).
func ApplyChessSubMove(startFEN string, color chess.Color, isFirst bool, choose ChessMoveChooser) (ChessSubMoveResult, error) {
	posFEN := startFEN
	if isFirst {
		posFEN = core.ForceTurnInFEN(posFEN, color)
	} else {
		posFEN = forceColorClearEnPassant(posFEN, color)
	}

	cfg, err := chess.FEN(posFEN)
	if err != nil {
		return ChessSubMoveResult{}, fmt.Errorf("invalid FEN %q: %w", posFEN, err)
	}
	engine := chess.NewGame(cfg)

	legal := engine.ValidMoves()
	if len(legal) == 0 {
		return ChessSubMoveResult{Status: SubMoveNoLegalMoves, FEN: startFEN, Outcome: chess.NoOutcome}, nil
	}

	move := choose(legal)
	if move == nil {
		return ChessSubMoveResult{Status: SubMoveDeclined, FEN: startFEN, Outcome: chess.NoOutcome}, nil
	}
	// King capture is the UnoChess winning condition. During a multi-move combo,
	// forceColorClearEnPassant keeps the same player's turn active after delivering
	// check, so the engine includes "capture the king" in ValidMoves. Detect it
	// before engine.Move so we always declare the correct outcome regardless of
	// what outcome the engine reports for a king-less position.
	targetPiece := engine.Position().Board().Piece(move.S2())
	kingCaptured := targetPiece != chess.NoPiece && targetPiece.Type() == chess.King

	if err := engine.Move(move); err != nil {
		return ChessSubMoveResult{}, fmt.Errorf("move %s: %w", move.String(), err)
	}

	// engine.Outcome() reports chess.NoOutcome ("*") for a game still in progress —
	// always carry it through, since the zero value of chess.Outcome is "" and would
	// be misread by callers as "the game ended".
	outcome := engine.Outcome()

	if kingCaptured {
		// King capture wins immediately regardless of moves remaining.
		if color == chess.White {
			outcome = chess.WhiteWon
		} else {
			outcome = chess.BlackWon
		}
		return ChessSubMoveResult{
			Status:  SubMovePlayed,
			Move:    move,
			FEN:     engine.Position().String(),
			Outcome: outcome,
			Method:  chess.Checkmate,
		}, nil
	}

	return ChessSubMoveResult{
		Status:  SubMovePlayed,
		Move:    move,
		FEN:     engine.Position().String(),
		Outcome: outcome,
		Method:  engine.Method(),
	}, nil
}

// PlayConsecutiveChessMoves executes up to `count` back-to-back chess moves for a
// single color, starting from startFEN. This is the heart of the UnoChess
// number-card mechanic: one card = N moves by the same player, in succession.
//
// A normal chess engine hands the turn to the opponent after every move, so
// between sub-moves we rewrite the FEN to force the side-to-move back to `color`.
// The combo ends early if the player runs out of legal moves, the chooser declines
// (returns nil), or a sub-move ends the game — the "checkmate intercept" from the
// rulebook, where delivering mate mid-combo wins immediately without playing out
// the remaining moves.
func PlayConsecutiveChessMoves(startFEN string, color chess.Color, count int, choose ChessMoveChooser) (ChessTurnResult, error) {
	result := ChessTurnResult{FinalFEN: startFEN, Outcome: chess.NoOutcome}

	for i := 0; i < count; i++ {
		sub, err := ApplyChessSubMove(result.FinalFEN, color, i == 0, choose)
		if err != nil {
			return result, fmt.Errorf("sub-move %d: %w", i+1, err)
		}
		if sub.Status != SubMovePlayed {
			break // no legal move, or the chooser declined — the combo ends here
		}

		result.Moves = append(result.Moves, sub.Move)
		result.FinalFEN = sub.FEN
		result.BoardStates = append(result.BoardStates, sub.FEN)

		// Checkmate intercept: a sub-move that ends the game stops the combo at once.
		if sub.Outcome != chess.NoOutcome {
			result.Outcome = sub.Outcome
			result.Method = sub.Method
			break
		}
	}

	return result, nil
}
