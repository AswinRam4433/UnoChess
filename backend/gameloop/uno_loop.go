// Package gameloop holds the logic driving the turns for each game
package gameloop

import (
	"fmt"
	"math/rand/v2"
	"unochess/core" // Imported to use your validators
	"unochess/models"
)

// Deck is used universally for draw piles, discard piles, and player hands.
type Deck []models.UnoCard

func InitUnoGame(playersCount int) {
	drawPile := InitialiseFullUnoDeck()
	allPlayersHands := DealAllPlayerHands(playersCount, &drawPile)

	if len(drawPile) == 0 {
		fmt.Println("Error: Deck is empty before game even started!")
		return
	}

	// Take the first card from the draw pile as the top card
	topCard := drawPile[0]
	drawPile = drawPile[1:]
	fmt.Printf("Game started! Initial top card: [%s %s]\n", topCard.Color, topCard.Value)

	currentPlayer := 0
	direction := 1 // 1 for forward, -1 for reverse

	for {
		fmt.Printf("\n--- Player %d's Turn ---\n", currentPlayer)
		fmt.Printf("Top Card: [%s %s]\n", topCard.Color, topCard.Value)

		hand := &allPlayersHands[currentPlayer]

		fmt.Printf("The current hand of the player is:%s\n", hand.PrintDeck())

		validMoves := core.GetValidUnoMoves(topCard, []models.UnoCard(*hand))
		fmt.Printf("Valid moves in hand: %v\n", validMoves)

		if len(validMoves) > 0 {
			// Player has a valid move! For now, let's just make them play their first valid option
			playedCard := validMoves[0]
			fmt.Printf("Player %d plays: [%s %s]\n", currentPlayer, playedCard.Color, playedCard.Value)

			hand.RemoveCard(playedCard)
			topCard = playedCard // Played card becomes the new top card

			switch playedCard.Value {
			case models.Rev:
				direction = -direction
				fmt.Println("🔄 Direction reversed!")
			case models.Skip:
				currentPlayer = (currentPlayer + direction + playersCount) % playersCount
				fmt.Printf("🚫 Player %d was skipped!\n", currentPlayer)
			case models.Pl2:
				nextPlayer := (currentPlayer + direction + playersCount) % playersCount
				allPlayersHands[nextPlayer].DrawCard(&drawPile, 2)
				fmt.Printf("🌊 Player %d had to draw 2 cards!\n", nextPlayer)
			case models.Pl4:
				nextPlayer := (currentPlayer + direction + playersCount) % playersCount
				allPlayersHands[nextPlayer].DrawCard(&drawPile, 4)
				fmt.Printf("🔥 Player %d had to draw 4 cards!\n", nextPlayer)

				// A wild card carries no color of its own, so the player must
				// declare a new active color for matching to continue.
				chosenColor := ChooseWildColor(*hand)
				topCard.Color = chosenColor
				fmt.Printf("🎨 Player %d set the active color to %s\n", currentPlayer, chosenColor)
			}

		} else {
			fmt.Printf("Player %d has no valid moves. Drawing a card...\n", currentPlayer)
			hand.DrawCard(&drawPile, 1)

			// Check if the newly drawn card can be played immediately
			newlyDrawnCard := (*hand)[len(*hand)-1]
			if core.IsValidUnoMove(topCard, newlyDrawnCard) {
				fmt.Printf("Player %d plays the drawn card immediately: [%s %s]\n", currentPlayer, newlyDrawnCard.Color, newlyDrawnCard.Value)
				topCard = newlyDrawnCard
				hand.RemoveCard(newlyDrawnCard)
			}
		}

		// Check Win Condition / Uno shoutouts
		if hand.CheckGameWon() {
			fmt.Printf("\n🏆 Player %d won the game!\n", currentPlayer)
			break
		} else if hand.ShouldShoutUno() {
			fmt.Printf("\n📣 Player %d shouts: UNO!!\n", currentPlayer)
		}

		// Move to the next player smoothly handling reverse step patterns
		currentPlayer = (currentPlayer + direction + playersCount) % playersCount
	}
}

// DealAllPlayerHands now accepts the live draw pile pointer so we pull directly from it
func DealAllPlayerHands(playersCount int, drawPile *Deck) []Deck {
	var allPlayerHands []Deck

	for i := 0; i < playersCount; i++ {
		curPlayerHand := drawPile.DealStartingUnoCards(7)
		allPlayerHands = append(allPlayerHands, curPlayerHand)
	}
	return allPlayerHands
}

func InitialiseFullUnoDeck() Deck {
	startingFullUnoDeck := make(Deck, 0, 100)

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

	for count := 0; count < 4; count++ { // Uno usually has 4 Wild Draw 4s
		startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: models.Pl4, Color: models.Wild})
	}

	return startingFullUnoDeck
}

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

// DrawCard safely takes N cards from the draw pile and pushes them into the hand
func (d *Deck) DrawCard(drawPile *Deck, count int) {
	for i := 0; i < count; i++ {
		if len(*drawPile) == 0 {
			fmt.Println("⚠️ Draw pile is empty!") // In a production game, you'd reshuffle the discard pile here
			return
		}
		// Pull card off top of drawPile, append to player deck
		card := (*drawPile)[0]
		*drawPile = (*drawPile)[1:]
		*d = append(*d, card)
	}
}

// RemoveCard searches a hand for a specific card and purges it
func (d *Deck) RemoveCard(target models.UnoCard) {
	for i, card := range *d {
		if card.Value == target.Value && card.Color == target.Color {
			// Efficient Order-destructive swap and trim
			(*d)[i] = (*d)[len(*d)-1]
			*d = (*d)[:len(*d)-1]
			return
		}
	}
}

func (d Deck) CheckGameWon() bool {
	return len(d) == 0
}

func (d Deck) ShouldShoutUno() bool {
	return len(d) == 1
}
