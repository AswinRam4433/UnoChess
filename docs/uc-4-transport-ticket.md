# UC-4 · HTTP + WebSocket transport for live UnoChess games
**Type:** Story · **Epic:** Chess + Uno flow · **Estimate:** L · **Depends on:** Phases 1–3 (complete)

## Background
Phases 1–3 delivered a state machine that runs full games in-process with bots. The handlers (PlayCard, PlaySubMove, PlayResurrection, DrawForTurn, AdvanceTurn, DeclareStartingColor) are deliberately stateless-per-call and operate on `*models.UnoChessGame`, which is exactly the shape a transport layer needs — every inbound frame from the network is one validated handler call. Phase 4 wires that engine to a real network so two browsers can play one game, with each player seeing only their own hand and the public state.

## User story
As a player wanting to play UnoChess in a browser,
I want to create or join a game and play turns over a real-time connection,
so that I can compete against another human (or a server-side bot) without needing to share a process with them.

---

## AC-1 · HTTP control plane
A new `backend/transport` package exposes an HTTP server with this surface:

| Endpoint | Body / params | Response |
|----------|---------------|----------|
| `POST /games` | `{"opponent": "human"\|"bot"}` (default `"human"`) | `{gameID, playerToken, playerColor: "White"}` |
| `POST /games/{gameID}/join` | — | `{playerToken, playerColor: "Black"}` |
| `GET /games/{gameID}` | — | public metadata: `{state, players, winner?, reason?}` — no private hand info |
| `GET /games` | — | list of game IDs in state `"lobby"` |

- Game IDs are UUIDv4.
- Tokens are opaque strings (32 bytes, base64url) minted at join time. They authenticate one specific (game, color) pair.
- Game state is held in an in-memory registry behind a single `sync.RWMutex` for the catalog plus a per-game mutex (AC-6).
- Errors: 404 on unknown game, 409 if joining a full or finished game, 400 on malformed payload. JSON error bodies: `{code, message}`.

## AC-2 · WebSocket connection per player
- `WS /games/{gameID}/play?token={playerToken}` upgrades to a WebSocket.
- Token validation: unknown → close with 1008 (policy violation); unknown game → 1011 (server error, reason=game_not_found).
- Each (game, color) holds at most one live connection. A new connection from the same token replaces the previous one (the old socket is closed with code 4000 superseded). Reconnection-with-state across server restarts is out of scope.
- On connect: the server immediately sends one state snapshot (AC-4) filtered for that player.

## AC-3 · Inbound command messages
JSON frames, one validated handler call each:

```json
{"type": "play_card", "card": {"value": "3", "color": "Red"}, "declaredColor": "Red"}
{"type": "play_sub_move", "uci": "b1c3"}
{"type": "play_resurrection", "placements": [{"piece": "queen", "square": "e3"}]}
{"type": "draw_for_turn"}
{"type": "declare_starting_color", "color": "Red"}
```

Semantics:
- Each command serializes through the per-game mutex (AC-6).
- A command from the inactive color → error event `not_your_turn`; no state mutation.
- A handler error (illegal move, wrong phase, etc.) → error event to the sender only; no broadcast, no mutation.
- A successful command → a filtered state snapshot (AC-4) is broadcast to both connections (each its own view).
- `declaredColor` is required for wild plays; ignored for non-wild.

## AC-4 · Player-filtered state snapshot (outbound)
Sent to one player after each successful command and on (re)connect:

```json
{
  "type": "state",
  "phase": "AwaitingCard",
  "yourColor": "White",
  "activeColor": "White",
  "yourHand": [{"value": "5", "color": "Red"}, ...],
  "opponentHandCount": 6,
  "discardTop": {"value": "5", "color": "Red"},
  "drawPileSize": 75,
  "boardFEN": "rnbq...",
  "pendingCombo": {"movesRemaining": 2},
  "history": [...],
  "winner": null,
  "reason": null
}
```

**The opponent's hand contents are NEVER serialized for any client. Only `opponentHandCount`.** This is the single most important security property — enforce it at the serialization boundary with a test (AC-7).

## AC-5 · Outbound event types beyond state

```json
{"type": "error", "code": "illegal_card_play", "message": "..."}
{"type": "game_over", "winner": "White", "reason": "Uno", "turns": 47}
```

