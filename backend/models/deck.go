package models

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Deck is used universally for draw piles, discard piles, and player hands.
type Deck []UnoCard

// Shuffle randomizes the order of the cards in place using the package-level v2
// RNG. Convenient for one-shot/non-deterministic callers; tests and reproducible
// games should use ShuffleWith with a seeded *rand.Rand.
func (d Deck) Shuffle() {
	rand.Shuffle(len(d), func(i, j int) {
		d[i], d[j] = d[j], d[i]
	})
}

// ShuffleWith randomizes the order of the cards in place using the supplied RNG.
// Pair it with rand.New(rand.NewPCG(seed1, seed2)) for byte-reproducible games.
func (d Deck) ShuffleWith(r *rand.Rand) {
	r.Shuffle(len(d), func(i, j int) {
		d[i], d[j] = d[j], d[i]
	})
}

// DealStartingUnoCards removes cardsCount cards at random from the deck and returns
// them as a new hand, shrinking the receiver. It deals fewer than requested only if
// the deck runs out first. Uses the package-level v2 RNG; see DealStartingUnoCardsWith
// for a deterministic variant.
func (startingDeck *Deck) DealStartingUnoCards(cardsCount int) Deck {
	return startingDeck.dealWith(cardsCount, func(n int) int { return rand.IntN(n) })
}

// DealStartingUnoCardsWith is the seeded variant of DealStartingUnoCards.
func (startingDeck *Deck) DealStartingUnoCardsWith(r *rand.Rand, cardsCount int) Deck {
	return startingDeck.dealWith(cardsCount, r.IntN)
}

// dealWith is the shared implementation behind both deal variants. randIntN supplies
// the integer-in-range source — either the package-level v2 RNG or an injected one.
func (startingDeck *Deck) dealWith(cardsCount int, randIntN func(int) int) Deck {
	playerHand := make(Deck, cardsCount)

	for i := 0; i < cardsCount; i++ {
		if len(*startingDeck) == 0 {
			break
		}

		randomIndex := randIntN(len(*startingDeck))
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
