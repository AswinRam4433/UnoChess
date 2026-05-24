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
