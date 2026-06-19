package transport

import (
	"context"
	"sync"
	"time"

	"github.com/notnil/chess"

	"unochess/gameloop"
	"unochess/models"
)

const (
	gcTimeout = 30 * time.Minute
	sendDepth = 16
)

type sessionState int

const (
	sessionLobby    sessionState = iota // waiting for Black to join
	sessionPlaying                       // both players have tokens; game in progress
	sessionFinished                      // game ended
)

// Registry is the in-memory catalog of live games, safe for concurrent access.
type Registry struct {
	mu          sync.RWMutex
	games       map[string]*GameSession
	gameFactory func() *models.UnoChessGame // injectable for deterministic tests
}

// NewRegistry returns an empty Registry using the default (non-deterministic) game factory.
func NewRegistry() *Registry {
	return &Registry{
		games:       make(map[string]*GameSession),
		gameFactory: gameloop.NewUnoChessGame,
	}
}

// NewRegistryWithFactory returns a Registry that uses factory to create each new
// UnoChessGame. Intended for tests that need a seeded, deterministic game state.
func NewRegistryWithFactory(factory func() *models.UnoChessGame) *Registry {
	return &Registry{
		games:       make(map[string]*GameSession),
		gameFactory: factory,
	}
}

// newGame creates a GameSession, registers it, and returns it.
func (r *Registry) newGame(id string, isBot bool) *GameSession {
	s := &GameSession{
		id:     id,
		game:   r.gameFactory(),
		tokens: make(map[string]chess.Color),
		conns:  make(map[chess.Color]*playerConn),
		isBot:  isBot,
	}
	if isBot {
		s.state = sessionPlaying
	} else {
		s.state = sessionLobby
	}
	s.gcTimer = time.AfterFunc(gcTimeout, func() { r.remove(id) })

	r.mu.Lock()
	r.games[id] = s
	r.mu.Unlock()
	return s
}

func (r *Registry) get(id string) (*GameSession, bool) {
	r.mu.RLock()
	s, ok := r.games[id]
	r.mu.RUnlock()
	return s, ok
}

func (r *Registry) remove(id string) {
	r.mu.Lock()
	delete(r.games, id)
	r.mu.Unlock()
}

// lobbyIDs returns IDs of human games still waiting for a second player.
func (r *Registry) lobbyIDs() []string {
	// Snapshot the map under the read lock, then inspect each session outside it
	// to avoid holding r.mu while acquiring s.mu (would invert lock ordering).
	r.mu.RLock()
	type entry struct {
		id string
		s  *GameSession
	}
	snap := make([]entry, 0, len(r.games))
	for id, s := range r.games {
		snap = append(snap, entry{id, s})
	}
	r.mu.RUnlock()

	var ids []string
	for _, e := range snap {
		e.s.mu.Lock()
		inLobby := e.s.state == sessionLobby && !e.s.isBot
		e.s.mu.Unlock()
		if inLobby {
			ids = append(ids, e.id)
		}
	}
	return ids
}

// GameSession holds all mutable state for one live game.
type GameSession struct {
	mu      sync.Mutex
	id      string
	game    *models.UnoChessGame
	tokens  map[string]chess.Color     // opaque token → player color
	conns   map[chess.Color]*playerConn // live WebSocket connections (nil if not connected)
	state   sessionState
	isBot   bool
	gcTimer *time.Timer
}

// playerConn is the send channel + cancel handle for one WebSocket client.
// The WebSocket connection itself is owned by the playWS goroutine; playerConn
// is the interface between that goroutine and the rest of the session.
type playerConn struct {
	send   chan []byte       // buffered outbox; writer goroutine drains this
	cancel context.CancelFunc // cancels the writer context → closes the WebSocket
}

// enqueue tries to place data on the outbox; if the buffer is full it cancels
// the connection (slow client) rather than blocking the caller under the session lock.
func (pc *playerConn) enqueue(data []byte) {
	select {
	case pc.send <- data:
	default:
		pc.cancel()
	}
}
