package gameloop

import (
	"errors"
	"testing"

	"github.com/notnil/chess"

	"unochess/models"
)

// newTurnTestGame builds a deterministic UnoChessGame for the turn-orchestrator
// tests: explicit hands and discard top, an empty draw pile (override in the few
// tests that need it), White to move, ready to accept a card.
func newTurnTestGame(t *testing.T, top models.UnoCard, whiteHand, blackHand models.Deck) *models.UnoChessGame {
	t.Helper()
	return &models.UnoChessGame{
		ChessEngine: chess.NewGame(),
		History:     []models.TurnRecord{},
		Hands: map[chess.Color]models.Deck{
			chess.White: whiteHand,
			chess.Black: blackHand,
		},
		DrawPile:      models.Deck{},
		DiscardPile:   models.Deck{top},
		ActiveColor:   chess.White,
		PlayDirection: 1,
		Phase:         models.PhaseAwaitingCard,
		Winner:        chess.NoColor,
		Captured:      map[chess.Color][]chess.PieceType{},
	}
}

// --- PlayCard per card type ------------------------------------------------

func TestPlayCard_NumberStartsCombo(t *testing.T) {
	play := card(models.Red, "3")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Yellow, "5")},
	)

	res, err := PlayCard(g, play, "")
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseInCombo {
		t.Errorf("Phase = %v, want PhaseInCombo", g.Phase)
	}
	if res.ComboMoves != 3 {
		t.Errorf("ComboMoves = %d, want 3", res.ComboMoves)
	}
	if g.Pending == nil || g.Pending.MovesRemaining != 3 {
		t.Errorf("expected Pending with 3 moves remaining, got %+v", g.Pending)
	}
	if topOfDiscard(g).Value != "3" {
		t.Errorf("discard top = %v, want the played 3", topOfDiscard(g))
	}
}

func TestPlayCard_PlusTwoStartsResurrection(t *testing.T) {
	play := card(models.Red, models.Pl2)
	g := newTurnTestGame(t,
		card(models.Red, "7"),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Green, "5")},
	)

	res, err := PlayCard(g, play, "")
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseAwaitingResurrection {
		t.Errorf("Phase = %v, want PhaseAwaitingResurrection", g.Phase)
	}
	if res.Resurrections != 2 {
		t.Errorf("Resurrections = %d, want 2", res.Resurrections)
	}
	if res.Recolored {
		t.Error("+2 should not recolor the active play")
	}
}

func TestPlayCard_PlusFourRecolorsAndStartsResurrection(t *testing.T) {
	wild := card(models.Wild, models.Pl4)
	// No red card in hand, so +4 is legal (the last-resort rule is satisfied).
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{wild, card(models.Blue, "9")},
		models.Deck{card(models.Green, "3")},
	)

	res, err := PlayCard(g, wild, models.Green)
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseAwaitingResurrection {
		t.Errorf("Phase = %v, want PhaseAwaitingResurrection", g.Phase)
	}
	if res.Resurrections != 4 {
		t.Errorf("Resurrections = %d, want 4", res.Resurrections)
	}
	if !res.Recolored {
		t.Error("+4 must recolor the active play")
	}
	if topOfDiscard(g).Color != models.Green {
		t.Errorf("discard top color = %v, want Green", topOfDiscard(g).Color)
	}
	if res.Card.Color != models.Green {
		t.Errorf("result.Card.Color = %v, want Green", res.Card.Color)
	}
}

func TestPlayCard_SkipSetsPendingSkipAndCompletes(t *testing.T) {
	play := card(models.Red, models.Skip)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Green, "3")},
	)

	res, err := PlayCard(g, play, "")
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase = %v, want PhaseTurnComplete", g.Phase)
	}
	if !g.PendingSkip {
		t.Error("PendingSkip should be true after a Skip")
	}
	if !res.SkipsOpponent {
		t.Error("SkipsOpponent should be true")
	}
	if len(g.History) != 1 || g.History[0].CardPlayed != play {
		t.Errorf("expected one TurnRecord for the skip, got %+v", g.History)
	}
}

