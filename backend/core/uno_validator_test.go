package core

import (
	"testing"

	"unochess/models"
)

func uno(color models.CardColor, value models.CardValue) models.UnoCard {
	return models.UnoCard{Color: color, Value: value}
}

func TestIsValidUnoMove(t *testing.T) {
	top := uno(models.Red, models.CardValue("5"))

	cases := []struct {
		name string
		card models.UnoCard
		want bool
	}{
		{"plain wild always playable", uno(models.Wild, models.WildCard), true},
		{"wild draw four always playable", uno(models.Wild, models.Pl4), true},
		{"color match", uno(models.Red, models.CardValue("9")), true},
		{"value match", uno(models.Blue, models.CardValue("5")), true},
		{"no match", uno(models.Blue, models.CardValue("9")), false},
	}

	for _, c := range cases {
		if got := IsValidUnoMove(top, c.card); got != c.want {
			t.Errorf("%s: IsValidUnoMove(%v, %v) = %v, want %v", c.name, top, c.card, got, c.want)
		}
	}
}

func containsValue(cards []models.UnoCard, v models.CardValue) bool {
	for _, c := range cards {
		if c.Value == v {
			return true
		}
	}
	return false
}

// TestWildDrawFourLastResort verifies a +4 is only offered when the hand has no
// card matching the active color.
func TestWildDrawFourLastResort(t *testing.T) {
	top := uno(models.Red, models.CardValue("5"))

	// A color match exists, so the +4 must be withheld.
	withMatch := []models.UnoCard{
		uno(models.Red, models.CardValue("9")), // matches color
		uno(models.Wild, models.Pl4),
	}
	moves := GetValidUnoMoves(top, withMatch)
	if containsValue(moves, models.Pl4) {
		t.Error("Wild Draw Four should be withheld while a color match is held")
	}
	if !containsValue(moves, models.CardValue("9")) {
		t.Error("the color-matching card should be a valid move")
	}

	// No color match (and no value match): the +4 becomes a legal last resort.
	noMatch := []models.UnoCard{
		uno(models.Blue, models.CardValue("9")),
		uno(models.Wild, models.Pl4),
	}
	moves = GetValidUnoMoves(top, noMatch)
	if !containsValue(moves, models.Pl4) {
		t.Error("Wild Draw Four should be playable when no color match exists")
	}
}
