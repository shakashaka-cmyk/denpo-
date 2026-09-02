package main

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Game struct {
	GameID    string    `json:"gameId"`
	Players   []Player  `json:"players"`
	Rounds    []Round   `json:"rounds"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Player struct {
	PlayerID   string `json:"playerId"`
	Name       string `json:"name"`
	TotalScore int    `json:"totalScore"`
}

type Round struct {
	RoundNumber   int            `json:"roundNumber"`
	ParentID      string         `json:"parentId"`
	Answer        string         `json:"answer"`
	Status        string         `json:"status"`
	Hints         []Hint         `json:"hints"`
	CorrectAnswer bool           `json:"correctAnswer"`
	Scores        map[string]int `json:"scores"`
	CreatedAt     time.Time      `json:"createdAt"`
	AnsweredAt    *time.Time     `json:"answeredAt"`
}

type Hint struct {
	PlayerID  string    `json:"playerId"`
	Text      string    `json:"text"`
	CharCount int       `json:"charCount"`
	Order     int       `json:"order"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
}

type AnswerRequest struct {
	Answer string `json:"answer"`
}

type HintRequest struct {
	Text string `json:"text"`
}

type GameRoom struct {
	Game      *Game
	Clients   map[*websocket.Conn]bool
	Broadcast chan interface{}
	Mu        sync.RWMutex
}

var (
	rooms    = make(map[string]*GameRoom)
	roomsMu  = sync.RWMutex{}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func calculateScore(charCount, order int) int {
	if order == 0 {
		order = 1
	}
	divisor := float64(charCount) / float64(order)
	return 18 - int(math.Ceil(divisor))
}

func CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Players []struct {
			Name string `json:"name"`
		} `json:"players"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	gameID := uuid.New().String()
	game := &Game{
		GameID:    gameID,
		Players:   []Player{},
		Rounds:    []Round{},
		Status:    "waiting",
		CreatedAt: time.Now(),
	}

	for i, p := range req.Players {
		game.Players = append(game.Players, Player{
			PlayerID:   uuid.New().String(),
			Name:       p.Name,
			TotalScore: 0,
		})

		for j := 0; j < 2; j++ {
			game.Rounds = append(game.Rounds, Round{
				RoundNumber: len(game.Rounds) + 1,
				ParentID:    game.Players[i].PlayerID,
				Status:      "waiting",
				Hints:       []Hint{},
				Scores:      make(map[string]int),
				CreatedAt:   time.Now(),
			})
		}
	}

	room := &GameRoom{
		Game:      game,
		Clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan interface{}, 10),
	}

	roomsMu.Lock()
	rooms[gameID] = room
	roomsMu.Unlock()

	go room.broadcastLoop()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

func GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func StartRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		Answer string `json:"answer"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	for i := range room.Game.Rounds {
		if room.Game.Rounds[i].Status == "waiting" {
			room.Game.Rounds[i].Answer = req.Answer
			room.Game.Rounds[i].Status = "hint_phase"
			room.Broadcast <- room.Game

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(room.Game.Rounds[i])
			return
		}
	}

	http.Error(w, "No waiting round found", http.StatusNotFound)
}

func SubmitHint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req HintRequest
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	for i := range room.Game.Rounds {
		if room.Game.Rounds[i].Status == "hint_phase" {
			hint := Hint{
				PlayerID:  r.Header.Get("X-Player-ID"),
				Text:      req.Text,
				CharCount: len([]rune(req.Text)),
				Order:     len(room.Game.Rounds[i].Hints) + 1,
				CreatedAt: time.Now(),
			}

			hint.Score = calculateScore(hint.CharCount, hint.Order)
			room.Game.Rounds[i].Hints = append(room.Game.Rounds[i].Hints, hint)
			room.Broadcast <- room.Game

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(hint)
			return
		}
	}

	http.Error(w, "No active hint phase found", http.StatusNotFound)
}

func SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req AnswerRequest
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	playerID := r.Header.Get("X-Player-ID")

	for i := range room.Game.Rounds {
		if room.Game.Rounds[i].Status == "hint_phase" {
			round := &room.Game.Rounds[i]
			now := time.Now()
			round.AnsweredAt = &now

			if req.Answer == round.Answer {
				round.CorrectAnswer = true
				round.Status = "finished"

				score := 0
				if len(round.Hints) > 0 {
					lastHint := round.Hints[len(round.Hints)-1]
					score = calculateScore(lastHint.CharCount, lastHint.Order)
				}

				if score > 0 {
					round.Scores[round.ParentID] = score
					round.Scores[playerID] = score

					for j := range room.Game.Players {
						if room.Game.Players[j].PlayerID == round.ParentID || room.Game.Players[j].PlayerID == playerID {
							room.Game.Players[j].TotalScore += score
						}
					}
				}
			} else {
				round.CorrectAnswer = false
				round.Status = "finished"
			}

			room.Broadcast <- room.Game

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(round)
			return
		}
	}

	http.Error(w, "No active round found", http.StatusNotFound)
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	room.Mu.Lock()
	room.Clients[ws] = true
	room.Mu.Unlock()

	ws.WriteJSON(room.Game)

	go func() {
		defer func() {
			room.Mu.Lock()
			delete(room.Clients, ws)
			room.Mu.Unlock()
			ws.Close()
		}()

		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (room *GameRoom) broadcastLoop() {
	for msg := range room.Broadcast {
		room.Mu.RLock()
		clients := make(map[*websocket.Conn]bool)
		for k, v := range room.Clients {
			clients[k] = v
		}
		room.Mu.RUnlock()

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				room.Mu.Lock()
				delete(room.Clients, client)
				room.Mu.Unlock()
				client.Close()
			}
		}
	}
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/health", HealthCheck).Methods("GET")
	r.HandleFunc("/api/games", CreateGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}", GetGame).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/rounds/start", StartRound).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/hints", SubmitHint).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/answer", SubmitAnswer).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/ws", WebSocketHandler)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "https://denpo-ten.vercel.app")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Player-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.ListenAndServe(":"+port, r)
}
