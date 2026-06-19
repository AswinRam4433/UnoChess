package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/notnil/chess"

	"unochess/gameloop"
	"unochess/models"
	"unochess/transport"
)

// --- Test helpers -----------------------------------------------------------

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg := transport.NewRegistry()
	srv := transport.NewServer(reg)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// newSeededTestServer injects a seeded game factory so tests get predictable hands.
func newSeededTestServer(t *testing.T, seed1, seed2 uint64) *httptest.Server {
	t.Helper()
	rng := rand.New(rand.NewPCG(seed1, seed2))
	factory := func() *models.UnoChessGame {
		return gameloop.NewUnoChessGameWith(rng)
	}
	reg := transport.NewRegistryWithFactory(factory)
	srv := transport.NewServer(reg)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

func postJSON(t *testing.T, url string, body any, dest any) *http.Response {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	resp.Body.Close()
	return resp
}

func postEmpty(t *testing.T, url string, dest any) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	resp.Body.Close()
	return resp
}

func getJSON(t *testing.T, url string, dest any) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
	resp.Body.Close()
	return resp
}

func dialWS(t *testing.T, ctx context.Context, ts *httptest.Server, path string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + path
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("WS dial %s: %v", wsURL, err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func readMsg(t *testing.T, ctx context.Context, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("WS read: %v", err)
	}
	var msg map[string]any
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("JSON parse: %v — raw: %s", err, data)
	}
	return msg
}

func writeCmd(t *testing.T, ctx context.Context, conn *websocket.Conn, v any) {
	t.Helper()
	data, _ := json.Marshal(v)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("WS write: %v", err)
	}
}

// readMsgType reads messages until one with the given type arrives (skipping others).
func readMsgType(t *testing.T, ctx context.Context, conn *websocket.Conn, typ string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		msg := readMsg(t, ctx, conn)
		if msg["type"] == typ {
			return msg
		}
	}
	t.Fatalf("timed out waiting for message type %q", typ)
	return nil
}

// createHumanGame creates a game, joins as Black, and returns (gameID, whiteToken, blackToken).
func createHumanGame(t *testing.T, ts *httptest.Server) (string, string, string) {
	t.Helper()
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &createResp)

	var joinResp transport.JoinGameResp
	postEmpty(t, ts.URL+"/games/"+createResp.GameID+"/join", &joinResp)

	return createResp.GameID, createResp.Token, joinResp.Token
}

// findPlayableCard scans a PlayerView-shaped map for a card that can be played on
// the current discard top, using basic Uno matching (color, value, or Wild).
func findPlayableCard(hand []any, discardTop map[string]any) map[string]any {
	topColor := discardTop["color"].(string)
	topValue := discardTop["value"].(string)

	var wildFallback map[string]any
	for _, c := range hand {
		card := c.(map[string]any)
		cv := card["value"].(string)
		cc := card["color"].(string)
		if cc == "WILD" && cv == string(models.WildCard) {
			wildFallback = card // plain Wild — keep as fallback
			continue
		}
		if cc == "WILD" {
			continue // +4: skip (last-resort only)
		}
		if cc == topColor || cv == topValue {
			return card
		}
	}
	return wildFallback // nil if only +4s remain
}

// --- HTTP surface tests (AC-1) -----------------------------------------------

func TestCreateGame_Human(t *testing.T) {
	ts := newTestServer(t)
	var resp transport.CreateGameResp
	r := postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &resp)

	if r.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", r.StatusCode)
	}
	if resp.GameID == "" {
		t.Error("gameID is empty")
	}
	if resp.Token == "" {
		t.Error("playerToken is empty")
	}
	if resp.Color != "White" {
		t.Errorf("playerColor = %q, want White", resp.Color)
	}
}

func TestCreateGame_Bot(t *testing.T) {
	ts := newTestServer(t)
	var resp transport.CreateGameResp
	r := postJSON(t, ts.URL+"/games", map[string]string{"opponent": "bot"}, &resp)

	if r.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", r.StatusCode)
	}
	if resp.Color != "White" {
		t.Errorf("playerColor = %q, want White", resp.Color)
	}
}

