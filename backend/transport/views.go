package transport

import (
	"github.com/notnil/chess"

	"unochess/gameloop"
	"unochess/models"
)

// CardView is the JSON shape of a single Uno card.
type CardView struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

// ComboView describes an in-progress number-card chess combo.
type ComboView struct {
	MovesRemaining int `json:"movesRemaining"`
}

// TurnView is the public record of one completed turn (no hidden hand info).
type TurnView struct {
	Player      string   `json:"player"`
	CardPlayed  CardView `json:"cardPlayed"`
	BoardStates []string `json:"boardStates"`
}

// PlayerView is the filtered game snapshot sent to one player. The opponent's
// hand contents are never included — only the count. This is the primary
// security boundary of the transport layer.
type PlayerView struct {
	Type              string     `json:"type"` // always "state"
	Phase             string     `json:"phase"`
	YourColor         string     `json:"yourColor"`
	ActiveColor       string     `json:"activeColor"`
	YourHand          []CardView `json:"yourHand"`
	OpponentHandCount int        `json:"opponentHandCount"`
	DiscardTop        CardView   `json:"discardTop"`
	DrawPileSize      int        `json:"drawPileSize"`
	BoardFEN          string     `json:"boardFEN"`
	PendingCombo      *ComboView `json:"pendingCombo,omitempty"`
	History           []TurnView `json:"history"`
	Winner            *string    `json:"winner"`
	Reason            *string    `json:"reason"`
}

// gameOverMsg is sent after the terminal PlayerView when a game ends.
type gameOverMsg struct {
	Type   string `json:"type"` // always "game_over"
	Winner string `json:"winner"`
	Reason string `json:"reason"`
	Turns  int    `json:"turns"`
}

// errMsg is sent to the command sender when their request was rejected.
type errMsg struct {
	Type    string `json:"type"` // always "error"
	Code    string `json:"code"`
	Message string `json:"message"`
}

// buildPlayerView constructs the filtered view for color. Must be called with
// s.mu held (or when no other goroutine can mutate s.game).
func buildPlayerView(s *GameSession, color chess.Color) PlayerView {
	g := s.game
	opponent := color.Other()

	hand := make([]CardView, len(g.Hands[color]))
	for i, c := range g.Hands[color] {
		hand[i] = cardView(c)
	}

	hist := make([]TurnView, len(g.History))
	for i, rec := range g.History {
		states := rec.BoardStates
		if states == nil {
			states = []string{}
		}
		hist[i] = TurnView{
			Player:      colorName(rec.Player),
			CardPlayed:  cardView(rec.CardPlayed),
			BoardStates: states,
		}
	}

	v := PlayerView{
		Type:              "state",
		Phase:             phaseName(g.Phase),
		YourColor:         colorName(color),
		ActiveColor:       colorName(g.ActiveColor),
		YourHand:          hand,
		OpponentHandCount: len(g.Hands[opponent]),
		DiscardTop:        discardTopView(g),
		DrawPileSize:      len(g.DrawPile),
		BoardFEN:          g.ChessEngine.Position().String(),
		History:           hist,
	}

	if g.Pending != nil {
		v.PendingCombo = &ComboView{MovesRemaining: g.Pending.MovesRemaining}
	}

	if g.Phase == models.PhaseGameOver {
		winner := colorName(g.Winner)
		reason := string(gameloop.ClassifyEnd(g))
		v.Winner = &winner
		v.Reason = &reason
	}

	return v
}

func buildGameOverMsg(s *GameSession) gameOverMsg {
	g := s.game
	return gameOverMsg{
		Type:   "game_over",
		Winner: colorName(g.Winner),
		Reason: string(gameloop.ClassifyEnd(g)),
		Turns:  len(g.History),
	}
}

func cardView(c models.UnoCard) CardView {
	return CardView{Value: string(c.Value), Color: string(c.Color)}
}

func discardTopView(g *models.UnoChessGame) CardView {
	if len(g.DiscardPile) == 0 {
		return CardView{}
	}
	return cardView(g.DiscardPile[len(g.DiscardPile)-1])
}

func colorName(c chess.Color) string {
	switch c {
	case chess.White:
		return "White"
	case chess.Black:
		return "Black"
	default:
		return ""
	}
}

func phaseName(p models.TurnPhase) string {
	switch p {
	case models.PhaseAwaitingCard:
		return "AwaitingCard"
	case models.PhaseInCombo:
		return "InCombo"
	case models.PhaseAwaitingResurrection:
		return "AwaitingResurrection"
	case models.PhaseTurnComplete:
		return "TurnComplete"
	case models.PhaseGameOver:
		return "GameOver"
	default:
		return "Unknown"
	}
}
