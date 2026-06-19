# Chess + Uno Flow: Architecture & Integration Plan

**Status:** Proposed — awaiting sign-off before implementation
**Date:** 2026-05-27
**Scope:** How the Uno card track and the Chess board track come together into one playable turn engine, and the phased plan to build it.

---

## 0. The question this document answers

The master game-state struct (`UnoChessGame`) lives in package `models`. All the move/turn logic lives in package `gameloop`. That split feels odd coming from an OOP mindset where behavior hangs off the object (`game.PlayCard(...)`). This document explains **why** the separation exists, **why it is not a workaround but the intended design**, **how the two halves integrate without merging**, and **the concrete plan** to wire up the full flow.

---

## 1. The tension

In classic OOP you would write methods on the struct: `game.PlayCard(c)`, `game.MakeMove(m)`. We cannot, and the reason is concrete rather than stylistic:

> When we tried to make `PlaySubMove` a method on `models.UnoChessGame`, it would have required `models` to call into `gameloop`. But `gameloop` already imports `models`. That is an **import cycle**, which Go forbids at compile time.

So "methods on the struct" is simply not an option for any behavior that depends on the rules engine or orchestration. The alternative — free functions that take `*models.UnoChessGame` — is the idiomatic Go answer, and the rest of this document shows it is coherent.

---

## 2. The layering and why dependencies point inward

```
models   (nouns / state)          imports: chess only
  ▲
core     (rules / validation)     imports: models
  ▲
gameloop (verbs / orchestration)  imports: core, models
  ▲
main / future api / storage       imports: gameloop, ...
```

Dependencies point **inward, toward the data**. `models` is the stable centre and depends on nothing but the third-party `chess` library. This is deliberate:

- **`models` cannot hold logic precisely because everything depends on it.** If the rules engine lived on the struct, every consumer of the data — a JSON API encoder, a database persistence layer, a replay tool — would transitively pull in the entire game engine just to *describe* a game.
- **The dependency direction is the architecture.** `models` knows nothing about `core` or `gameloop`. `core` knows nothing about `gameloop`. Each inner layer is reusable and testable without the outer ones.

This is the standard "stable dependencies" principle: volatile, high-level orchestration depends on stable, low-level data — never the reverse.

### Verified import graph (as of writing)

- `models`: imports `github.com/notnil/chess` only.
- `core`: imports `models`.
- `gameloop`: imports `core` and `models`.
- `main`: imports `gameloop`.

---

## 3. The pattern: state is passive, logic operates on it

The design is **data/behavior separation** (sometimes called a "procedural core" or, pejoratively but not always wrongly, an "anemic model"):

- **State is a passive record.** `UnoChessGame` is a bag of fields describing the game right now.
- **Logic is functions that take `*models.UnoChessGame` and mutate it.** `StartChessCombo(g, ...)`, `PlaySubMove(g, ...)`, `PlayResurrection(g, ...)` are already exactly this shape. This is not a workaround — it is the chosen pattern.

### The honest trade-off

The model's invariants cannot be protected *by the model itself* — any package can set its exported fields directly. We accept this. A wrapper type would not fix it either (see §5), because the fields remain exported in another package. The discipline lives in keeping all mutation flowing through the `gameloop` handler functions, by convention and code review, not by language enforcement.

---

## 4. The key insight: the model is the integration point

This is the crux of "how does it come together."

The two subsystems already exist and **never call each other**:

- **Chess half** — `backend/gameloop/chess_turn.go`, `backend/gameloop/resurrection.go`. Reads/writes `g.ChessEngine`, `g.Pending`, `g.Captured`, `g.History`.
- **Uno half** — `backend/gameloop/uno_loop.go`, `backend/gameloop/uno_utils.go`. Hands, draw/discard piles, card matching, choosers.

They meet **only in the shared `UnoChessGame` struct**. The chess combo logic does not know Uno exists; the Uno card logic does not know about checkmate. They both just transform fields on the same record.

> **Low coupling looks like this:** subsystems share *data definitions*, not *function calls*. The only component that knows the *sequence* (play a card → it is a 4 → run a 4-move combo → check for a win → pass the turn) is a thin **orchestrator** layered on top.

"Putting it together" is therefore **not** merging packages. It is writing one turn engine in `gameloop` that owns a `*models.UnoChessGame` and dispatches each played card to a primitive that already exists:

| Card played        | Chess-track action                                  |
|--------------------|-----------------------------------------------------|
| Number (0–9)       | `StartChessCombo` then N × `PlaySubMove`            |
| Draw Two (+2)      | `PlayResurrection` (2 pieces) + recolor flow        |
| Wild Draw Four (+4)| `PlayResurrection` (4 pieces) + recolor active color|
| Skip               | Turn-flow effect; no board change                  |
| Reverse            | Turn-flow effect (acts as Skip in 2-player)        |
| Wild (plain)       | Recolor active color; no board change              |

---

## 5. Orchestration decision: free functions as request-handlers

**Decision: free functions in `gameloop` operating on `*models.UnoChessGame`, organized as request-handlers — NOT an `Engine`/`Game` wrapper type.**

Reasons, in priority order:

1. **It fits the staggered transport model we already committed to.** A turn is *not* atomic. A "4" card is one `PlayCard` request plus four separate `PlaySubMove` requests arriving over time (e.g. over HTTP/WebSocket from a frontend). A +4 is a `PlayCard` plus a `PlayResurrection`. An `Engine` that "owns input sources" implies a long-lived loop *pulling* moves — exactly the blocking model we rejected earlier. With request-handlers, **the inbound request is the input**, and the resumable state lives in the struct.

2. **Consistency.** `PlaySubMove` and `PlayResurrection` already are free functions of this shape. A wrapper would make them the odd ones out or force a rewrite.

3. **A wrapper cannot truly encapsulate here.** `UnoChessGame`'s fields are exported and live in another package. A `gameloop.Game` around it gives method *syntax* but zero invariant protection — callers can still reach `g.Inner.ChessEngine`. The encapsulation would be cosmetic.

4. **YAGNI.** If a consumer later wants method ergonomics, a thin `Game` facade can wrap these functions without touching the core. Cheap to add later; expensive to unwind if we guess wrong now.

**Borrowed from the wrapper instinct:** group all turn handlers in one file (`backend/gameloop/turn.go`) with a coherent naming scheme (`PlayCard`, `PlaySubMove`, `PlayResurrection`, `DrawForTurn`, `AdvanceTurn`) so they read as one cohesive API even though they are package functions.

### Why the state machine lives in the struct

Because input is staggered, every handler must be callable "fresh" and pick up where the turn left off. So the turn's progress is stored on the model:

- `Pending *ActiveCombo` (already exists) tracks an in-progress number-card combo.
- A new `TurnPhase` marker (see §7) tracks whether we are awaiting a card, mid-combo, awaiting resurrection placements, or done.

This keeps handlers thin and resumable — the exact property a request/response transport needs.

---

## 6. Locked design decisions

These were decided and are not open for re-litigation unless explicitly revisited:

1. **`Deck` moves into `models`.** The `Deck` type (`[]UnoCard`) and its *pure data methods* (`Shuffle`, `DealStartingUnoCards`, `RemoveCard`, `CheckGameWon`, `ShouldShoutUno`, `PrintDeck`) belong in `models`, so `UnoChessGame` fields can be typed `Deck` and self-describing. **Strategy/bot logic stays in `gameloop`** (`ChooseMove`, `ChooseWildColor`, `moveImpact`). I/O (the `fmt.Println` messaging in draw/reshuffle) must NOT move into `models`; pile-flow helpers that print stay in `gameloop` or are split so `models` stays I/O-free.

2. **Two players, keyed by `chess.Color`.** The rulebook (`docs/rules.md`) describes a strictly 2-player White-vs-Black game (Reverse even degenerates to Skip). So `chess.Color` *is* the player identity. We drop the generic n-player seat arithmetic in `InitUnoGame`. This aligns with the model's existing `Hands map[chess.Color][]UnoCard`.

3. **Uno "active color" = the top discard card.** Represented as `DiscardPile[last]`; a wild / +4 mutates that card's `Color`. No new field. (Revisit only if it proves awkward.)

---

## 7. Target state shape

`UnoChessGame` becomes the single source of truth for everything. Current fields plus the additions:

```go
type UnoChessGame struct {
    // Chess track
    ChessEngine *chess.Game              // authoritative board (between combos)
    Pending     *ActiveCombo             // in-progress number-card combo, or nil
    Captured    map[chess.Color][]chess.PieceType // resurrection pool, per loser

    // Uno track
    Hands       map[chess.Color]Deck     // retyped to Deck (was []UnoCard)
    DrawPile    Deck                     // retyped
    DiscardPile Deck                     // retyped; top card = DiscardPile[last]

    // Turn management
    ActiveColor   chess.Color            // whose turn (White / Black)
    PlayDirection int                    // vestigial in 2-player; kept for clarity
    Phase         TurnPhase              // NEW: where we are within the active turn

    // Shared
    History []TurnRecord
}
```

### Turn phase state machine

```
AwaitingCard
   │  PlayCard(number)            PlayCard(+2/+4)          PlayCard(skip/rev/wild)
   ▼                                   │                        │
InCombo  ──PlaySubMove×N──►        AwaitingResurrection         │ (apply effect)
   │  (combo done /                    │  PlayResurrection       │
   │   checkmate intercept)            ▼                        ▼
   └──────────────►  AdvanceTurn  ◄────┴────────────────────────┘
                          │
                          ▼
                    AwaitingCard (next player)  OR  GameOver
```

`PlayCard` validates the card against the top of the discard pile (via `core.IsValidUnoMove`), removes it from the hand, pushes it to the discard pile, then transitions `Phase` based on card type and invokes the matching primitive.

---

## 8. Current-state inventory (gaps to close)

What exists and works:
- Chess combo primitives: `StartChessCombo`, `PlaySubMove`, `ApplyChessSubMove`, `MoveByUCI`, `PlayConsecutiveChessMoves` — all tested.
- Resurrection: `PlayResurrection`, `ResurrectionCount`, capture tracking via `recordCaptures` — all tested.
- A standalone bot-driven **Uno-only** loop: `InitUnoGame` — works, but disconnected from chess.
- Deck mechanics and choosers in `gameloop`.

What is missing or mismatched:
1. **`InitUnoGame` does not use `UnoChessGame`.** It runs on local variables and integer seat indices; it never touches a chess board.
2. **No chess board is created** anywhere in the Uno flow.
3. **Seat-index vs. chess-color mismatch.** Resolved by decision §6.2 (2 players by color).
4. **No turn-phase state** on the model yet.
5. **Active Uno color** has no representation. Resolved by decision §6.3.
6. **Win conditions span both tracks** and are not checked together anywhere yet:
   - Uno victory: a player's hand reaches 0 cards (rulebook §4.2).
   - Chess victory: checkmate or king capture (rulebook §4.1).

---

## 9. Phased implementation plan

### Phase 1 — Foundation (low-risk, mechanical)
- Move `Deck` + pure data methods into `models`; leave choosers/strategy in `gameloop`.
- Keep `models` I/O-free (strip or relocate `fmt.Println` from any moved helper).
- Retype `UnoChessGame` fields: `Hands map[chess.Color]Deck`, `DrawPile Deck`, `DiscardPile Deck`.
- Add `NewUnoChessGame()` constructor in `gameloop`: shuffles a full deck, deals 7 to White and 7 to Black, seeds the discard pile with one card, sets `ActiveColor = White`, `PlayDirection = 1`, `Phase = AwaitingCard`, `ChessEngine = chess.NewGame()`, initializes empty `History`/`Captured`/`Pending`.
- Adapt or set aside the existing `InitUnoGame` so the package still builds and all current tests stay green.
- **Exit criteria:** `go build ./... && go test ./...` green; new constructor unit-tested.

### Phase 2 — Orchestrator core
- Add `TurnPhase` type + `Phase` field to the model.
- Implement turn handlers in `backend/gameloop/turn.go`:
  - `PlayCard(g, card)` — validate against discard top, move to discard, dispatch by card type, set `Phase`.
  - `DrawForTurn(g)` — draw when no playable card (reuse existing draw/reshuffle helpers).
  - `AdvanceTurn(g)` — flip `ActiveColor` (honoring Skip/Reverse), reset `Phase`, run win checks.
  - Recolor logic for wild / +4 (mutate discard-top color).
- Dual-track win detection helper: Uno (empty hand) + Chess (`g.ChessEngine.Outcome()` / king capture).
- **Exit criteria:** unit tests per handler and per card type, including phase transitions and the checkmate-intercept path.

### Phase 3 — Prove the integration ✅ delivered

What shipped (see ticket UC-3 for the AC framing):