func TestPlayCard_ReverseSetsPendingSkipAndCompletes(t *testing.T) {
	play := card(models.Red, models.Rev)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Green, "3")},
	)

	if _, err := PlayCard(g, play, ""); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if !g.PendingSkip {
		t.Error("Reverse in a 2-player game should set PendingSkip")
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase = %v, want PhaseTurnComplete", g.Phase)
	}
}

func TestPlayCard_WildRecolorsAndCompletes(t *testing.T) {
	wild := card(models.Wild, models.WildCard)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{wild, card(models.Blue, "9")},
		models.Deck{card(models.Green, "3")},
	)

	res, err := PlayCard(g, wild, models.Yellow)
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase = %v, want PhaseTurnComplete", g.Phase)
	}
	if !res.Recolored || topOfDiscard(g).Color != models.Yellow {
		t.Errorf("expected recolor to Yellow, top = %v", topOfDiscard(g))
	}
}

// --- Uno win short-circuits the chess effect (rulebook §4.2) ---------------

func TestPlayCard_UnoWinEndsGameImmediately(t *testing.T) {
	// White has exactly one card: a number card. Playing it would normally start a
	// 3-move combo, but emptying the hand wins instantly with no chess action.
	play := card(models.Red, "3")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{play},
		models.Deck{card(models.Green, "5")},
	)

	res, err := PlayCard(g, play, "")
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if !res.UnoWin {
		t.Error("UnoWin should be true")
	}
	if g.Phase != models.PhaseGameOver {
		t.Errorf("Phase = %v, want PhaseGameOver", g.Phase)
	}
	if g.Winner != chess.White {
		t.Errorf("Winner = %v, want White", g.Winner)
	}
	if g.Pending != nil {
		t.Error("no combo should have been started")
	}
}

// --- PlayCard error branches ----------------------------------------------

func TestPlayCard_RejectsIllegalCard(t *testing.T) {
	play := card(models.Blue, "5")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{play, card(models.Green, "9")},
		models.Deck{card(models.Green, "3")},
	)

	_, err := PlayCard(g, play, "")
	if !errors.Is(err, ErrIllegalCardPlay) {
		t.Fatalf("expected ErrIllegalCardPlay, got %v", err)
	}
	if len(g.Hands[chess.White]) != 2 {
		t.Errorf("hand should be untouched, got %d cards", len(g.Hands[chess.White]))
	}
	if len(g.DiscardPile) != 1 {
		t.Errorf("discard pile should be untouched, got %d", len(g.DiscardPile))
	}
}

func TestPlayCard_RejectsCardNotInHand(t *testing.T) {
	play := card(models.Red, "5")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{card(models.Red, "9")}, // hand does NOT contain the 5
		models.Deck{card(models.Green, "3")},
	)

	_, err := PlayCard(g, play, "")
	if !errors.Is(err, ErrCardNotInHand) {
		t.Fatalf("expected ErrCardNotInHand, got %v", err)
	}
}

func TestPlayCard_RejectsWildWithoutDeclaredColor(t *testing.T) {
	wild := card(models.Wild, models.Pl4)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{wild, card(models.Blue, "9")},
		models.Deck{card(models.Green, "3")},
	)

	_, err := PlayCard(g, wild, models.Wild) // Wild is not a real declared color
	if !errors.Is(err, ErrInvalidWildColor) {
		t.Fatalf("expected ErrInvalidWildColor, got %v", err)
	}
}

func TestPlayCard_RejectsPlusFourWhenColorMatchExists(t *testing.T) {
	// Rulebook +4 last-resort: holding a Red card disallows the +4 on a Red top.
	wild := card(models.Wild, models.Pl4)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{wild, card(models.Red, "9")},
		models.Deck{card(models.Green, "3")},
	)

	_, err := PlayCard(g, wild, models.Green)
	if !errors.Is(err, ErrIllegalCardPlay) {
		t.Fatalf("expected ErrIllegalCardPlay (+4 last-resort), got %v", err)
	}
}