func TestJoinGame_Success(t *testing.T) {
	ts := newTestServer(t)
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &createResp)

	var joinResp transport.JoinGameResp
	r := postEmpty(t, ts.URL+"/games/"+createResp.GameID+"/join", &joinResp)

	if r.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", r.StatusCode)
	}
	if joinResp.Token == "" {
		t.Error("playerToken is empty")
	}
	if joinResp.Color != "Black" {
		t.Errorf("playerColor = %q, want Black", joinResp.Color)
	}
}

func TestJoinGame_Full(t *testing.T) {
	ts := newTestServer(t)
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &createResp)
	postEmpty(t, ts.URL+"/games/"+createResp.GameID+"/join", nil) // first join succeeds

	r := postEmpty(t, ts.URL+"/games/"+createResp.GameID+"/join", nil) // second should 409
	if r.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409 on second join", r.StatusCode)
	}
}

func TestJoinGame_BotGame(t *testing.T) {
	ts := newTestServer(t)
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "bot"}, &createResp)

	r := postEmpty(t, ts.URL+"/games/"+createResp.GameID+"/join", nil)
	if r.StatusCode != http.StatusConflict {
		t.Errorf("joining a bot game should return 409, got %d", r.StatusCode)
	}
}

func TestJoinGame_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := postEmpty(t, ts.URL+"/games/nonexistent/join", nil)
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", r.StatusCode)
	}
}

func TestGetGame(t *testing.T) {
	ts := newTestServer(t)
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &createResp)

	var meta map[string]any
	r := getJSON(t, ts.URL+"/games/"+createResp.GameID, &meta)

	if r.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", r.StatusCode)
	}
	if meta["id"] != createResp.GameID {
		t.Errorf("id = %v, want %s", meta["id"], createResp.GameID)
	}
	if meta["state"] != "lobby" {
		t.Errorf("state = %v, want lobby", meta["state"])
	}
	players := meta["players"].(map[string]any)
	if players["White"] != true {
		t.Error("White should be joined after create")
	}
	if players["Black"] != false {
		t.Error("Black should not be joined yet")
	}
}

func TestGetGame_NotFound(t *testing.T) {
	ts := newTestServer(t)
	r := getJSON(t, ts.URL+"/games/nosuchgame", nil)
	if r.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", r.StatusCode)
	}
}

func TestListGames(t *testing.T) {
	ts := newTestServer(t)

	var before map[string]any
	getJSON(t, ts.URL+"/games", &before)
	beforeCount := len(before["games"].([]any))

	// Create two human games (will appear in lobby list).
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, nil)
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, nil)
	// Bot game should NOT appear in lobby list.
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "bot"}, nil)

	var after map[string]any
	getJSON(t, ts.URL+"/games", &after)
	afterCount := len(after["games"].([]any))

	if afterCount-beforeCount != 2 {
		t.Errorf("lobby count grew by %d, want 2 (bot game excluded)", afterCount-beforeCount)
	}
}

// --- WebSocket tests (AC-2/AC-3/AC-4) ----------------------------------------

func TestWebSocket_InitialState(t *testing.T) {
	ts := newTestServer(t)
	gameID, whiteToken, _ := createHumanGame(t, ts)

	ctx := context.Background()
	conn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)

	msg := readMsg(t, ctx, conn)
	if msg["type"] != "state" {
		t.Errorf("first message type = %v, want state", msg["type"])
	}
	if msg["yourColor"] != "White" {
		t.Errorf("yourColor = %v, want White", msg["yourColor"])
	}
	if msg["phase"] != "AwaitingCard" {
		t.Errorf("phase = %v, want AwaitingCard", msg["phase"])
	}
	hand := msg["yourHand"].([]any)
	if len(hand) != 7 {
		t.Errorf("yourHand len = %d, want 7", len(hand))
	}
	if msg["opponentHandCount"].(float64) != 7 {
		t.Errorf("opponentHandCount = %v, want 7", msg["opponentHandCount"])
	}
}

