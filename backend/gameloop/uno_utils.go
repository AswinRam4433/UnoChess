package gameloop

import (
	"fmt"
	"strings"
	"unochess/models"
)

func (d *Deck) PrintDeck() string {
	var sb strings.Builder
	for k, v := range *d {
		sb.WriteString(fmt.Sprintf("\nCard %d:\tColor:%s and Value:%s", k+1, v.Color, v.Value))
	}

	return sb.String()
}

// ChooseWildColor picks the active color to declare after playing a wild card.
// It favors whichever real color the player holds most of, so follow-up plays
// are more likely. Iteration order is fixed to keep the choice deterministic.
func ChooseWildColor(hand Deck) models.CardColor {
	colors := []models.CardColor{models.Red, models.Green, models.Blue, models.Yellow}

	counts := map[models.CardColor]int{}
	for _, card := range hand {
		counts[card.Color]++
	}

	bestColor := colors[0]
	bestCount := -1
	for _, color := range colors {
		if counts[color] > bestCount {
			bestCount = counts[color]
			bestColor = color
		}
	}
	return bestColor
}

// ChooseMove picks which of the player's legal moves the bot will play. It favors
// the lowest-impact card so the bot stops blindly leading with draw/wild cards —
// number and action cards go down before draw-twos and wilds. This keeps hands
// shrinking and games converging on a real Uno win. Ties keep hand order.
func ChooseMove(validMoves []models.UnoCard) models.UnoCard {
	best := validMoves[0]
	bestRank := moveImpact(best)
	for _, card := range validMoves[1:] {
		if rank := moveImpact(card); rank < bestRank {
			best = card
			bestRank = rank
		}
	}
	return best
}

// moveImpact ranks a card by how disruptive it is to play (lower = play sooner).
func moveImpact(card models.UnoCard) int {
	switch card.Value {
	case models.Pl4:
		return 4 // forces the opponent to draw 4 — last resort
	case models.Pl2:
		return 3 // forces the opponent to draw 2
	case models.WildCard:
		return 2 // recolors play, no draw; save it when a plain card will do
	case models.Skip, models.Rev:
		return 1
	default:
		return 0 // plain number card
	}
}