func TestPlayCard_RejectsOutsideAwaitingCardPhase(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{card(models.Red, "3"), card(models.Blue, "9")},
		models.Deck{card(models.Green, "5")},
	)
	g.Phase = models.PhaseInCombo

	_, err := PlayCard(g, card(models.Red, "3"), "")
	if !errors.Is(err, ErrNotAwaitingCard) {
		t.Fatalf("expected ErrNotAwaitingCard, got %v", err)
	}
}

func TestPlayCard_RejectsAfterGameOver(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{card(models.Red, "3")},
		models.Deck{card(models.Green, "5")},
	)
	g.Phase = models.PhaseGameOver

	_, err := PlayCard(g, card(models.Red, "3"), "")
	if !errors.Is(err, ErrGameOver) {
		t.Fatalf("expected ErrGameOver, got %v", err)
	}
}

// --- DrawForTurn ----------------------------------------------------------

func TestDrawForTurn_RejectsWithPlayableCard(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{card(models.Red, "9")}, // playable by color
		models.Deck{card(models.Green, "3")},
	)
	g.DrawPile = models.Deck{card(models.Blue, "8")}

	_, err := DrawForTurn(g)
	if !errors.Is(err, ErrHasPlayableCard) {
		t.Fatalf("expected ErrHasPlayableCard, got %v", err)
	}
	if len(g.Hands[chess.White]) != 1 {
		t.Errorf("hand should be untouched, got %d", len(g.Hands[chess.White]))
	}
}

func TestDrawForTurn_PlayableDrawnKeepsPhase(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{card(models.Blue, "8")}, // unplayable on Red 5
		models.Deck{card(models.Green, "3")},
	)
	g.DrawPile = models.Deck{card(models.Red, "9")} // drawn card IS playable

	res, err := DrawForTurn(g)
	if err != nil {
		t.Fatalf("DrawForTurn: %v", err)
	}
	if !res.Drew || !res.Playable {
		t.Errorf("expected drew=true playable=true, got %+v", res)
	}
	if g.Phase != models.PhaseAwaitingCard {
		t.Errorf("Phase = %v, want PhaseAwaitingCard (player may now PlayCard)", g.Phase)
	}
	if len(g.Hands[chess.White]) != 2 {
		t.Errorf("hand should now have 2 cards, got %d", len(g.Hands[chess.White]))
	}
}

func TestDrawForTurn_UnplayableDrawnEndsTurn(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{card(models.Blue, "8")},
		models.Deck{card(models.Green, "3")},
	)
	g.DrawPile = models.Deck{card(models.Green, "1")} // unplayable on Red 5

	res, err := DrawForTurn(g)
	if err != nil {
		t.Fatalf("DrawForTurn: %v", err)
	}
	if !res.Drew || res.Playable {
		t.Errorf("expected drew=true playable=false, got %+v", res)
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase = %v, want PhaseTurnComplete", g.Phase)
	}
}

func TestDrawForTurn_ExhaustedPilesEndTurn(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{card(models.Blue, "8")},
		models.Deck{card(models.Green, "3")},
	)
	// Draw pile empty; discard has only the top → reshuffle yields nothing.

	res, err := DrawForTurn(g)
	if err != nil {
		t.Fatalf("DrawForTurn: %v", err)
	}
	if res.Drew {
		t.Error("no card should have been drawn from exhausted piles")
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase = %v, want PhaseTurnComplete", g.Phase)
	}
}

// --- AdvanceTurn ----------------------------------------------------------

func TestAdvanceTurn_FlipsActiveColor(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{}, models.Deck{},
	)
	g.Phase = models.PhaseTurnComplete

	if err := AdvanceTurn(g); err != nil {
		t.Fatalf("AdvanceTurn: %v", err)
	}
	if g.ActiveColor != chess.Black {
		t.Errorf("ActiveColor = %v, want Black", g.ActiveColor)
	}
	if g.Phase != models.PhaseAwaitingCard {
		t.Errorf("Phase = %v, want PhaseAwaitingCard", g.Phase)
	}
}

