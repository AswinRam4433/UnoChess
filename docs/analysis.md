# UnoChess Codebase Analysis
**Date:** 2026-06-19  
**Status:** Backend complete through Phase 3. No frontend. No transport layer.

---

## 1. What Exists

### Architecture (3-layer Go backend)
```
models   — data types: UnoCard, Deck, UnoChessGame, TurnPhase, ActiveCombo, TurnRecord
core     — stateless rule functions: IsValidUnoMove, GetValidUnoMoves, ForceTurnInFEN
gameloop — orchestration: PlayCard, PlaySubMove, PlayResurrection, AdvanceTurn, RunGame
main.go  — CLI entry point: seeds a game, runs it with RunGame, prints result
```

All import cycles are correctly avoided. Only one external dependency: `github.com/notnil/chess v1.10.0`.

### What is done and tested
- Full Uno card validation (including +4 last-resort rule)
- Number-card chess combo mechanic (N consecutive moves, same color)
- Checkmate intercept (winning mid-combo ends the game instantly)
- Resurrection mechanic (+2/+4 bringing captured pieces back)
- Wild card color declaration, Skip/Reverse → PendingSkip in 2-player
- Draw pile reshuffle from discard when exhausted
- Stalemate draw detection (both piles exhausted, no progress for a full round)
- Deterministic seeded games via PCG RNG
- Turn-cap safety valve (10,000 turns default)
- Full bot-vs-bot runner (RunGame)
- Per-turn invariant tests: 104-card conservation, FEN validity, phase coherence
- All 3 packages pass `go test ./...`

---

## 2. Code Quality Findings

### Bugs and Logical Issues

**BUG — Reshuffle breaks determinism (gameloop/uno_loop.go:238)**  
`reshuffleDiscardIntoDraw` calls `drawPile.Shuffle()` which uses the global (non-seeded) RNG. When a reshuffle happens mid-game, a seeded game is no longer deterministic from that point forward. The test `TestRunGame_Deterministic` only catches this if a reshuffle happens in that specific seed's game — it may not be triggered. Fix: pass the seeded RNG through `drawCards` and `reshuffleDiscardIntoDraw`, or use the game struct to store it.

**BUG — `rand` v1 vs v2 mismatch (gameloop/gamerunner.go:7)**  
`gamerunner.go` imports `"math/rand"` (v1) for `RunOptions.RandomSourcer rand.Source`. All other files use `"math/rand/v2"`. The `RandomSourcer` field is currently unused, but any future wiring will conflict with the v2 types used everywhere else.

**LOGIC ISSUE — 0-card missing from deck**  
Standard Uno has 108 cards (one 0 per color). `buildFullUnoDeck` builds only 104 cards (1–9, two of each per color). The `u0` constant exists but is absent from both maps and the deck builder. The integration tests hardcode `const fullDeck = 104`. This is consistent internally but deviates from standard Uno without documentation.

**LOGIC ISSUE — "Moving Through Check" rule not fully enforced**  
The rulebook says a player can end a sub-move in self-check only if a subsequent sub-move in the same turn resolves it. The chess engine (`notnil/chess`) simply rejects any move leaving the king in check, enforced independently per sub-move. This makes the rule stricter than intended — you can never end any sub-move in check, even temporarily within a combo.

**LOGIC ISSUE — Chess engine PGN/50-move/threefold history lost on combo commit**  
After each number-card combo, `commitCombo` rebuilds the chess engine from FEN. This drops the move history, so 50-move rule and threefold repetition cannot be enforced by the engine. Only per-turn FENs in `g.History` survive. Acknowledged in code comments; acceptable for now but must be addressed before adding draw detection.

### Dead / Orphaned Code

**`InitUnoGame` in uno_loop.go (line 16)**  
Fully disconnected from `UnoChessGame`. Runs an old integer-seat bot loop with emoji prints. Not called from anywhere meaningful (main.go uses `RunGame`). Should be deleted. Its only callers are within `uno_loop.go` itself.

**`DealAllPlayerHands` and `wrapSeat` in uno_loop.go**  
Kept only to satisfy their own unit tests. Not used by the real game flow. Safe to remove along with `TestWrapSeat`.

**`PlayDirection` field on `UnoChessGame`**  
Set to `1` at construction, never mutated during play. The 2-player Reverse logic uses `PendingSkip` instead. Dead state field.

