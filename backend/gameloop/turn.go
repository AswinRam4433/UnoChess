package gameloop

import (
	"errors"
	"fmt"

	"github.com/notnil/chess"

	"unochess/core"
	"unochess/models"
)

// Errors returned by the turn orchestrator. They are sentinel values so a transport
// layer can map them onto HTTP status codes via errors.Is.
var (
	ErrGameOver         = errors.New("game is already over")
	ErrNotAwaitingCard  = errors.New("the active player is not awaiting a card")
	ErrCardNotInHand    = errors.New("the card is not in the active player's hand")
	ErrIllegalCardPlay  = errors.New("the card does not match the discard top")
	ErrInvalidWildColor = errors.New("a wild card requires a real declared color")
	ErrHasPlayableCard  = errors.New("the active player must play a card when one is available")
	ErrTurnNotComplete  = errors.New("the turn is not yet complete")
	ErrNotStartOfGame   = errors.New("starting color can only be declared before the first turn")
	ErrNoWildToDeclare  = errors.New("the discard top is not a wild card")
)

// PlayCardResult reports what happened when the active player played a card so the
// caller (a bot driver in Phase 3, or a transport handler later) knows what to do
// next without re-reading the whole game state.
type PlayCardResult struct {
	Card          models.UnoCard // the card played, with its post-recolor color for wilds
	UnoWin        bool           // emptied the hand → instant Uno victory
	ComboMoves    int            // chess sub-moves owed (set only when Phase == PhaseInCombo)
	Resurrections int            // pieces allowed (set only when Phase == PhaseAwaitingResurrection)
	SkipsOpponent bool           // a Skip / Reverse cancelled the opponent's turn
	Recolored     bool           // a wild changed the active play color
}

// DrawResult reports a draw attempt.
type DrawResult struct {
	Drew     bool           // a card was actually pulled (false only when both piles are exhausted)
	Card     models.UnoCard // the drawn card (zero value when Drew is false)
	Playable bool           // the drawn card is playable on the current discard top
}

// PlayCard resolves the active player's chosen card: it validates the play against
// the discard top, removes the card from hand, applies any wild-recolor, and either
// declares an instant Uno victory (if the play empties the hand — rulebook §4.2) or
// dispatches into the matching chess-track primitive (number → StartChessCombo,
// +2 / +4 → arm a resurrection). For Skip / Reverse / plain Wild it has no chess
// effect and the turn is left ready for AdvanceTurn.
//
// For non-wild cards declaredColor is ignored; for a wild (Pl4 or WildCard) it must
// be one of Red / Blue / Green / Yellow.
func PlayCard(g *models.UnoChessGame, card models.UnoCard, declaredColor models.CardColor) (PlayCardResult, error) {
	if g.Phase == models.PhaseGameOver {
		return PlayCardResult{}, ErrGameOver
	}
	if g.Phase != models.PhaseAwaitingCard {
		return PlayCardResult{}, fmt.Errorf("%w: phase=%d", ErrNotAwaitingCard, g.Phase)
	}

	active := g.ActiveColor

	if !handContains(g.Hands[active], card) {
		return PlayCardResult{}, fmt.Errorf("%w: %s %s", ErrCardNotInHand, card.Color, card.Value)
	}

	isWild := isWildCard(card)
	if isWild && !isRealColor(declaredColor) {
		return PlayCardResult{}, fmt.Errorf("%w: got %q", ErrInvalidWildColor, declaredColor)
	}

	// Route through GetValidUnoMoves rather than IsValidUnoMove so the +4 last-resort
	// rule (Pl4 only legal when no color match exists) is enforced uniformly.
	top := topOfDiscard(g)
	valid := core.GetValidUnoMoves(top, []models.UnoCard(g.Hands[active]))
	if !cardInList(valid, card) {
		return PlayCardResult{}, fmt.Errorf("%w: %s %s on top %s %s", ErrIllegalCardPlay, card.Color, card.Value, top.Color, top.Value)
	}

	// Move the card from hand to the discard pile, then recolor the new top for wilds.
	hand := g.Hands[active]
	hand.RemoveCard(card)
	g.Hands[active] = hand
	g.DiscardPile = append(g.DiscardPile, card)

	result := PlayCardResult{Card: card}
	if isWild {
		g.DiscardPile[len(g.DiscardPile)-1].Color = declaredColor
		result.Card.Color = declaredColor
		result.Recolored = true
	}

	// Rulebook §4.2: emptying the hand wins immediately — the card's chess effect is
	// skipped entirely, even if it was a number card mid-combo or a +4 resurrection.
	if g.Hands[active].CheckGameWon() {
		result.UnoWin = true
		recordCardOnlyTurn(g, active, result.Card)
		markGameOver(g, active)
		return result, nil
	}

	// Dispatch the played card into the chess-track primitive it owns.
	switch {
	case isNumberCard(card.Value):
		n := models.CardValueToNumberUnoMap[card.Value]
		if err := StartChessCombo(g, result.Card, n); err != nil {
			return PlayCardResult{}, fmt.Errorf("starting combo: %w", err)
		}
		g.Phase = models.PhaseInCombo
		result.ComboMoves = n

	case card.Value == models.Pl2 || card.Value == models.Pl4:
		g.Phase = models.PhaseAwaitingResurrection
		result.Resurrections = ResurrectionCount(card)

	case card.Value == models.Skip || card.Value == models.Rev:
		g.PendingSkip = true
		g.Phase = models.PhaseTurnComplete
		result.SkipsOpponent = true
		recordCardOnlyTurn(g, active, result.Card)

	case card.Value == models.WildCard:
		g.Phase = models.PhaseTurnComplete
		recordCardOnlyTurn(g, active, result.Card)
	}

	return result, nil
}

