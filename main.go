package main

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Game struct {
	GameID    string    `json:"gameId"`
	Players   []Player  `json:"players"`
	Rounds    []Round   `json:"rounds"`
	Status    string    `json:"status"` // "waiting", "playing", "finished"
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
	Status        string         `json:"status"` // "hint_phase", "answering", "finished"
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

var games = make(map[string]*Game)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// スコア計算：18 - 文字数 ÷ 順番（小数点切り上げ）
func calculateScore(charCount, order int) int {
	if order == 0 {
		order = 1
	}
	divisor := float64(charCount) / float64(order)
	return 18 - int(math.Ceil(divisor))
}

// CreateGame - 新しいゲームを作成
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

		// 各プレイヤーが親として2回担当するラウンドを作成
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

	games[gameID] = game
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// GetGame - ゲーム情報を取得
func GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// StartRound - ラウンドを開始
func StartRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		Answer string `json:"answer"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// ラウンド番号でラウンドを検索
	var round *Round
	for i := range game.Rounds {
		if game.Rounds[i].RoundNumber == 1 {
			round = &game.Rounds[i]
			break
		}
	}

	if round == nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	round.Answer = req.Answer
	round.Status = "hint_phase"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(round)
}

// SubmitHint - ヒントを投稿
func SubmitHint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req HintRequest
	json.NewDecoder(r.Body).Decode(&req)

	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// ラウンドを探す
	var round *Round
	for i := range game.Rounds {
		if game.Rounds[i].RoundNumber == 1 {
			round = &game.Rounds[i]
			break
		}
	}

	if round == nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	hint := Hint{
		PlayerID:  r.Header.Get("X-Player-ID"),
		Text:      req.Text,
		CharCount: len([]rune(req.Text)), // 日本語に対応
		Order:     len(round.Hints) + 1,
		CreatedAt: time.Now(),
	}

	round.Hints = append(round.Hints, hint)

	// スコアを計算
	hint.Score = calculateScore(hint.CharCount, hint.Order)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hint)
}

// SubmitAnswer - 解答を投稿
func SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req AnswerRequest
	json.NewDecoder(r.Body).Decode(&req)

	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	// ラウンドを探す
	var round *Round
	for i := range game.Rounds {
		if game.Rounds[i].RoundNumber == 1 {
			round = &game.Rounds[i]
			break
		}
	}

	if round == nil {
		http.Error(w, "Round not found", http.StatusNotFound)
		return
	}

	playerID := r.Header.Get("X-Player-ID")
	now := time.Now()
	round.AnsweredAt = &now

	// 正解判定
	if req.Answer == round.Answer {
		round.CorrectAnswer = true
		round.Status = "finished"

		// 親と解答者に同じスコアを付与
		score := 0
		if len(round.Hints) > 0 {
			lastHint := round.Hints[len(round.Hints)-1]
			score = calculateScore(lastHint.CharCount, lastHint.Order)
		}

		round.Scores[round.ParentID] = score
		round.Scores[playerID] = score

		// 総スコアを更新
		for i, player := range game.Players {
			if player.PlayerID == round.ParentID || player.PlayerID == playerID {
				game.Players[i].TotalScore += score
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(round)
}

// WebSocket接続ハンドラ
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	game, ok := games[gameID]
	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// ゲーム状態をブロードキャスト
	for {
		err := ws.WriteJSON(game)
		if err != nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func main() {
	r := mux.NewRouter()

	// API エンドポイント
	r.HandleFunc("/api/games", CreateGame).Methods("POST")
	r.HandleFunc("/api/games/{gameId}", GetGame).Methods("GET")
	r.HandleFunc("/api/games/{gameId}/rounds/{roundNumber}/start", StartRound).Methods("POST")
	r.HandleFunc("/api/games/{gameId}/rounds/{roundNumber}/hints", SubmitHint).Methods("POST")
	r.HandleFunc("/api/games/{gameId}/rounds/{roundNumber}/answer", SubmitAnswer).Methods("POST")
	r.HandleFunc("/api/games/{gameId}/ws", WebSocketHandler)

	// ヘルスチェック
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// CORS対応
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowedOrigins := []string{
				"https://denpo-ten.vercel.app",
				"https://denpou-game.vercel.app",
				"http://localhost:3000",
				"http://localhost:8080",
			}

			origin := r.Header.Get("Origin")
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Player-ID")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// ポート
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.ListenAndServe(":"+port, r)
}
