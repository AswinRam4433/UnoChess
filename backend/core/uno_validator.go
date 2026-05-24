package core

import models "unochess/models"

// GetValidUnoMoves returns the list of valid moves for the player based on their cardsInHand
func GetValidUnoMoves(topCard models.UnoCard, cardsInHand []models.UnoCard) []models.UnoCard {
	validMoves := []models.UnoCard{}

	for _, card := range cardsInHand {
		if IsValidUnoMove(topCard, card) {
			validMoves = append(validMoves, card)
		}

	}

	return validMoves
}

// IsValidUnoMove tells us if a card can be played now
func IsValidUnoMove(topCard models.UnoCard, curCard models.UnoCard) bool {

	if curCard.Value == models.Pl4 || // check if power card
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
