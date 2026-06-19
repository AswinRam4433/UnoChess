package transport

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/notnil/chess"

	"unochess/gameloop"
)

// Server is the HTTP + WebSocket server for UnoChess.
type Server struct {
	reg *Registry
	mux *http.ServeMux
}

// NewServer creates a Server backed by reg and registers all routes.
func NewServer(reg *Registry) *Server {
	s := &Server{reg: reg, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /games", s.createGame)
	s.mux.HandleFunc("POST /games/{gameID}/join", s.joinGame)
	s.mux.HandleFunc("GET /games/{gameID}/play", s.playWS)
	s.mux.HandleFunc("GET /games/{gameID}", s.getGame)
	s.mux.HandleFunc("GET /games", s.listGames)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Permissive CORS for development; tighten in production.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

// --- HTTP request/response types --------------------------------------------

type createGameReq struct {
	Opponent string `json:"opponent"` // "human" | "bot"; default "human"
}

// CreateGameResp is the response body for POST /games.
type CreateGameResp struct {
	GameID string `json:"gameID"`
	Token  string `json:"playerToken"`
	Color  string `json:"playerColor"`
}

// JoinGameResp is the response body for POST /games/{id}/join.
type JoinGameResp struct {
	Token string `json:"playerToken"`
	Color string `json:"playerColor"`
}

type gameMetaResp struct {
	ID      string          `json:"id"`
	State   string          `json:"state"`
	Players map[string]bool `json:"players"` // "White"/"Black" → has joined
	Winner  *string         `json:"winner"`
	Reason  *string         `json:"reason"`
	Turns   int             `json:"turns"`
}

type listGamesResp struct {
	Games []string `json:"games"`
}

// --- HTTP handlers ----------------------------------------------------------

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	var req createGameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid_request", "malformed JSON body", http.StatusBadRequest)
		return
	}
	isBot := req.Opponent == "bot"

	id := newID()
	token := newToken()

	session := s.reg.newGame(id, isBot)
	session.mu.Lock()
	session.tokens[token] = chess.White
	session.mu.Unlock()

	writeJSON(w, http.StatusCreated, CreateGameResp{
		GameID: id,
		Token:  token,
		Color:  "White",
	})
}

func (s *Server) joinGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameID")
	session, ok := s.reg.get(gameID)
	if !ok {
		jsonError(w, "game_not_found", "no such game", http.StatusNotFound)
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.isBot {
		jsonError(w, "game_full", "bot games have no second human player", http.StatusConflict)
		return
	}
	if session.state != sessionLobby {
		jsonError(w, "game_full", "game is already in progress or finished", http.StatusConflict)
		return
	}

	token := newToken()
	session.tokens[token] = chess.Black
	session.state = sessionPlaying

	writeJSON(w, http.StatusOK, JoinGameResp{Token: token, Color: "Black"})
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	gameID := r.PathValue("gameID")
	session, ok := s.reg.get(gameID)
	if !ok {
		jsonError(w, "game_not_found", "no such game", http.StatusNotFound)
		return
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	players := map[string]bool{"White": false, "Black": false}
	for _, color := range session.tokens {
		players[colorName(color)] = true
	}
	if session.isBot {
		players["Black"] = true
	}

	resp := gameMetaResp{
		ID:      session.id,
		State:   stateName(session.state),
		Players: players,
		Turns:   len(session.game.History),
	}
	if session.state == sessionFinished {
		winner := colorName(session.game.Winner)
		reason := string(gameloop.ClassifyEnd(session.game))
		resp.Winner = &winner
		resp.Reason = &reason
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) listGames(w http.ResponseWriter, r *http.Request) {
	ids := s.reg.lobbyIDs()
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, listGamesResp{Games: ids})
}

// --- helpers ----------------------------------------------------------------

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func jsonError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiErrorBody{Code: code, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func stateName(s sessionState) string {
	switch s {
	case sessionLobby:
		return "lobby"
	case sessionPlaying:
		return "playing"
	case sessionFinished:
		return "finished"
	default:
		return "unknown"
	}
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failure: %v", err))
	}
	// UUID v4 encoding
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failure: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// tokenEqual performs a constant-time string comparison to prevent timing attacks.
func tokenEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
