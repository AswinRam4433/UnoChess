// Package gameloop holds the logic driving the turns for each game
package gameloop

import (
	"fmt"
	"unochess/core" // Imported to use your validators
	"unochess/models"
)

// Deck aliases models.Deck so existing gameloop code keeps the short name. The type
// and its data methods now live in models; gameloop layers only strategy on top.
type Deck = models.Deck

func InitUnoGame(playersCount int) {
	drawPile := InitialiseFullUnoDeck()
	allPlayersHands := DealAllPlayerHands(playersCount, &drawPile)

	if len(drawPile) == 0 {
		fmt.Println("Error: Deck is empty before game even started!")
		return
	}

	// Take the first card from the draw pile as the top card, and seed the
	// discard pile with it so it can later be reshuffled back into the draw pile.
	topCard := drawPile[0]
	drawPile = drawPile[1:]
	discardPile := Deck{topCard}
	fmt.Printf("Game started! Initial top card: [%s %s]\n", topCard.Color, topCard.Value)

	currentPlayer := 0
	direction := 1 // 1 for forward, -1 for reverse

	// If the first flipped card is a wild it carries no color, so the first player
	// declares the starting active color. (The discard pile keeps the original
	// wild, so it reverts to a wild if it is ever reshuffled back into play.)
	if topCard.Color == models.Wild {
		chosenColor := ChooseWildColor(allPlayersHands[currentPlayer])
		topCard.Color = chosenColor
		fmt.Printf("🎨 First card is wild — Player %d sets the active color to %s\n", currentPlayer, chosenColor)
	}

	// stalledTurns counts consecutive turns in which a player neither played nor
	// drew a card. Once an entire round passes with no progress (draw and discard
	// piles both exhausted), the game is a draw. This is what prevents the
	// infinite loop that previously occurred when nobody could move.
	stalledTurns := 0

	// maxTurns is a last-resort safety valve. Termination is already guaranteed
	// by the stalemate guard, but this backstop ensures a future logic error can
	// never spin the loop unbounded (and fill the disk) again.
	const maxTurns = 100_000

	for turnCount := 1; ; turnCount++ {
		if turnCount > maxTurns {
			fmt.Printf("\n🛑 Safety limit of %d turns reached — ending game to avoid a runaway loop.\n", maxTurns)
			break
		}

		// turnPlayer is the player acting this turn; it stays fixed for the whole
		// turn so messaging (and the win announcement) always names the right
		// player even after a Skip changes who plays next.
		turnPlayer := currentPlayer
		fmt.Printf("\n--- Player %d's Turn ---\n", turnPlayer)
		fmt.Printf("Top Card: [%s %s]\n", topCard.Color, topCard.Value)

		hand := &allPlayersHands[turnPlayer]

		fmt.Printf("The current hand of the player is:%s\n", hand.PrintDeck())

		validMoves := core.GetValidUnoMoves(topCard, []models.UnoCard(*hand))
		fmt.Printf("Valid moves in hand: %v\n", validMoves)

		steps := 1          // how many seats to advance after this turn
		progressed := false // did a card get played or drawn this turn?

		var played *models.UnoCard // the card chosen to play this turn, if any
		fromDraw := false          // was it the card just drawn?

		if len(validMoves) > 0 {
			// The bot prefers low-impact moves rather than blindly dumping wilds.
			chosen := ChooseMove(validMoves)
			played = &chosen
		} else {
			fmt.Printf("Player %d has no valid moves. Drawing a card...\n", turnPlayer)
			if drawCards(hand, &drawPile, &discardPile, 1) > 0 {
				progressed = true

				// The freshly drawn card may be playable immediately.
				drawn := (*hand)[len(*hand)-1]
				if core.IsValidUnoMove(topCard, drawn) {
					played = &drawn
					fromDraw = true
				}
			}
		}

		if played != nil {
			playedCard := *played
			progressed = true
			if fromDraw {
				fmt.Printf("Player %d plays the drawn card immediately: [%s %s]\n", turnPlayer, playedCard.Color, playedCard.Value)
			} else {
				fmt.Printf("Player %d plays: [%s %s]\n", turnPlayer, playedCard.Color, playedCard.Value)
			}

			hand.RemoveCard(playedCard)
			discardPile = append(discardPile, playedCard)
			topCard = playedCard // Played card becomes the new top card

			switch playedCard.Value {
			case models.Rev:
				direction = -direction
				fmt.Println("🔄 Direction reversed!")
				if playersCount == 2 {
					// With two players a Reverse behaves like a Skip: play comes
					// straight back to the current player.
					steps = 2
					fmt.Println("↩️  Two-player game — Reverse acts as a Skip.")
				}
			case models.Skip:
				skipped := wrapSeat(turnPlayer+direction, playersCount)
				steps = 2 // advance an extra seat past the skipped player
				fmt.Printf("🚫 Player %d was skipped!\n", skipped)
			case models.Pl2:
				nextPlayer := wrapSeat(turnPlayer+direction, playersCount)
				drawCards(&allPlayersHands[nextPlayer], &drawPile, &discardPile, 2)
				fmt.Printf("🌊 Player %d had to draw 2 cards!\n", nextPlayer)
			case models.Pl4:
				nextPlayer := wrapSeat(turnPlayer+direction, playersCount)
				drawCards(&allPlayersHands[nextPlayer], &drawPile, &discardPile, 4)
				fmt.Printf("🔥 Player %d had to draw 4 cards!\n", nextPlayer)

				// A wild card carries no color of its own, so the player must
				// declare a new active color for matching to continue.
				chosenColor := ChooseWildColor(*hand)
				topCard.Color = chosenColor
				fmt.Printf("🎨 Player %d set the active color to %s\n", turnPlayer, chosenColor)
			case models.WildCard:
				// Plain Wild only recolors play — no one draws.
				chosenColor := ChooseWildColor(*hand)
				topCard.Color = chosenColor
				fmt.Printf("🎨 Player %d set the active color to %s\n", turnPlayer, chosenColor)
			}
		}

		// Check Win Condition / Uno shoutouts
		if hand.CheckGameWon() {
			fmt.Printf("\n🏆 Player %d won the game!\n", turnPlayer)
			break
		} else if hand.ShouldShoutUno() {
			fmt.Printf("\n📣 Player %d shouts: UNO!!\n", turnPlayer)
		}

		// Stalemate detection: if a full round elapses with no card played or
		// drawn, no progress is possible (both piles are exhausted) — call it a draw.
		if progressed {
			stalledTurns = 0
		} else {
			stalledTurns++
			if stalledTurns >= playersCount {
				fmt.Println("\n🤝 No player can move and the deck is exhausted — the game is a draw.")
				break
			}
		}

		// Move to the next player, honoring direction and any skip.
		currentPlayer = wrapSeat(turnPlayer+steps*direction, playersCount)
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

	for count := 0; count < 4; count++ { // ...and 4 plain Wild cards
		startingFullUnoDeck = append(startingFullUnoDeck, models.UnoCard{Value: models.WildCard, Color: models.Wild})
	}

	// Shuffle so both the deal and the residual draw pile come out in random order.
	startingFullUnoDeck.Shuffle()

	return startingFullUnoDeck
}

// reshuffleDiscardIntoDraw moves every discard except the current top card back
// into the draw pile and shuffles it, letting play continue once the draw pile
// runs dry. It returns false when there is nothing left to reclaim (fewer than
// two cards in the discard pile).
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

// drawCards pulls up to count cards from the draw pile into hand, reshuffling the
// discard pile back in when the draw pile empties. It returns how many cards were
// actually drawn, which is fewer than count only when every pile is exhausted.
func drawCards(hand *Deck, drawPile *Deck, discardPile *Deck, count int) int {
	drawn := 0
	for i := 0; i < count; i++ {
		if len(*drawPile) == 0 {
			if !reshuffleDiscardIntoDraw(drawPile, discardPile) {
				fmt.Println("⚠️ Draw and discard piles are both exhausted — no card drawn.")
				return drawn
			}
			fmt.Println("♻️  Draw pile empty — reshuffled the discard pile back in.")
		}

		// Pull card off top of drawPile, append to player deck
		card := (*drawPile)[0]
		*drawPile = (*drawPile)[1:]
		*hand = append(*hand, card)
		drawn++
	}
	return drawn
}

// wrapSeat reduces a seat index into [0, playersCount), handling the negative
// values that arise when play is running in reverse.
func wrapSeat(seat, playersCount int) int {
	seat %= playersCount
	if seat < 0 {
		seat += playersCount
	}
	return seat
}

// wrapSeat and the deck helpers above are the only turn-flow utilities that remain in
// gameloop; the Deck data methods (Shuffle, deal, RemoveCard, win predicates, Print)
// now live in package models.
