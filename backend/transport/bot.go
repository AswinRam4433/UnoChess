package transport

import (
	"time"

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
//
// afterSubMove broadcasts the current state and then briefly releases the lock so
// the human client can receive and render each intermediate board position before
// the next sub-move lands. The brief release is safe here: the human player
// cannot send commands while it is the bot's turn (the active-color guard in
// dispatchCommand rejects them), so there is no meaningful contention.
func (s *GameSession) runBotTurn() {
	s.mu.Lock()
	defer s.mu.Unlock()

	afterSubMove := func() {
		s.broadcastLocked()
		s.mu.Unlock()
		time.Sleep(350 * time.Millisecond)
		s.mu.Lock()
	}

	for s.state == sessionPlaying &&
		s.game.Phase == models.PhaseAwaitingCard &&
		s.game.ActiveColor == chess.Black {

		if _, err := gameloop.RunBotTurn(s.game, gameloop.PreferCapturesAndChecks, afterSubMove); err != nil {
			break
		}
		s.broadcastLocked()

		if s.game.Phase == models.PhaseGameOver {
			break
		}
	}
}
