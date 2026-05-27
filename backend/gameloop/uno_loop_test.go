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

// TestDeckComposition verifies the deck is the intended 104-card house deck:
// numbers 1-9 (no zeros), the standard action cards, 4 Wild Draw Four, and 4 Wild.
func TestDeckComposition(t *testing.T) {
	deck := InitialiseFullUnoDeck()

	if len(deck) != 104 {
		t.Fatalf("expected 104 cards, got %d", len(deck))
	}

	counts := map[models.CardValue]int{}
	for _, c := range deck {
		counts[c.Value]++
	}

	if counts[models.CardValue("0")] != 0 {
		t.Errorf("deck should contain no 0 cards, found %d", counts[models.CardValue("0")])
	}
	for n := 1; n <= 9; n++ {
		v := models.NumberToCardvalueUnoMap[n]
		if counts[v] != 8 { // 2 per color * 4 colors
			t.Errorf("expected 8 of number %d, got %d", n, counts[v])
		}
	}
	for _, v := range []models.CardValue{models.Skip, models.Rev, models.Pl2} {
		if counts[v] != 8 {
			t.Errorf("expected 8 of %s, got %d", v, counts[v])
		}
	}
	if counts[models.Pl4] != 4 {
		t.Errorf("expected 4 Wild Draw Four, got %d", counts[models.Pl4])
	}
	if counts[models.WildCard] != 4 {
		t.Errorf("expected 4 plain Wild, got %d", counts[models.WildCard])
	}
}

// TestChooseMovePrefersLowImpact verifies the bot leads with low-impact cards
// instead of dumping draw/wild cards.
func TestChooseMovePrefersLowImpact(t *testing.T) {
	moves := []models.UnoCard{
		card(models.Wild, models.Pl4),
		card(models.Red, models.Pl2),
		card(models.Blue, "7"), // lowest impact — should be chosen
		card(models.Green, models.Skip),
	}
	if got := ChooseMove(moves); got.Value != "7" {
		t.Errorf("expected the number card to be chosen, got %v", got)
	}

	// When only wilds are legal, the plain Wild is preferred over the +4.
	onlyWilds := []models.UnoCard{
		card(models.Wild, models.Pl4),
		card(models.Wild, models.WildCard),
	}
	if got := ChooseMove(onlyWilds); got.Value != models.WildCard {
		t.Errorf("expected plain Wild over Wild Draw Four, got %v", got)
	}
}

func TestWrapSeat(t *testing.T) {
	cases := []struct {
		seat, n, want int
	}{
		{0, 2, 0},
		{2, 2, 0},
		{-1, 2, 1}, // reverse from seat 0 in a 2-player game
		{-2, 4, 2}, // two seats back, wrapping around
		{5, 4, 1},  // skip past the end
	}
	for _, c := range cases {
		if got := wrapSeat(c.seat, c.n); got != c.want {
			t.Errorf("wrapSeat(%d, %d) = %d, want %d", c.seat, c.n, got, c.want)
		}
	}
}