func TestAdvanceTurn_HonorsPendingSkip(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{}, models.Deck{},
	)
	g.Phase = models.PhaseTurnComplete
	g.PendingSkip = true

	if err := AdvanceTurn(g); err != nil {
		t.Fatalf("AdvanceTurn: %v", err)
	}
	if g.ActiveColor != chess.White {
		t.Errorf("ActiveColor = %v, want White (skipped opponent)", g.ActiveColor)
	}
	if g.PendingSkip {
		t.Error("PendingSkip should be consumed")
	}
}

func TestAdvanceTurn_RejectsBeforeTurnComplete(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{}, models.Deck{},
	)
	// Phase stays PhaseAwaitingCard.

	if err := AdvanceTurn(g); !errors.Is(err, ErrTurnNotComplete) {
		t.Fatalf("expected ErrTurnNotComplete, got %v", err)
	}
}

func TestAdvanceTurn_GameOverIsNoop(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{}, models.Deck{},
	)
	g.Phase = models.PhaseGameOver
	g.Winner = chess.White

	if err := AdvanceTurn(g); err != nil {
		t.Fatalf("expected no error from GameOver advance, got %v", err)
	}
	if g.Phase != models.PhaseGameOver || g.Winner != chess.White {
		t.Errorf("GameOver state should be untouched, got phase=%v winner=%v", g.Phase, g.Winner)
	}
}

// --- End-to-end flows -----------------------------------------------------

func TestPhase2_NumberComboFlowToTurnComplete(t *testing.T) {
	// PlayCard(number) → PlaySubMove × N → Phase = PhaseTurnComplete, then AdvanceTurn.
	play := card(models.Red, "2")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Green, "5")},
	)

	if _, err := PlayCard(g, play, ""); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseInCombo {
		t.Fatalf("Phase after PlayCard = %v, want PhaseInCombo", g.Phase)
	}

	if _, err := PlaySubMove(g, "b1c3"); err != nil {
		t.Fatalf("first PlaySubMove: %v", err)
	}
	if g.Phase != models.PhaseInCombo {
		t.Errorf("Phase mid-combo = %v, want PhaseInCombo", g.Phase)
	}

	out, err := PlaySubMove(g, "c3d5")
	if err != nil {
		t.Fatalf("second PlaySubMove: %v", err)
	}
	if !out.ComboDone {
		t.Error("ComboDone should be true on the final sub-move")
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase after combo = %v, want PhaseTurnComplete", g.Phase)
	}

	if err := AdvanceTurn(g); err != nil {
		t.Fatalf("AdvanceTurn: %v", err)
	}
	if g.ActiveColor != chess.Black {
		t.Errorf("ActiveColor = %v, want Black", g.ActiveColor)
	}
	if g.Phase != models.PhaseAwaitingCard {
		t.Errorf("Phase after AdvanceTurn = %v, want PhaseAwaitingCard", g.Phase)
	}
}

func TestPhase2_CheckmateInterceptSetsWinner(t *testing.T) {
	// White has a 3-move combo and a mate-in-1; the mating sub-move ends the game
	// instantly with Winner=White, even though MovesRemaining > 0.
	const mateIn1 = "6k1/5ppp/8/8/8/8/8/R6K w - - 0 1"
	play := card(models.Red, "3")
	g := newTurnTestGame(t,
		card(models.Red, models.Skip),
		models.Deck{play, card(models.Blue, "9")},
		models.Deck{card(models.Green, "5")},
	)
	cfg, err := chess.FEN(mateIn1)
	if err != nil {
		t.Fatalf("FEN: %v", err)
	}
	g.ChessEngine = chess.NewGame(cfg)

	if _, err := PlayCard(g, play, ""); err != nil {
		t.Fatalf("PlayCard: %v", err)
	}

	out, err := PlaySubMove(g, "a1a8")
	if err != nil {
		t.Fatalf("PlaySubMove: %v", err)
	}
	if out.GameOutcome != chess.WhiteWon || out.GameMethod != chess.Checkmate {
		t.Errorf("expected WhiteWon/Checkmate, got %v/%v", out.GameOutcome, out.GameMethod)
	}
	if g.Phase != models.PhaseGameOver {
		t.Errorf("Phase = %v, want PhaseGameOver", g.Phase)
	}
	if g.Winner != chess.White {
		t.Errorf("Winner = %v, want White", g.Winner)
	}
	if g.Pending != nil {
		t.Error("Pending should be cleared on game-end commit")
	}
}