### I/O Contamination

**`fmt.Println` in `drawCards` / `reshuffleDiscardIntoDraw` (uno_loop.go:262-268)**  
These pure mechanics functions print directly to stdout ("♻️ Draw pile empty", "⚠️ Draw and discard piles both exhausted"). This makes the library noisy in bot games/tests and breaks testability. The I/O should live only in the transport layer or be injectable as a logger.

### Minor Issues

- `card value constants u0–u9` are unexported. Only `u0` is notable — it's a constant that can never be used outside the package and doesn't appear in any deck.
- `topOfDiscard` panics on empty `DiscardPile` (index out of range). The constructor always seeds it with one card and `drawCards` never consumes the top during reshuffle, so it's safe in practice but fragile.
- `chooseResurrections` (gamerunner.go:228) silently places fewer pieces than the card allows if the pool or empty squares run short. This is per-spec but not signaled to the caller — worth logging in a future transport layer.

---

## 3. Security Analysis

**Current surface: zero.** The codebase is a pure library + CLI. No HTTP, no WebSocket, no file I/O, no external calls, no user-facing input.

**Future transport layer will need:**

| Risk | Location | Mitigation needed |
|------|----------|-------------------|
| UCI string injection | PlaySubMove receives uci string from frontend | Already safe: MoveByUCI matches against legal-move set, never executes arbitrary strings |
| Game state ownership | Who owns which game? | Session tokens / auth before Phase 4 |
| Replay attacks | Sending the same move twice | Idempotent if Phase checked; handler rejects wrong-phase calls |
| DoS via long combos | Player plays card "9" (9 sub-moves) | Already bounded; TurnCap is the outer safety valve |
| Race conditions | Two concurrent requests mutating one game | Needs mutex or channel-per-game in the server layer |
| FEN injection | Resurrection produces FEN string | `chess.FEN()` validates it before accepting; safe |

---

## 4. Deployment State

**No deployment infrastructure exists.** There is:
- No HTTP server
- No WebSocket
- No Dockerfile
- No CI/CD config
- No build scripts
- No environment configuration

The only runnable artifact is `go run backend/main.go` which prints a bot game to stdout.

---

## 5. What the Next Step Is

### Immediate: Phase 4 — HTTP/WebSocket Transport Layer

The plan doc explicitly names this as next. The handlers are ready; they just need a server wrapper. Minimum viable:

1. **Game lifecycle endpoints**
   - `POST /game` → creates a new game, returns game ID + initial state
   - `GET /game/:id` → current state (filtered by player: own hand + public state)

2. **Turn action endpoints** (map 1:1 to existing handlers)
   - `POST /game/:id/card` → `PlayCard(g, card, declaredColor)`
   - `POST /game/:id/move` → `PlaySubMove(g, uci)`
   - `POST /game/:id/resurrection` → `PlayResurrection(g, card, placements)`
   - `POST /game/:id/draw` → `DrawForTurn(g)`
   - `POST /game/:id/color` → `DeclareStartingColor(g, color)`

3. **WebSocket** (optional but better UX)
   - Push board updates to both clients after each sub-move
   - Eliminates polling

4. **State serialization**
   - `UnoChessGame` → JSON response struct (filtered per player)
   - FEN strings → already JSON-friendly

5. **Session management**
   - Minimal: game-ID → game map in memory
   - Player tokens so White can't submit Black's moves

### After transport: Frontend
The frontend depends on the transport being defined first (needs API contract). Once the API exists:
- Chess board display (FEN rendering)
- Uno hand display (card images/components)
- Turn phase awareness (what action is expected next)
- WebSocket listener for live updates

---

## 6. Summary Table

| Area | Status | Rating |
|------|--------|--------|
| Game rules fidelity | Complete, tested | Good |
| Architecture layering | Clean 3-layer | Good |
| Test coverage | All packages, invariant suite | Good |
| Determinism | Partial (reshuffle breaks it) | Bug |
| Dead code | `InitUnoGame`, `wrapSeat`, `PlayDirection` | Cleanup needed |
| I/O hygiene | stdout leaks in library code | Fix before transport |
| Transport layer | Absent | Not started |
| Frontend | Absent | Not started |
| Deployment | None | Not started |
| Security | N/A (no surface yet) | Pre-transport OK |
