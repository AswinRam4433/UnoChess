package gameloop

import (
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
