package transport

import (
	"github.com/notnil/chess"

	"unochess/gameloop"
	"unochess/models"
)

// runBotTurn drives the server-side bot (Black) player's turn. It is called in
// its own goroutine so it never blocks the human player's WS reader goroutine.
// It re-acquires the session lock after being spawned and re-validates conditions
// in case the game state changed between the spawn and the lock acquisition.
//
// The inner loop handles consecutive bot turns that arise when the bot plays a
// Skip or Reverse (which returns play to Black in a two-player game).
func (s *GameSession) runBotTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.state == sessionPlaying &&
		s.game.Phase == models.PhaseAwaitingCard &&
		s.game.ActiveColor == chess.Black {

		if _, err := gameloop.RunBotTurn(s.game, gameloop.PreferCapturesAndChecks); err != nil {
			break
		}
		s.broadcastLocked()

		if s.game.Phase == models.PhaseGameOver {
			break
		}
	}
}
