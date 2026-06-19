package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/notnil/chess"

	"unochess/gameloop"
	"unochess/models"
)

// --- Inbound message types --------------------------------------------------

type inboundMsg struct {
	Type          string         `json:"type"`
	Card          *cardMsg       `json:"card,omitempty"`
	DeclaredColor string         `json:"declaredColor,omitempty"`
	UCI           string         `json:"uci,omitempty"`
	Placements    []placementMsg `json:"placements,omitempty"`
	Color         string         `json:"color,omitempty"`
}

type cardMsg struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

type placementMsg struct {
	Piece  string `json:"piece"`
	Square string `json:"square"`
}

// --- WebSocket handler ------------------------------------------------------

func (s *Server) playWS(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameID")
	token := r.URL.Query().Get("token")

	// Upgrade first — WebSocket errors must go through the WS close handshake, not
	// plain HTTP responses, once the upgrade is initiated.
	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // allow all origins; restrict per-origin in production
	})
	if err != nil {
		return
	}

	session, ok := s.reg.get(gameID)
	if !ok {
		wsConn.Close(websocket.StatusInternalError, "game_not_found")
		return
	}

	// Validate token with constant-time comparison.
	session.mu.Lock()
	var color chess.Color
	var found bool
	for tok, col := range session.tokens {
		if tokenEqual(tok, token) {
			color = col
			found = true
			break
		}
	}
	session.mu.Unlock()

	if !found {
		wsConn.Close(websocket.StatusPolicyViolation, "invalid token")
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	pc := &playerConn{
		send:   make(chan []byte, sendDepth),
		cancel: cancel,
	}

	// Replace any previous connection for this color (superseded reconnect).
	session.mu.Lock()
	if old := session.conns[color]; old != nil {
		old.cancel()
	}
	session.conns[color] = pc
	initialData, _ := json.Marshal(buildPlayerView(session, color))
	session.mu.Unlock()

	// Writer goroutine: drains pc.send and writes to the WebSocket.
	go func() {
		defer wsConn.Close(websocket.StatusNormalClosure, "")
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-pc.send:
				if !ok {
					return
				}
				if err := wsConn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}
			}
		}
	}()

	// Send the initial state snapshot immediately after connecting.
	pc.send <- initialData

	// Reader loop: parse and dispatch commands until the connection closes.
	defer func() {
		cancel()
		session.mu.Lock()
		if session.conns[color] == pc {
			delete(session.conns, color)
		}
		session.mu.Unlock()
	}()

	for {
		_, raw, err := wsConn.Read(ctx)
		if err != nil {
			break
		}
		dispatchCommand(session, color, pc, raw)
	}
}

// --- Command dispatch -------------------------------------------------------

