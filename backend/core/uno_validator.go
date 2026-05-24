package core

import models "unochess/models"

// GetValidUnoMoves returns the list of valid moves for the player based on their cardsInHand.
// It enforces the Wild Draw Four "last resort" rule: a +4 is only legal when the player
// holds no card matching the current active color.
func GetValidUnoMoves(topCard models.UnoCard, cardsInHand []models.UnoCard) []models.UnoCard {
	validMoves := []models.UnoCard{}

	hasColorMatch := false
	for _, card := range cardsInHand {
		if card.Color == topCard.Color {
			hasColorMatch = true
			break
		}
	}

	for _, card := range cardsInHand {
		// Wild Draw Four may only be played as a last resort — skip it while the
		// player still holds a color-matching card they could play instead.
		if card.Value == models.Pl4 && hasColorMatch {
			continue
		}
		if IsValidUnoMove(topCard, card) {
			validMoves = append(validMoves, card)
		}
	}

	return validMoves
}

// IsValidUnoMove tells us if a card can be played now
func IsValidUnoMove(topCard models.UnoCard, curCard models.UnoCard) bool {

	if curCard.Value == models.Pl4 || // Wild Draw Four — always playable
		curCard.Value == models.WildCard || // plain Wild — always playable
		topCard.Color == curCard.Color || // check if color matches
		topCard.Value == curCard.Value { // check if number matches
		return true
	}

	return false
}

// CanPlayUnoMove tells us if the player can play a valid Uno move with their cards in hand
func CanPlayUnoMove(topCard models.UnoCard, cardsInHand []models.UnoCard) bool {
	return len(GetValidUnoMoves(topCard, cardsInHand)) != 0
}