- **`RunGame(g, opts)`** in `backend/gameloop/gamerunner.go` — pure orchestration over the Phase-2 handlers. Returns a `GameResult{Winner, Reason, Turns}`.
- **`RunOptions`** carries `ChessMoveChooser`, `TurnCap` (default `10_000`), an `OnTurnEnd` hook for per-turn invariant checking, and a reserved `RandomSourcer` slot.
- **Four terminal reasons** via the `GameWinningReason` enum: `UnoWin`, `CheckmateWin`, `Draw`, `TurnCapHit`. The turn-cap case is distinct from a genuine no-progress draw.
- **AC-3 wild opening discard** resolved by a new `DeclareStartingColor` handler (Q-1 choice (b) — rule-faithful); `RunGame` calls it automatically at setup using `ChooseWildColor`.
- **AC-4 bot chess strategy** is `PreferCapturesAndChecks` (Q-2 choice (a)) — capture > check > first-legal — exposed as a chooser and injectable via `RunOptions`.
- **AC-5 determinism** via `NewUnoChessGameWith(rng *rand.Rand)` plus `Deck.ShuffleWith` / `DealStartingUnoCardsWith`. Same seed → byte-identical `GameResult` and `History`. The `RandomSourcer` field on `RunOptions` is currently unused — determinism flows in at construction since the deck shuffle/deal are the only random ops.
- **AC-7 / AC-8 integration test** `TestPhase3_BotGameInvariants` runs four seeded games to terminal and the `OnTurnEnd` hook asserts at every turn: 104-card conservation, valid `Phase` enum, boundary-state invariants (`Phase ∈ {AwaitingCard, GameOver}`, `Pending == nil`), `ActiveColor` validity, and parseability of every FEN ever written to `History`. Terminal invariants verify `Winner ↔ Reason` coherence (UnoWin → winner's hand empty, CheckmateWin → engine outcome agrees, Draw/TurnCapHit → `NoColor`). Runs in ~90ms.
- **DoD cleanup:** `InitUnoGame` deleted from `uno_loop.go`; `main.go` rewired to construct a seeded game via `NewUnoChessGameWith` and run it through `RunGame`, printing the seed for reproducibility.

Known carry-over:

- The legacy seat helpers `DealAllPlayerHands` and `wrapSeat` remain — they're now unused by gameplay but still referenced by their own unit tests (`TestWrapSeat`, deck tests). Safe to remove in a small follow-up if desired.
- Engine PGN/threefold/50-move history is lost across combo/resurrection re-syncs (documented in `chess_turn.go`). Per-turn board history lives in `g.History` instead.

### Phase 4 — Transport (later, separate sign-off)
- HTTP/WebSocket layer that calls the very same handlers. Each inbound move/card frame is one validated handler call. Per-client state visibility (own hand + public state only). Not in scope for this plan beyond noting that Phases 1–3 are designed to make it a thin addition.

---

## 10. Testing strategy

- **Unit, per handler** — `PlayCard` for each card type; `AdvanceTurn`; recolor; win detection. Use fixed FENs and constructed `UnoChessGame` values (as the existing chess tests do).
- **Property/invariant** — after any handler: total cards conserved (hands + draw + discard), `Phase` always valid, `ActiveColor` ∈ {White, Black}.
- **Integration** — seeded full game in Phase 3.
- **Determinism** — thread a seeded RNG so games are reproducible in CI.

---

## 11. Open questions / future work

- **Chess move strategy for bots.** `FirstLegalMove` produces silly games (rook shuffles). Phase 3 may want a slightly smarter chooser (prefer captures/checks) purely for demonstrative games. Not required for correctness.
- **Engine history loss.** Re-syncing `ChessEngine` from FEN after a combo/resurrection drops PGN and threefold/50-move counters (documented in `chess_turn.go`). If draw-by-repetition matters later, we will need to track it separately.
- **`PlayDirection`** is vestigial in a 2-player game. Kept for clarity; could be removed.
- **Human input seam.** The `ChessMoveChooser` interface and `MoveByUCI` already provide the seam; Phase 4 routes real requests through them.

---

## 12. Summary

- The data/logic split is intentional and enforced by Go's import rules: **state in `models`, rules in `core`, orchestration in `gameloop`.**
- The two game tracks integrate by **sharing the `UnoChessGame` record, not by calling each other.** A thin orchestrator sequences them.
- Orchestration is **free functions as request-handlers**, with the turn state machine stored on the model so the flow is resumable for staggered (frontend) input.
- We build it in phases: foundation → orchestrator → integration proof → transport.
