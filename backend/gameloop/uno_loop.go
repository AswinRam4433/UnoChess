// Package gameloop holds the logic driving the turns for each game
package gameloop

import (
	"math/rand/v2"

	"unochess/models"
)

// Deck aliases models.Deck so existing gameloop code keeps the short name.
type Deck = models.Deck

// InitialiseFullUnoDeck builds and shuffles the 104-card house deck using the
// package-level RNG. See initialiseFullUnoDeckWith for the seeded variant.
func InitialiseFullUnoDeck() Deck {
	deck := buildFullUnoDeck()
	deck.Shuffle()
	return deck
}

// initialiseFullUnoDeckWith builds the 104-card house deck and shuffles it with
// the caller's RNG, so a seeded source produces a deterministic ordering.
func initialiseFullUnoDeckWith(r *rand.Rand) Deck {
	deck := buildFullUnoDeck()
	deck.ShuffleWith(r)
	return deck
}

// buildFullUnoDeck composes the 104-card house deck (no zeros, 4 Wild Draw 4s, 4
// plain Wilds) in a deterministic order. Callers shuffle it themselves.
func buildFullUnoDeck() Deck {
	startingFullUnoDeck := make(Deck, 0, 104)

	colors := []models.CardColor{models.Red, models.Green, models.Blue, models.Yellow}
	for _, clr := range colors {
		for faceVal := 1; faceVal <= 9; faceVal++ {
			for count := 0; count < 2; count++ {
				startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: models.NumberToCardvalueUnoMap[faceVal], Color: clr})
			}
		}
	}

	actionCards := []models.CardValue{models.Skip, models.Rev, models.Pl2}
	for _, clr := range colors {
		for _, card := range actionCards {
			for count := 0; count < 2; count++ {
				startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: card, Color: clr})
			}
		}
	}

	for count := 0; count < 4; count++ {
		startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: models.Pl4, Color: models.Wild})
	}

	for count := 0; count < 4; count++ {
		startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: models.WildCard, Color: models.Wild})
	}

	return startingFullUnoDeck
}

// reshuffleDiscardIntoDraw moves every discard except the current top card back
// into the draw pile and shuffles it. Returns false when there is nothing left
// to reclaim (fewer than two cards in the discard pile).
func reshuffleDiscardIntoDraw(drawPile *Deck, discardPile *Deck) bool {
	if len(*discardPile) < 2 {
		return false
	}

	last := len(*discardPile) - 1
	top := (*discardPile)[last]
	reclaimed := (*discardPile)[:last]

	// append copies the card values, so it is safe even though reclaimed still
	// aliases the old discard backing array we replace on the next line.
	*drawPile = append(*drawPile, reclaimed...)
	drawPile.Shuffle()
	*discardPile = Deck{top}
	return true
}

// drawCards pulls up to count cards from the draw pile into hand, reshuffling
// the discard pile back in when the draw pile empties. Returns how many cards
// were actually drawn — fewer than count only when every pile is exhausted.
func drawCards(hand *Deck, drawPile *Deck, discardPile *Deck, count int) int {
	drawn := 0
	for i := 0; i < count; i++ {
		if len(*drawPile) == 0 {
			if !reshuffleDiscardIntoDraw(drawPile, discardPile) {
				return drawn
			}
		}

		card := (*drawPile)[0]
		*drawPile = (*drawPile)[1:]
		*hand = append(*hand, card)
		drawn++
	}
	return drawn
}
