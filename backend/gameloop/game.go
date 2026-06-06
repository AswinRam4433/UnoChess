package gameloop

import (
	"math/rand/v2"

	"github.com/notnil/chess"

	"unochess/models"
)

// startingHandSize is the number of Uno cards each player is dealt at setup.
const startingHandSize = 7

// NewUnoChessGameWith builds a fresh two-player UnoChess game seeded by the given
// RNG. Pair it with rand.New(rand.NewPCG(seed1, seed2)) for byte-for-byte
// reproducible games — the property RunGame's determinism tests depend on. White
// moves first. A wild card flipped as the opening discard is left colorless here;
// resolving it is a turn-flow concern (DeclareStartingColor / RunGame at setup).
func NewUnoChessGameWith(rng *rand.Rand) *models.UnoChessGame {
	drawPile := initialiseFullUnoDeckWith(rng)

	hands := map[chess.Color]models.Deck{
		chess.White: drawPile.DealStartingUnoCardsWith(rng, startingHandSize),
		chess.Black: drawPile.DealStartingUnoCardsWith(rng, startingHandSize),
	}

	topCard := drawPile[0]
	drawPile = drawPile[1:]

	return &models.UnoChessGame{
		ChessEngine:   chess.NewGame(),
		History:       []models.TurnRecord{},
		Hands:         hands,
		DrawPile:      drawPile,
		DiscardPile:   models.Deck{topCard},
		ActiveColor:   chess.White,
		PlayDirection: 1,
		Captured:      map[chess.Color][]chess.PieceType{},
		Phase:         models.PhaseAwaitingCard,
		Winner:        chess.NoColor,
	}
}

// NewUnoChessGame builds a fresh two-player UnoChess game using the package-level
// RNG (non-deterministic). White moves first. A wild card flipped as the opening
// discard is left colorless here; resolving it is the orchestrator's job.
func NewUnoChessGame() *models.UnoChessGame {
	drawPile := InitialiseFullUnoDeck()

	hands := map[chess.Color]models.Deck{
		chess.White: drawPile.DealStartingUnoCards(startingHandSize),
		chess.Black: drawPile.DealStartingUnoCards(startingHandSize),
	}

	topCard := drawPile[0]
	drawPile = drawPile[1:]

	return &models.UnoChessGame{
		ChessEngine:   chess.NewGame(),
		History:       []models.TurnRecord{},
		Hands:         hands,
		DrawPile:      drawPile,
		DiscardPile:   models.Deck{topCard},
		ActiveColor:   chess.White,
		PlayDirection: 1,
		Captured:      map[chess.Color][]chess.PieceType{},
		Phase:         models.PhaseAwaitingCard,
		Winner:        chess.NoColor,
	}
}
