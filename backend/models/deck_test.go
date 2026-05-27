package models

import "testing"

func TestDeckRemoveCard(t *testing.T) {
	d := Deck{
		{Value: "5", Color: Red},
		{Value: Skip, Color: Blue},
		{Value: "5", Color: Green},
	}
	d.RemoveCard(UnoCard{Value: Skip, Color: Blue})

	if len(d) != 2 {
		t.Fatalf("expected 2 cards after removal, got %d", len(d))
	}
	for _, c := range d {
		if c.Value == Skip && c.Color == Blue {
			t.Error("removed card is still present")
		}
	}
}

func TestDeckRemoveCardMissingIsNoop(t *testing.T) {
	d := Deck{{Value: "5", Color: Red}}
	d.RemoveCard(UnoCard{Value: "9", Color: Yellow})
	if len(d) != 1 {
		t.Errorf("removing an absent card should leave the deck unchanged, got %d", len(d))
	}
}

func TestDeckRemoveCardOnlyFirstMatch(t *testing.T) {
	// Two identical cards: only one should be purged.
	d := Deck{
		{Value: "5", Color: Red},
		{Value: "5", Color: Red},
	}
	d.RemoveCard(UnoCard{Value: "5", Color: Red})
	if len(d) != 1 {
		t.Errorf("expected exactly one of the duplicates removed, got len %d", len(d))
	}
}

func TestDeckWinPredicates(t *testing.T) {
	if !(Deck{}).CheckGameWon() {
		t.Error("empty hand should report the game won")
	}
	if (Deck{{Value: "1", Color: Red}}).CheckGameWon() {
		t.Error("non-empty hand should not report the game won")
	}
	if !(Deck{{Value: "1", Color: Red}}).ShouldShoutUno() {
		t.Error("single-card hand should shout UNO")
	}
	if (Deck{}).ShouldShoutUno() {
		t.Error("empty hand should not shout UNO")
	}
}
