package models

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Deck is used universally for draw piles, discard piles, and player hands.
type Deck []UnoCard

// Shuffle randomizes the order of the cards in place.
func (d Deck) Shuffle() {
	rand.Shuffle(len(d), func(i, j int) {
		d[i], d[j] = d[j], d[i]
	})
}

// DealStartingUnoCards removes cardsCount cards at random from the deck and returns
// them as a new hand, shrinking the receiver. It deals fewer than requested only if
// the deck runs out first.
func (startingDeck *Deck) DealStartingUnoCards(cardsCount int) Deck {
	playerHand := make(Deck, cardsCount)

	for i := 0; i < cardsCount; i++ {
		if len(*startingDeck) == 0 {
			break
		}

		randomIndex := rand.IntN(len(*startingDeck))
		pickedItem := (*startingDeck)[randomIndex]
		playerHand[i] = pickedItem

		lastIndex := len(*startingDeck) - 1
		(*startingDeck)[randomIndex] = (*startingDeck)[lastIndex]
		*startingDeck = (*startingDeck)[:lastIndex]
	}

	return playerHand
}

// RemoveCard searches the deck for a specific card and purges the first match.
func (d *Deck) RemoveCard(target UnoCard) {
	for i, card := range *d {
		if card.Value == target.Value && card.Color == target.Color {
			// Efficient order-destructive swap and trim.
			(*d)[i] = (*d)[len(*d)-1]
			*d = (*d)[:len(*d)-1]
			return
		}
	}
}

// CheckGameWon reports whether the hand is empty — its owner has gone out.
func (d Deck) CheckGameWon() bool {
	return len(d) == 0
}

// ShouldShoutUno reports whether the hand is down to its final card.
func (d Deck) ShouldShoutUno() bool {
	return len(d) == 1
}

// PrintDeck renders the deck as a human-readable, multi-line string.
func (d *Deck) PrintDeck() string {
	var sb strings.Builder
	for k, v := range *d {
		sb.WriteString(fmt.Sprintf("\nCard %d:\tColor:%s and Value:%s", k+1, v.Color, v.Value))
	}

	return sb.String()
}
