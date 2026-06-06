package gameloop

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/notnil/chess"

	"unochess/models"
)

// pcgRand builds a deterministic *rand.Rand from two named seeds, so test runs are
// independent of system entropy and identical across machines.
func pcgRand(s1, s2 uint64) *rand.Rand {
	return rand.New(rand.NewPCG(s1, s2))
}

// RunGame terminates in a valid terminal state ------------

func TestRunGame_TerminatesWithValidResult(t *testing.T) {
	g := NewUnoChessGameWith(pcgRand(42, 7))
	res, err := RunGame(g, RunOptions{})
	if err != nil {
		t.Fatalf("RunGame: %v", err)
	}

	if g.Phase != models.PhaseGameOver {
		t.Errorf("Phase = %v, want PhaseGameOver", g.Phase)
	}
	if res.Turns <= 0 {
		t.Errorf("Turns = %d, want > 0", res.Turns)
	}
	switch res.Reason {
	case UnoWin, CheckmateWin, Draw, TurnCapHit:
		// valid
	default:
		t.Errorf("Reason = %q, want one of UnoWin/CheckmateWin/Draw/TurnCapHit", res.Reason)
	}

	// Winner ↔ Reason coherence.
	switch res.Reason {
	case UnoWin, CheckmateWin:
		if res.Winner != chess.White && res.Winner != chess.Black {
			t.Errorf("Winner = %v on win Reason=%q, want White or Black", res.Winner, res.Reason)
		}
	case Draw, TurnCapHit:
		if res.Winner != chess.NoColor {
			t.Errorf("Winner = %v on Reason=%q, want NoColor", res.Winner, res.Reason)
		}
	}
}

func TestRunGame_TurnCapEndsAsTurnCapHit(t *testing.T) {
	g := NewUnoChessGameWith(pcgRand(1, 1))
	res, err := RunGame(g, RunOptions{TurnCap: 1})
	if err != nil {
		t.Fatalf("RunGame: %v", err)
	}
	if res.Reason != TurnCapHit {
		t.Errorf("Reason = %q, want TurnCapHit", res.Reason)
	}
	if res.Winner != chess.NoColor {
		t.Errorf("Winner = %v, want NoColor on turn-cap", res.Winner)
	}
	if g.Phase != models.PhaseGameOver {
		t.Errorf("Phase = %v, want PhaseGameOver", g.Phase)
	}
}

// same seed → byte-identical game -------------------------------

func TestRunGame_Deterministic(t *testing.T) {
	g1 := NewUnoChessGameWith(pcgRand(99, 1234))
	r1, err := RunGame(g1, RunOptions{})
	if err != nil {
		t.Fatalf("first RunGame: %v", err)
	}

	g2 := NewUnoChessGameWith(pcgRand(99, 1234))
	r2, err := RunGame(g2, RunOptions{})
	if err != nil {
		t.Fatalf("second RunGame: %v", err)
	}

	if r1 != r2 {
		t.Errorf("GameResult diverged: %+v vs %+v", r1, r2)
	}
	if !reflect.DeepEqual(g1.History, g2.History) {
		t.Errorf("History diverged across runs: len %d vs %d", len(g1.History), len(g2.History))
	}
	if !reflect.DeepEqual(g1.Hands, g2.Hands) {
		t.Error("final Hands diverged across runs")
	}
}

// TestPhase3_BotGameInvariants drives several seeded games to completion
// This is the test that locks the rules engine into CI —
// any regression that violates card conservation, phase coherence, or History
// integrity surfaces here.
func TestPhase3_BotGameInvariants(t *testing.T) {
	seeds := []struct{ a, b uint64 }{
		{42, 7},
		{99, 1234},
		{1, 1},
		{31415, 27182},
	}
	for _, s := range seeds {
		s := s
		t.Run(fmt.Sprintf("seed_%d_%d", s.a, s.b), func(t *testing.T) {
			g := NewUnoChessGameWith(pcgRand(s.a, s.b))
			opts := RunOptions{
				// Tight cap so a runaway test fails fast rather than spinning to
				// the 10k default. Real games end well under this.
				TurnCap: 3000,
				OnTurnEnd: func(g *models.UnoChessGame, turn int) {
					assertPerTurnInvariants(t, g, turn)
				},
			}
			res, err := RunGame(g, opts)
			if err != nil {
				t.Fatalf("RunGame: %v", err)
			}
			assertTerminalInvariants(t, g, res)
		})
	}
}