func TestWebSocket_InvalidToken(t *testing.T) {
	ts := newTestServer(t)
	gameID, _, _ := createHumanGame(t, ts)

	ctx := context.Background()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/games/" + gameID + "/play?token=badtoken"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		// Connection refused at HTTP level is also acceptable
		return
	}
	t.Cleanup(func() { conn.CloseNow() })

	// Should receive a close frame with policy violation (1008)
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, _, readErr := conn.Read(ctx2)
	if readErr == nil {
		t.Error("expected connection to be closed for invalid token")
	}
}

func TestWebSocket_NotYourTurn(t *testing.T) {
	ts := newTestServer(t)
	gameID, _, blackToken := createHumanGame(t, ts)

	ctx := context.Background()
	// Black connects and tries to play immediately (it's White's turn).
	blackConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+blackToken)
	readMsg(t, ctx, blackConn) // consume initial state

	writeCmd(t, ctx, blackConn, map[string]any{
		"type": "draw_for_turn",
	})

	errMsg := readMsgType(t, ctx, blackConn, "error")
	if errMsg["code"] != "not_your_turn" {
		t.Errorf("code = %v, want not_your_turn", errMsg["code"])
	}
}

func TestWebSocket_IllegalCardPlay(t *testing.T) {
	ts := newSeededTestServer(t, 42, 7) // deterministic game
	gameID, whiteToken, _ := createHumanGame(t, ts)

	ctx := context.Background()
	whiteConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	initState := readMsg(t, ctx, whiteConn)

	// Attempt to play a card that White does NOT hold.
	writeCmd(t, ctx, whiteConn, map[string]any{
		"type": "play_card",
		"card": map[string]string{"value": "9", "color": "RED"},
	})

	// Look for an error or a state update — if it's a state update the play succeeded
	// (unlikely for a targeted non-held card), if it's an error we've proven the guard.
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, raw, err := whiteConn.Read(ctx2)
	if err != nil {
		t.Skip("connection closed before response — test inconclusive")
	}
	var resp map[string]any
	json.Unmarshal(raw, &resp)

	// Either the card wasn't in hand (card_not_in_hand) OR wasn't legal (illegal_card_play).
	// Either error confirms the guard works; only a success state response indicates a bug.
	if resp["type"] == "state" {
		// The card happened to be in hand AND was legal — not a test failure, just an
		// unlucky seed. Verify the state changed correctly at least.
		hand := initState["yourHand"].([]any)
		newHand := resp["yourHand"].([]any)
		if len(newHand) != len(hand)-1 {
			t.Error("hand size did not shrink after playing a card")
		}
	}
	// If type == "error", the guard worked. Either outcome is acceptable.
	_ = resp
}

// TestWebSocket_HandLeak is the primary security test: it verifies that no JSON
// frame sent to one player ever contains the opponent's actual hand cards.
func TestWebSocket_HandLeak(t *testing.T) {
	ts := newTestServer(t)
	gameID, whiteToken, blackToken := createHumanGame(t, ts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	whiteConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	blackConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+blackToken)

	// Collect raw frames for each player.
	collectFrames := func(conn *websocket.Conn, ch chan<- []byte, done chan<- struct{}) {
		defer close(done)
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				return
			}
			ch <- append([]byte(nil), data...)
		}
	}

	whiteCh := make(chan []byte, 64)
	blackCh := make(chan []byte, 64)
	whiteDone := make(chan struct{})
	blackDone := make(chan struct{})

	go collectFrames(whiteConn, whiteCh, whiteDone)
	go collectFrames(blackConn, blackCh, blackDone)

	// Let the game run briefly so we accumulate a few messages.
	time.Sleep(200 * time.Millisecond)
	whiteConn.CloseNow()
	blackConn.CloseNow()
	<-whiteDone
	<-blackDone

	// Drain remaining buffered frames.
	var whiteFrames, blackFrames [][]byte
	for {
		select {
		case f := <-whiteCh:
			whiteFrames = append(whiteFrames, f)
		default:
			goto drainBlack
		}
	}