func TestPhase2_ResurrectionFlowCompletesTurn(t *testing.T) {
	// PlayCard(+4) → PlayResurrection → Phase = PhaseTurnComplete.
	wild := card(models.Wild, models.Pl4)
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{wild, card(models.Blue, "9")}, // no Red → +4 is legal
		models.Deck{card(models.Green, "3")},
	)
	// Pretend White has a captured queen sitting in the pool.
	g.Captured[chess.White] = []chess.PieceType{chess.Queen}

	res, err := PlayCard(g, wild, models.Green)
	if err != nil {
		t.Fatalf("PlayCard: %v", err)
	}
	if g.Phase != models.PhaseAwaitingResurrection {
		t.Fatalf("Phase after PlayCard = %v, want PhaseAwaitingResurrection", g.Phase)
	}

	err = PlayResurrection(g, res.Card, []Resurrection{
		{Piece: chess.Queen, Square: chess.NewSquare(chess.FileE, chess.Rank3)},
	})
	if err != nil {
		t.Fatalf("PlayResurrection: %v", err)
	}
	if g.Phase != models.PhaseTurnComplete {
		t.Errorf("Phase after PlayResurrection = %v, want PhaseTurnComplete", g.Phase)
	}
	if len(g.Captured[chess.White]) != 0 {
		t.Errorf("queen should have been consumed, pool = %v", g.Captured[chess.White])
	}
	if g.ChessEngine.Position().Board().Piece(chess.E3) != chess.WhiteQueen {
		t.Errorf("expected White queen on e3")
	}
}

// --- DeclareStartingColor (AC-3) ------------------------------------------

func TestDeclareStartingColor_RecolorsWildTop(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Wild, models.Pl4),
		models.Deck{card(models.Red, "5")},
		models.Deck{card(models.Blue, "9")},
	)

	if err := DeclareStartingColor(g, models.Red); err != nil {
		t.Fatalf("DeclareStartingColor: %v", err)
	}
	if topOfDiscard(g).Color != models.Red {
		t.Errorf("top color = %v, want Red", topOfDiscard(g).Color)
	}
}

func TestDeclareStartingColor_RejectsAfterFirstTurn(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Wild, models.Pl4),
		models.Deck{card(models.Red, "5")},
		models.Deck{card(models.Blue, "9")},
	)
	g.History = append(g.History, models.TurnRecord{Player: chess.White})

	if err := DeclareStartingColor(g, models.Red); !errors.Is(err, ErrNotStartOfGame) {
		t.Fatalf("expected ErrNotStartOfGame, got %v", err)
	}
}

func TestDeclareStartingColor_RejectsNonWildTop(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Red, "5"),
		models.Deck{card(models.Red, "9")},
		models.Deck{card(models.Blue, "9")},
	)

	if err := DeclareStartingColor(g, models.Blue); !errors.Is(err, ErrNoWildToDeclare) {
		t.Fatalf("expected ErrNoWildToDeclare, got %v", err)
	}
}

func TestDeclareStartingColor_RejectsInvalidColor(t *testing.T) {
	g := newTurnTestGame(t,
		card(models.Wild, models.Pl4),
		models.Deck{card(models.Red, "5")},
		models.Deck{card(models.Blue, "9")},
	)

	if err := DeclareStartingColor(g, models.Wild); !errors.Is(err, ErrInvalidWildColor) {
		t.Fatalf("expected ErrInvalidWildColor, got %v", err)
	}
}