// DrawForTurn implements rulebook §2.2: when the active player has no playable card
// they must draw one. If the drawn card is playable the player may then PlayCard
// (this function leaves Phase == PhaseAwaitingCard); otherwise the turn ends
// immediately with no chess effect. A second draw is naturally prevented — once a
// playable card is in hand the precondition below rejects another draw.
func DrawForTurn(g *models.UnoChessGame) (DrawResult, error) {
	if g.Phase == models.PhaseGameOver {
		return DrawResult{}, ErrGameOver
	}
	if g.Phase != models.PhaseAwaitingCard {
		return DrawResult{}, fmt.Errorf("%w: phase=%d", ErrNotAwaitingCard, g.Phase)
	}

	active := g.ActiveColor
	top := topOfDiscard(g)
	hand := g.Hands[active]

	if len(core.GetValidUnoMoves(top, []models.UnoCard(hand))) > 0 {
		return DrawResult{}, ErrHasPlayableCard
	}

	n := drawCards(&hand, &g.DrawPile, &g.DiscardPile, 1)
	g.Hands[active] = hand

	if n == 0 {
		// Both piles are exhausted — the turn ends with no card played and no chess
		// effect. This is the path that prevents the old infinite-loop hang.
		g.Phase = models.PhaseTurnComplete
		return DrawResult{}, nil
	}

	drawn := hand[len(hand)-1]
	playable := core.IsValidUnoMove(top, drawn)
	if !playable {
		g.Phase = models.PhaseTurnComplete
	}
	// When playable, Phase stays PhaseAwaitingCard. The player can now PlayCard;
	// DrawForTurn cannot be re-entered because they now hold a playable card.
	return DrawResult{Drew: true, Card: drawn, Playable: playable}, nil
}

// AdvanceTurn passes play to the opponent once the current player's actions are done
// (Phase == PhaseTurnComplete). Skip / Reverse keep the same player active, since in
// a two-player game both cards collapse to "play returns to you" (rulebook §3D/§3E
// plus the 2-player Reverse note in §3D / InitUnoGame). When the game is already
// over this is a no-op so a driver can call it unconditionally at the end of a turn.
func AdvanceTurn(g *models.UnoChessGame) error {
	if g.Phase == models.PhaseGameOver {
		return nil
	}
	if g.Phase != models.PhaseTurnComplete {
		return fmt.Errorf("%w: phase=%d", ErrTurnNotComplete, g.Phase)
	}

	if g.PendingSkip {
		g.PendingSkip = false
	} else {
		g.ActiveColor = g.ActiveColor.Other()
	}
	g.Phase = models.PhaseAwaitingCard
	return nil
}

// DeclareStartingColor resolves a wild opening discard by setting its color before
// turn 1. It is valid only at game start (no turns played yet) and only when the
// discard top is actually a wild card; otherwise the active color is already
// unambiguous and this handler must reject the call. The driver (RunGame) calls it
// automatically at setup using ChooseWildColor against White's hand; a transport
// layer would call it with the human first-player's choice at the same point.
func DeclareStartingColor(g *models.UnoChessGame, color models.CardColor) error {
	if g.Phase != models.PhaseAwaitingCard {
		return fmt.Errorf("%w: phase=%d", ErrNotAwaitingCard, g.Phase)
	}
	if len(g.History) != 0 {
		return ErrNotStartOfGame
	}
	if topOfDiscard(g).Color != models.Wild {
		return ErrNoWildToDeclare
	}
	if !isRealColor(color) {
		return fmt.Errorf("%w: got %q", ErrInvalidWildColor, color)
	}
	g.DiscardPile[len(g.DiscardPile)-1].Color = color
	return nil
}

// --- helpers ---------------------------------------------------------------

// topOfDiscard returns the matchable top card. Assumes the invariant that the
// discard pile is never empty during a live game (NewUnoChessGame seeds it and
// drawCards refuses to consume the top during reshuffles).
func topOfDiscard(g *models.UnoChessGame) models.UnoCard {
	return g.DiscardPile[len(g.DiscardPile)-1]
}

func handContains(d models.Deck, c models.UnoCard) bool {
	for _, x := range d {
		if x.Value == c.Value && x.Color == c.Color {
			return true
		}
	}
	return false
}

func cardInList(list []models.UnoCard, c models.UnoCard) bool {
	for _, x := range list {
		if x.Value == c.Value && x.Color == c.Color {
			return true
		}
	}
	return false
}

func isWildCard(c models.UnoCard) bool {
	return c.Value == models.Pl4 || c.Value == models.WildCard
}

func isRealColor(c models.CardColor) bool {
	switch c {
	case models.Red, models.Blue, models.Green, models.Yellow:
		return true
	}
	return false
}

func isNumberCard(v models.CardValue) bool {
	_, ok := models.CardValueToNumberUnoMap[v]
	return ok
}

func markGameOver(g *models.UnoChessGame, winner chess.Color) {
	g.Phase = models.PhaseGameOver
	g.Winner = winner
}

func recordCardOnlyTurn(g *models.UnoChessGame, player chess.Color, card models.UnoCard) {
	g.History = append(g.History, models.TurnRecord{
		Player:     player,
		CardPlayed: card,
	})
}

// winnerFromOutcome maps a chess engine outcome onto the game-level Winner color.
// A draw or unfinished game yields chess.NoColor, the zero value.
func winnerFromOutcome(o chess.Outcome) chess.Color {
	switch o {
	case chess.WhiteWon:
		return chess.White
	case chess.BlackWon:
		return chess.Black
	default:
		return chess.NoColor
	}
}