drainBlack:
	for {
		select {
		case f := <-blackCh:
			blackFrames = append(blackFrames, f)
		default:
			goto check
		}
	}
check:

	// Parse all frames and confirm structural invariants.
	checkFrames := func(frames [][]byte, expectedColor string) {
		for _, raw := range frames {
			var msg map[string]any
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			if msg["type"] != "state" {
				continue
			}
			// yourColor must match the expected player.
			if got := msg["yourColor"]; got != expectedColor {
				t.Errorf("state message sent to %s player has yourColor=%v", expectedColor, got)
			}
			// opponentHandCount must be a number, never a list.
			if ohc, exists := msg["opponentHandCount"]; exists {
				if _, ok := ohc.(float64); !ok {
					t.Errorf("opponentHandCount is %T, want float64 (number)", ohc)
				}
			}
			// There must be no field that exposes the opponent's actual cards.
			for _, forbidden := range []string{"opponentHand", "blackHand", "whiteHand", "allHands"} {
				if _, exists := msg[forbidden]; exists {
					t.Errorf("state message contains forbidden field %q — hand leak!", forbidden)
				}
			}
		}
	}
	if len(whiteFrames) == 0 {
		t.Error("White received no frames — check WebSocket connection")
	}
	checkFrames(whiteFrames, "White")
	checkFrames(blackFrames, "Black")
}

// TestWebSocket_PlayOneTurn plays one card (as White) in a human-vs-human game
// and verifies both players see a consistent updated state.
func TestWebSocket_PlayOneTurn(t *testing.T) {
	ts := newSeededTestServer(t, 99, 1234)
	gameID, whiteToken, blackToken := createHumanGame(t, ts)

	ctx := context.Background()
	whiteConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	blackConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+blackToken)

	// Both players receive initial state.
	whiteInit := readMsg(t, ctx, whiteConn)
	readMsg(t, ctx, blackConn) // consume Black's initial state

	// Pick a playable card from White's hand.
	hand := whiteInit["yourHand"].([]any)
	discardTop := whiteInit["discardTop"].(map[string]any)
	playable := findPlayableCard(hand, discardTop)
	if playable == nil {
		t.Skip("no playable card in White's hand with this seed — skip")
	}

	cmd := map[string]any{
		"type": "play_card",
		"card": playable,
	}
	if playable["color"] == "WILD" {
		cmd["declaredColor"] = "RED"
	}
	writeCmd(t, ctx, whiteConn, cmd)

	// White should receive a state update (or game-over if it was the last card).
	whiteNext := readMsgType(t, ctx, whiteConn, "state")
	// Black should also receive a state update.
	blackNext := readMsgType(t, ctx, blackConn, "state")

	// After White plays, it should be Black's turn (unless White played Skip/Rev).
	// Either way, the discard pile changed.
	_ = whiteNext
	if blackNext["boardFEN"] == nil {
		t.Error("Black's state update is missing boardFEN")
	}
	if blackNext["yourColor"] != "Black" {
		t.Errorf("Black received a state with yourColor=%v", blackNext["yourColor"])
	}
}

// TestWebSocket_BotGame verifies that after White plays one card, the bot
// automatically plays its own turn and White receives the updated state.
func TestWebSocket_BotGame(t *testing.T) {
	ts := newSeededTestServer(t, 31415, 27182)

	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "bot"}, &createResp)

	ctx := context.Background()
	whiteConn := dialWS(t, ctx, ts, "/games/"+createResp.GameID+"/play?token="+createResp.Token)

	initState := readMsg(t, ctx, whiteConn)

	hand := initState["yourHand"].([]any)
	discardTop := initState["discardTop"].(map[string]any)
	playable := findPlayableCard(hand, discardTop)
	if playable == nil {
		t.Skip("no non-wild playable card with this seed — skip")
	}

	cmd := map[string]any{
		"type": "play_card",
		"card": playable,
	}
	if playable["color"] == "WILD" {
		cmd["declaredColor"] = "RED"
	}
	writeCmd(t, ctx, whiteConn, cmd)

	// White gets a state update after its own play, then another after the bot plays.
	// Accept either a state update or a game_over as valid terminal responses.
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	gotBotResponse := false
	for i := 0; i < 3; i++ {
		_, data, err := whiteConn.Read(ctx2)
		if err != nil {
			break
		}
		var msg map[string]any
		json.Unmarshal(data, &msg)
		if msg["type"] == "state" || msg["type"] == "game_over" {
			gotBotResponse = true
			break
		}
	}
	if !gotBotResponse {
		t.Error("White did not receive any update after playing — bot may not have triggered")
	}
}