// assertPerTurnInvariants checks the invariants that must hold at every turn
// boundary (the OnTurnEnd hook fires after AdvanceTurn or game-end).
func assertPerTurnInvariants(t *testing.T, g *models.UnoChessGame, turn int) {
	t.Helper()

	// Card conservation: the 104-card house deck stays whole at all times.
	const fullDeck = 104
	total := len(g.Hands[chess.White]) + len(g.Hands[chess.Black]) + len(g.DrawPile) + len(g.DiscardPile)
	if total != fullDeck {
		t.Errorf("turn %d: card conservation broken — %d on table, want %d", turn, total, fullDeck)
	}

	// Phase is always one of the enum values.
	switch g.Phase {
	case models.PhaseAwaitingCard, models.PhaseInCombo, models.PhaseAwaitingResurrection, models.PhaseTurnComplete, models.PhaseGameOver:
		// OK
	default:
		t.Errorf("turn %d: invalid Phase: %v", turn, g.Phase)
	}

	// At a turn boundary, Phase is either ready for the next player or game over —
	// never mid-turn states.
	if g.Phase != models.PhaseAwaitingCard && g.Phase != models.PhaseGameOver {
		t.Errorf("turn %d: Phase at boundary = %v, want PhaseAwaitingCard or PhaseGameOver", turn, g.Phase)
	}

	// Pending must be nil at a turn boundary — the combo state only lives across
	// PlaySubMove calls within a single turn.
	if g.Pending != nil {
		t.Errorf("turn %d: Pending should be nil at boundary, got %+v", turn, g.Pending)
	}

	// ActiveColor is always a real color.
	if g.ActiveColor != chess.White && g.ActiveColor != chess.Black {
		t.Errorf("turn %d: ActiveColor = %v, want White or Black", turn, g.ActiveColor)
	}

	// Every FEN ever recorded in History must parse cleanly.
	for i, rec := range g.History {
		for j, fen := range rec.BoardStates {
			if _, err := chess.FEN(fen); err != nil {
				t.Errorf("turn %d: History[%d].BoardStates[%d] invalid FEN %q: %v", turn, i, j, fen, err)
			}
		}
	}
}

// assertTerminalInvariants checks the invariants that must hold once
// PhaseGameOver is reached.
func assertTerminalInvariants(t *testing.T, g *models.UnoChessGame, res GameResult) {
	t.Helper()

	if g.Phase != models.PhaseGameOver {
		t.Fatalf("expected PhaseGameOver, got %v", g.Phase)
	}
	if res.Winner != g.Winner {
		t.Errorf("GameResult.Winner=%v but game.Winner=%v", res.Winner, g.Winner)
	}

	switch res.Reason {
	case UnoWin:
		if res.Winner != chess.White && res.Winner != chess.Black {
			t.Errorf("UnoWin: Winner = %v, want White or Black", res.Winner)
		}
		if got := len(g.Hands[res.Winner]); got != 0 {
			t.Errorf("UnoWin: winner's hand has %d cards, want 0", got)
		}
	case CheckmateWin:
		if res.Winner != chess.White && res.Winner != chess.Black {
			t.Errorf("CheckmateWin: Winner = %v, want White or Black", res.Winner)
		}
		oc := g.ChessEngine.Outcome()
		if oc != chess.WhiteWon && oc != chess.BlackWon {
			t.Errorf("CheckmateWin: chess engine outcome = %v, want a chess victory", oc)
		}
	case Draw, TurnCapHit:
		if res.Winner != chess.NoColor {
			t.Errorf("%v: Winner = %v, want NoColor", res.Reason, res.Winner)
		}
	default:
		t.Errorf("unknown Reason: %v", res.Reason)
	}
}

// Different seeds should (almost certainly) produce different games. This guards
// against a regression where everything collapses to a single trace.
func TestRunGame_DifferentSeedsDiverge(t *testing.T) {
	g1 := NewUnoChessGameWith(pcgRand(1, 2))
	r1, err := RunGame(g1, RunOptions{})
	if err != nil {
		t.Fatalf("first RunGame: %v", err)
	}
	g2 := NewUnoChessGameWith(pcgRand(3, 4))
	r2, err := RunGame(g2, RunOptions{})
	if err != nil {
		t.Fatalf("second RunGame: %v", err)
	}
	if r1 == r2 && reflect.DeepEqual(g1.History, g2.History) {
		t.Error("two different seeds produced identical games — RNG is not being threaded through")
	}
}
