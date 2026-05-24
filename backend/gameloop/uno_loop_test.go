package gameloop

import (
	"testing"

	"unochess/models"
)

func card(color models.CardColor, value models.CardValue) models.UnoCard {
	return models.UnoCard{Color: color, Value: value}
}

// TestReshuffleDiscardIntoDraw verifies the discard pile (minus its top card) is
// reclaimed into an empty draw pile, and that the top card is left behind.
func TestReshuffleDiscardIntoDraw(t *testing.T) {
	drawPile := Deck{}
	top := card(models.Red, models.Skip)
	discardPile := Deck{
		card(models.Blue, "5"),
		card(models.Green, "7"),
		card(models.Yellow, models.Rev),
		top, // current top card — must stay on the discard pile
	}

	if !reshuffleDiscardIntoDraw(&drawPile, &discardPile) {
		t.Fatal("expected reshuffle to succeed with 4 discards")
	}
	if len(drawPile) != 3 {
		t.Fatalf("expected 3 cards reclaimed into draw pile, got %d", len(drawPile))
	}
	if len(discardPile) != 1 || discardPile[0] != top {
		t.Fatalf("expected discard pile to keep only the top card %v, got %v", top, discardPile)
	}
}

// TestReshuffleNothingToReclaim verifies that a discard pile holding only the top
// card cannot be reshuffled (this is the genuine deadlock the stalemate guard catches).
func TestReshuffleNothingToReclaim(t *testing.T) {
	drawPile := Deck{}
	discardPile := Deck{card(models.Red, "1")}

	if reshuffleDiscardIntoDraw(&drawPile, &discardPile) {
		t.Fatal("expected reshuffle to fail with a single discard")
	}
	if len(drawPile) != 0 || len(discardPile) != 1 {
		t.Fatalf("piles should be unchanged, got draw=%d discard=%d", len(drawPile), len(discardPile))
	}
}

// TestDrawCardsReshufflesWhenEmpty verifies drawCards transparently reshuffles the
// discard pile when the draw pile runs dry mid-draw.
func TestDrawCardsReshufflesWhenEmpty(t *testing.T) {
	hand := Deck{}
	drawPile := Deck{}
	discardPile := Deck{
		card(models.Blue, "5"),
		card(models.Green, "7"),
		card(models.Yellow, "9"),
		card(models.Red, models.Skip), // top card, stays put
	}

	drawn := drawCards(&hand, &drawPile, &discardPile, 2)
	if drawn != 2 {
		t.Fatalf("expected to draw 2 cards via reshuffle, got %d", drawn)
	}
	if len(hand) != 2 {
		t.Fatalf("expected hand of 2, got %d", len(hand))
	}
}

// TestDrawCardsExhausted verifies drawCards reports a short draw (and does not loop
// forever) when neither pile can supply cards — the condition behind the old hang.
func TestDrawCardsExhausted(t *testing.T) {
	hand := Deck{}
	drawPile := Deck{}
	discardPile := Deck{card(models.Red, "1")} // only the top card, nothing to reclaim

	drawn := drawCards(&hand, &drawPile, &discardPile, 3)
	if drawn != 0 {
		t.Fatalf("expected 0 cards drawn from exhausted piles, got %d", drawn)
	}
	if len(hand) != 0 {
		t.Fatalf("expected empty hand, got %d", len(hand))
	}
}

func TestWrapSeat(t *testing.T) {
	cases := []struct {
		seat, n, want int
	}{
		{0, 2, 0},
		{2, 2, 0},
		{-1, 2, 1},  // reverse from seat 0 in a 2-player game
		{-2, 4, 2},  // two seats back, wrapping around
		{5, 4, 1},   // skip past the end
	}
	for _, c := range cases {
		if got := wrapSeat(c.seat, c.n); got != c.want {
			t.Errorf("wrapSeat(%d, %d) = %d, want %d", c.seat, c.n, got, c.want)
		}
	}
}