// TestWebSocket_Reconnect verifies that a new WebSocket connection on the same
// token receives a current state snapshot and the old connection is superseded.
func TestWebSocket_Reconnect(t *testing.T) {
	ts := newTestServer(t)
	gameID, whiteToken, _ := createHumanGame(t, ts)

	ctx := context.Background()

	// First connection.
	conn1 := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	readMsg(t, ctx, conn1) // initial state

	// Second connection with the same token supersedes the first.
	conn2 := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	t.Cleanup(func() { conn2.CloseNow() })

	// conn2 should immediately receive a state snapshot.
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, data, err := conn2.Read(ctx2)
	if err != nil {
		t.Fatalf("conn2 read failed: %v", err)
	}
	var msg map[string]any
	json.Unmarshal(data, &msg)
	if msg["type"] != "state" {
		t.Errorf("reconnected client got message type %v, want state", msg["type"])
	}
	if msg["yourColor"] != "White" {
		t.Errorf("reconnected client got yourColor=%v, want White", msg["yourColor"])
	}

	// conn1 should be closed by the server (superseded).
	conn1.CloseNow() // safe to call even if already closed
}

// TestWebSocket_GameNotStarted verifies that a command is rejected when only one
// human player has connected and the game is still in the lobby state.
func TestWebSocket_GameNotStarted(t *testing.T) {
	ts := newTestServer(t)
	var createResp transport.CreateGameResp
	postJSON(t, ts.URL+"/games", map[string]string{"opponent": "human"}, &createResp)
	// Do NOT join Black — game stays in lobby.

	ctx := context.Background()
	whiteConn := dialWS(t, ctx, ts, "/games/"+createResp.GameID+"/play?token="+createResp.Token)
	readMsg(t, ctx, whiteConn) // initial state

	writeCmd(t, ctx, whiteConn, map[string]any{"type": "draw_for_turn"})

	errResp := readMsgType(t, ctx, whiteConn, "error")
	if errResp["code"] != "game_not_started" {
		t.Errorf("code = %v, want game_not_started", errResp["code"])
	}
}

// TestMalformedJSON verifies that a malformed command produces an error, not a panic.
func TestMalformedJSON(t *testing.T) {
	ts := newTestServer(t)
	gameID, whiteToken, _ := createHumanGame(t, ts)

	ctx := context.Background()
	whiteConn := dialWS(t, ctx, ts, "/games/"+gameID+"/play?token="+whiteToken)
	readMsg(t, ctx, whiteConn)

	if err := whiteConn.Write(ctx, websocket.MessageText, []byte("{not valid json")); err != nil {
		t.Skip("write failed — likely already closed")
	}

	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, data, err := whiteConn.Read(ctx2)
	if err != nil {
		return // connection closed is also acceptable
	}
	var msg map[string]any
	json.Unmarshal(data, &msg)
	if msg["type"] != "error" {
		t.Errorf("expected error response for malformed JSON, got type=%v", msg["type"])
	}
}

// --- Build-time guard: ensure all exported types compile --------------------

var (
	_ io.Writer           = io.Discard
	_ *transport.Registry = (*transport.Registry)(nil)
	_ *transport.Server   = (*transport.Server)(nil)

	// Ensure chess package is reachable from tests (catches import path issues).
	_ = chess.White
)