Error codes map 1:1 to gameloop sentinel errors via `errors.Is`. Full set:
`not_your_turn`, `not_awaiting_card`, `card_not_in_hand`, `illegal_card_play`, `invalid_wild_color`, `has_playable_card`, `turn_not_complete`, `not_start_of_game`, `no_wild_to_declare`, `illegal_sub_move`, `cannot_resurrect_king`, `square_not_own_half`, `square_occupied`, `piece_not_captured`, `too_many_resurrections`, `not_resurrection_card`, `game_over`.

A `game_over` event is emitted after the terminal-state snapshot.

## AC-6 · Concurrency safety
- Each game has its own `sync.Mutex`. All gameloop handler calls run under that mutex.
- Snapshot serialization reads under the mutex.
- WebSocket writes go through a per-connection buffered channel (depth ~8) so a slow client cannot block the game. If the buffer fills, the connection is closed.
- The catalog map (game-id → game) has its own `sync.RWMutex`.

## AC-7 · Tests

| Test | What it proves |
|------|---------------|
| httptest table tests for `POST /games`, `POST /join`, `GET /games`, `GET /games/{id}` | HTTP surface behavior + error codes |
| WebSocket integration: two clients, play ≥3 turns end-to-end | Round-trip works; both sides see consistent state |
| Opponent-hand-leak test | Drive a game past several turns; assert no JSON frame ever sent to either client contains the opponent's actual cards |
| Wrong-color command | A frame on Black's socket while it's White's turn → `not_your_turn`, no state change |
| Illegal command | Returns proper error code; state unchanged; opponent sees no broadcast |
| Reconnect mid-game | New connection on the same token gets current snapshot; old connection closed with 4000 |
| Bot opponent (if AC-8 chosen) | Create with `opponent=bot`, play one human turn, bot's reply lands |

## AC-8 · Bot opponent mode
`POST /games {"opponent": "bot"}` creates a game where Black is a server-side bot. After the human plays, a goroutine watches for `PhaseAwaitingCard` with `ActiveColor == Black` and drives the bot's turn through the same handlers (`runOneTurn + PreferCapturesAndChecks + ChooseMove`). The human's snapshot updates as if a remote player moved.

## AC-9 · Graceful shutdown
- SIGTERM / SIGINT close all WebSocket connections with 1001 (going away).
- The HTTP server stops accepting new connections.
- Active games are marked terminated; no goroutines leak.
- Verified by `go test -race` and a shutdown-while-active test.

---

## Out of scope
- Persistence — in-memory only; finished games GC after 30 minutes.
- Authentication beyond per-game tokens.
- Reconnection across server restart.
- Frontend / UI — only a minimal `backend/transport/devui/` HTML page for manual smoke testing.
- TLS — assume reverse proxy termination.
- Spectator mode, replay, metrics, rate limiting, horizontal scale.

---

## Technical notes
- **WebSocket library:** `coder/websocket` (formerly `nhooyr.io/websocket`) preferred over `gorilla/websocket` (maintenance mode).
- **Serialization boundary:** `PlayerView` (or similar) is a separate type from `models.UnoChessGame`. The conversion is the only place hand-filtering happens.
- **main.go:** Runs the HTTP+WS server on `-port` flag or `PORT` env (default 8080). Existing bot-game mode reachable via `-mode=bot`.
- **Logging:** Structured JSON logs, one line per command. Tokens never logged.
- **Tokens:** `crypto/rand` → 32 bytes → `base64.RawURLEncoding`. Constant-time compare via `subtle.ConstantTimeCompare`.
- **History:** `TurnRecord` (Player, CardPlayed, BoardStates) is public — safe to serialize as-is.

---

## Open questions (to decide before implementation)

| # | Question | Recommendation |
|---|----------|---------------|
| Q-1 | Transport protocol | **(a) WebSocket for everything** — commands and push share one connection |
| Q-2 | Session storage | **(a) In-memory only for MVP** — lost on restart |
| Q-3 | Player identity | **(a) Anonymous opaque token per game** — no accounts |
| Q-4 | Bot opponent mode (AC-8) | **(a) Include it** — one-browser dev experience is huge |
| Q-5 | WebSocket library | **(a) coder/websocket** — modern, context-aware API |

---

## Definition of done
- `backend/transport` package with all endpoints.
- All AC tests pass; `go build ./... && go vet ./... && go test ./... -count=1 -race` green.
- Opponent-hand-leak test passes.
- `main.go` runs HTTP+WS server; bot-game runner reachable via `-mode=bot`.
- `backend/transport/devui/index.html` for manual two-tab smoke testing.
- `docs/api.md` documents every message type and error code.
- `InitUnoGame` carry-over helpers (`DealAllPlayerHands`, `wrapSeat`, their tests) cleaned up.