func dispatchCommand(s *GameSession, color chess.Color, sender *playerConn, raw []byte) {
	var msg inboundMsg
	if err := json.Unmarshal(raw, &msg); err != nil {
		sendErr(sender, "invalid_message", "malformed JSON")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == sessionFinished || s.game.Phase == models.PhaseGameOver {
		sendErr(sender, "game_over", "the game has ended")
		return
	}
	if s.state == sessionLobby {
		sendErr(sender, "game_not_started", "waiting for opponent to join")
		return
	}

	// Active-color guard (not_your_turn). DeclareStartingColor is also restricted
	// to the active (White) player via this same check.
	if color != s.game.ActiveColor {
		sendErr(sender, "not_your_turn", fmt.Sprintf("it is %s's turn", colorName(s.game.ActiveColor)))
		return
	}

	var cmdErr error
	switch msg.Type {
	case "play_card":
		cmdErr = handlePlayCard(s, msg)
	case "play_sub_move":
		cmdErr = handlePlaySubMove(s, msg)
	case "play_resurrection":
		cmdErr = handlePlayResurrection(s, msg)
	case "draw_for_turn":
		cmdErr = handleDrawForTurn(s)
	case "declare_starting_color":
		cmdErr = gameloop.DeclareStartingColor(s.game, models.CardColor(msg.Color))
	default:
		sendErr(sender, "unknown_command", fmt.Sprintf("unknown type %q", msg.Type))
		return
	}

	if cmdErr != nil {
		sendErr(sender, mapErrCode(cmdErr), cmdErr.Error())
		return
	}

	// Successful command: push filtered snapshots to every connected client.
	s.broadcastLocked()

	// Trigger the bot if it is now Black's turn in a bot game.
	if s.isBot && s.game.Phase == models.PhaseAwaitingCard && s.game.ActiveColor == chess.Black {
		go s.runBotTurn()
	}
}

// --- Per-command handlers ---------------------------------------------------

func handlePlayCard(s *GameSession, msg inboundMsg) error {
	if msg.Card == nil {
		return fmt.Errorf("%w: missing card field", gameloop.ErrCardNotInHand)
	}
	card := models.UnoCard{
		Value: models.CardValue(msg.Card.Value),
		Color: models.CardColor(msg.Card.Color),
	}
	res, err := gameloop.PlayCard(s.game, card, models.CardColor(msg.DeclaredColor))
	if err != nil {
		return err
	}
	if res.UnoWin || s.game.Phase == models.PhaseGameOver {
		return nil
	}
	if s.game.Phase == models.PhaseTurnComplete {
		return gameloop.AdvanceTurn(s.game)
	}
	return nil
}

func handlePlaySubMove(s *GameSession, msg inboundMsg) error {
	if _, err := gameloop.PlaySubMove(s.game, msg.UCI); err != nil {
		return err
	}
	if s.game.Phase == models.PhaseGameOver {
		return nil
	}
	if s.game.Phase == models.PhaseTurnComplete {
		return gameloop.AdvanceTurn(s.game)
	}
	return nil
}

func handlePlayResurrection(s *GameSession, msg inboundMsg) error {
	if s.game.Phase != models.PhaseAwaitingResurrection {
		return fmt.Errorf("%w: not awaiting resurrection", gameloop.ErrNotAwaitingCard)
	}
	// The card that triggered the resurrection is the current discard top.
	top := s.game.DiscardPile[len(s.game.DiscardPile)-1]

	placements := make([]gameloop.Resurrection, 0, len(msg.Placements))
	for _, p := range msg.Placements {
		piece, err := parsePiece(p.Piece)
		if err != nil {
			return err
		}
		sq, err := parseSquare(p.Square)
		if err != nil {
			return err
		}
		placements = append(placements, gameloop.Resurrection{Piece: piece, Square: sq})
	}

	if err := gameloop.PlayResurrection(s.game, top, placements); err != nil {
		return err
	}
	if s.game.Phase == models.PhaseTurnComplete {
		return gameloop.AdvanceTurn(s.game)
	}
	return nil
}

func handleDrawForTurn(s *GameSession) error {
	if _, err := gameloop.DrawForTurn(s.game); err != nil {
		return err
	}
	if s.game.Phase == models.PhaseTurnComplete {
		return gameloop.AdvanceTurn(s.game)
	}
	// Phase == PhaseAwaitingCard means the drawn card is playable; player may now PlayCard.
	return nil
}

// --- Broadcast & error helpers (all require s.mu held) ----------------------

// broadcastLocked pushes filtered state snapshots to all connected clients.
// Must be called with s.mu held.
func (s *GameSession) broadcastLocked() {
	isOver := s.game.Phase == models.PhaseGameOver
	if isOver {
		s.state = sessionFinished
		// Reset the GC timer so the finished game lingers for 30 min for late reads.
		if s.gcTimer != nil {
			s.gcTimer.Reset(gcTimeout)
		}
	}

	goData := []byte(nil)
	if isOver {
		goMsg := buildGameOverMsg(s)
		goData, _ = json.Marshal(goMsg)
	}

	for color, pc := range s.conns {
		if pc == nil {
			continue
		}
		view := buildPlayerView(s, color)
		data, _ := json.Marshal(view)
		pc.enqueue(data)
		if isOver {
			pc.enqueue(goData)
		}
	}
}

func sendErr(pc *playerConn, code, message string) {
	data, _ := json.Marshal(errMsg{Type: "error", Code: code, Message: message})
	pc.enqueue(data)
}

// mapErrCode maps a gameloop sentinel error to its transport-level code string.
func mapErrCode(err error) string {
	for sentinel, code := range errCodeMap {
		if errors.Is(err, sentinel) {
			return code
		}
	}
	return "internal_error"
}

var errCodeMap = map[error]string{
	gameloop.ErrGameOver:            "game_over",
	gameloop.ErrNotAwaitingCard:     "not_awaiting_card",
	gameloop.ErrCardNotInHand:       "card_not_in_hand",
	gameloop.ErrIllegalCardPlay:     "illegal_card_play",
	gameloop.ErrInvalidWildColor:    "invalid_wild_color",
	gameloop.ErrHasPlayableCard:     "has_playable_card",
	gameloop.ErrTurnNotComplete:     "turn_not_complete",
	gameloop.ErrNotStartOfGame:      "not_start_of_game",
	gameloop.ErrNoWildToDeclare:     "no_wild_to_declare",
	gameloop.ErrIllegalSubMove:      "illegal_sub_move",
	gameloop.ErrNoActiveCombo:       "no_active_combo",
	gameloop.ErrComboInProgress:     "combo_in_progress",
	gameloop.ErrCannotResurrectKing: "cannot_resurrect_king",
	gameloop.ErrSquareNotOwnHalf:    "square_not_own_half",
	gameloop.ErrSquareOccupied:      "square_occupied",
	gameloop.ErrPieceNotCaptured:    "piece_not_captured",
	gameloop.ErrTooManyResurrections: "too_many_resurrections",
	gameloop.ErrNotResurrectionCard: "not_resurrection_card",
}

// --- Parse helpers ----------------------------------------------------------

func parsePiece(s string) (chess.PieceType, error) {
	switch strings.ToLower(s) {
	case "queen":
		return chess.Queen, nil
	case "rook":
		return chess.Rook, nil
	case "bishop":
		return chess.Bishop, nil
	case "knight":
		return chess.Knight, nil
	case "pawn":
		return chess.Pawn, nil
	default:
		return chess.NoPieceType, fmt.Errorf("%w: unknown piece type %q", gameloop.ErrInvalidPieceType, s)
	}
}

func parseSquare(sq string) (chess.Square, error) {
	if len(sq) != 2 {
		return chess.A1, fmt.Errorf("invalid square %q: must be exactly 2 characters", sq)
	}
	sq = strings.ToLower(sq)
	f := chess.File(sq[0] - 'a')
	rk := chess.Rank(sq[1] - '1')
	if f < chess.FileA || f > chess.FileH || rk < chess.Rank1 || rk > chess.Rank8 {
		return chess.A1, fmt.Errorf("invalid square %q: out of range", sq)
	}
	return chess.NewSquare(f, rk), nil
}
