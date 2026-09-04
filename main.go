package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Game struct {
	GameID    string       `json:"gameId"`
	Players   []Player     `json:"players"`
	Rounds    []Round      `json:"rounds"`
	Status    string       `json:"status"`
	Mode      string       `json:"mode"`
	CreatedAt time.Time    `json:"createdAt"`
}

type Player struct {
	PlayerID   string `json:"playerId"`
	Name       string `json:"name"`
	TotalScore int    `json:"totalScore"`
	IsKicked   bool   `json:"isKicked"`
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

type GameRoom struct {
	Game           *Game
	Clients        map[*websocket.Conn]bool
	Broadcast      chan interface{}
	Mu             sync.RWMutex
	LastActiveTime map[string]time.Time
}

var (
	rooms    = make(map[string]*GameRoom)
	roomsMu  = sync.RWMutex{}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// ポケモンモードのお題
	pokemonTopics = []string{
		"10万ボルト", "ポケモンBW", "DDラリアット", "ポケモンSV", "アイリス", "アマルルガ",
		"あめふらし", "イカサマ", "イシツブテ", "イッカネズミ", "イワパレス", "ウソッキー",
		"うそなき", "ウッウ", "ウツロイド", "エースバーン", "オーロット", "オクタン",
		"オドリドリ", "オボンのみ", "おまもりこばん", "カイオーガ", "カイリュー", "カスミ",
		"カプ・コケコ", "カブト", "ガラル地方", "ガリョウテンセイ", "ギギギアル", "キノコのほうし",
		"ギルガルド", "キュレム", "キレイハナ", "くちたけん", "クラブ", "くろいヘドロ",
		"クロバット", "ゲッコウガ", "ゲノセクト", "コイキング", "ゴースト", "ゴツゴツメット",
		"ゴニョニョ", "サケブシッポ", "サトシ", "サーフゴー", "サンダー", "サンド",
		"ジグザグマ", "しっぽをふる", "じばく", "シロナ", "シンボラー", "すごいキズぐすり",
		"スプラトゥーン", "スマブラ", "ポケモンスリープ", "ぜったいれいど", "ソーラービーム", "そらをとぶ",
		"タマゴ", "ダークライ", "ダストダス", "タブンネ", "ダンデ", "ダンゴロ",
		"チェリム", "チラチーノ", "ちょうおんぱ", "ちょうはつ", "ツタージャ", "ツンデツンデ",
		"ディグダ", "デスマス", "テッキュウ", "テツノカイナ", "てっぺき", "デリバード",
		"でんこうせっか", "デント", "トーチカ", "どくばり", "ドダイトス", "トドゼルガ",
		"トリック", "トリトドン", "トリプルアクセル", "トルネロス", "どろぼう", "ナックラー",
		"ナッシー", "なみのり", "ナンジャモ", "にほんばれ", "ニャオハ", "ニンフィア",
		"ヌケニン", "ネイティオ", "ネオラント", "ネクロズマ", "ネコにこばん", "ネモ",
		"ネルケ", "ネンドール", "ノコッチ", "ノズパス", "のろい", "ハイパーボール",
		"パチリス", "パッチール", "バトンタッチ", "はらだいこ", "ビーダル", "ビーストブースト",
		"ピカチュウ", "ピクシー", "ひのこ", "ヒヒダルマ", "ビビヨン", "ファイヤー",
		"ブラッキー", "フリージオ", "ブルー", "ブルンゲル", "ヘイラッシャ", "ペパー",
		"ベンチ", "ホイーガ", "ホエルコ", "ポケカ", "ポケモンGO", "ポケモンサークル",
		"ホップ", "ポニータ", "ポリゴン", "マインクラフト", "マタドガス", "まひ",
		"まもる", "マリィ", "マリオカート", "ミガルーサ", "ミュウ", "ミュウツー",
		"ミント", "みがわり", "ムラっけ", "ムゲンダイナ", "メタモン", "モクロー",
		"モジャンボ", "モノズ", "ヤドキング", "ポケモンユナイト", "ゆびをふる", "ヨクバリス",
		"ヨワシ", "ラウドボーン", "ラッキー", "ランクマッチ", "リグレー", "ルカリオ",
		"ルギア", "レジスチル", "ロコン", "ロトム", "ワカシャモ", "ワナイダー", "ワンリキー",
	}

	// 一般モードのお題
	generalTopics = []string{
		"りんご", "バナナ", "猫", "犬", "象", "車", "飛行機", "自転車",
		"机", "椅子", "本", "ペン", "手帳", "スマートフォン", "パソコン",
		"テレビ", "冷蔵庫", "洗濯機", "掃除機", "ベッド", "布団", "枕",
		"太陽", "月", "星", "雲", "雨", "雪", "虹", "風",
		"火", "水", "土", "空気", "大気", "山", "川", "海",
		"森", "砂漠", "地球", "宇宙", "ロボット", "恐竜", "化石",
		"ケーキ", "アイス", "ピザ", "ハンバーガー", "寿司", "ラーメン", "カレー", "パスタ",
		"コーヒー", "紅茶", "牛乳", "ジュース", "ビール", "ワイン",
		"野球", "サッカー", "テニス", "バスケット", "バレーボール", "卓球",
		"水泳", "スキー", "スノーボード", "体操", "柔道", "剣道",
		"ピアノ", "ギター", "ドラム", "バイオリン", "トランペット", "フルート",
		"映画", "ドラマ", "アニメ", "ゲーム", "本", "漫画", "小説",
		"春", "夏", "秋", "冬", "朝", "昼", "夜", "夕焼け",
		"虹", "稲妻", "地震", "火山", "台風", "暴風雪",
		"旅行", "観光", "登山", "キャンプ", "ピクニック", "花見",
		"学校", "図書館", "病院", "駅", "空港", "ホテル", "レストラン",
		"家族", "友達", "先生", "医者", "警察官", "消防士", "兵士",
		"赤", "青", "黄色", "緑", "紫", "オレンジ", "黒", "白",
		"丸", "四角", "三角", "星", "ハート", "月", "太陽",
		"食べる", "飲む", "歩く", "走る", "跳ぶ", "泳ぐ", "寝る", "起きる",
	}
)

// 4桁のランダムIDを生成
func generateShortID() string {
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

// ランダムにお題を選ぶ
func pickRandomTopic(mode string) string {
	if mode == "pokemon" {
		return pokemonTopics[rand.Intn(len(pokemonTopics))]
	}
	return generalTopics[rand.Intn(len(generalTopics))]
}

func calculateScore(charCount, order int) int {
	if order == 0 {
		order = 1
	}
	divisor := float64(charCount) / float64(order)
	return 18 - int(math.Ceil(divisor))
}

// CreateGame - 待機部屋を作成
func CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
	}

	json.NewDecoder(r.Body).Decode(&req)

	if req.Mode == "" {
		req.Mode = "pokemon"
	}

	gameID := generateShortID()
	game := &Game{
		GameID:    gameID,
		Players:   []Player{},
		Rounds:    []Round{},
		Status:    "waiting",
		Mode:      req.Mode,
		CreatedAt: time.Now(),
	}

	// 親を追加
	game.Players = append(game.Players, Player{
		PlayerID:   uuid.New().String(),
		Name:       req.Name,
		TotalScore: 0,
		IsKicked:   false,
	})

	room := &GameRoom{
		Game:           game,
		Clients:        make(map[*websocket.Conn]bool),
		Broadcast:      make(chan interface{}, 10),
		LastActiveTime: make(map[string]time.Time),
	}

	roomsMu.Lock()
	rooms[gameID] = room
	roomsMu.Unlock()

	go room.broadcastLoop()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

// GetGame - ゲーム情報を取得
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

// JoinGame - ゲームにプレイヤーを追加
func JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Name == "" {
		http.Error(w, "Player name required", http.StatusBadRequest)
		return
	}

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	// ゲーム開始後は参加禁止
	if room.Game.Status != "waiting" {
		http.Error(w, "Game already started", http.StatusBadRequest)
		return
	}

	// 新しいプレイヤーを追加
	newPlayer := Player{
		PlayerID:   uuid.New().String(),
		Name:       req.Name,
		TotalScore: 0,
		IsKicked:   false,
	}

	room.Game.Players = append(room.Game.Players, newPlayer)
	room.Broadcast <- room.Game

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newPlayer)
}

// StartGame - ゲームを開始（待機部屋から本ゲームへ）
func StartGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	if room.Game.Status != "waiting" {
		http.Error(w, "Game already started", http.StatusBadRequest)
		return
	}

	// プレイヤーごとにラウンドを作成 + 自動でお題を割り当て
	room.Game.Rounds = []Round{}
	for i, player := range room.Game.Players {
		if player.IsKicked {
			continue
		}
		for j := 0; j < 2; j++ {
			room.Game.Rounds = append(room.Game.Rounds, Round{
				RoundNumber: len(room.Game.Rounds) + 1,
				ParentID:    room.Game.Players[i].PlayerID,
				Answer:      pickRandomTopic(room.Game.Mode), // 自動割り当て
				Status:      "hint_phase",                     // 親には見せず、すぐhint_phaseへ
				Hints:       []Hint{},
				Scores:      make(map[string]int),
				CreatedAt:   time.Now(),
			})
		}
	}

	room.Game.Status = "playing"
	room.Broadcast <- room.Game

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

// KickPlayer - プレイヤーをキック
func KickPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		PlayerID string `json:"playerId"`
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

	for i, player := range room.Game.Players {
		if player.PlayerID == req.PlayerID {
			room.Game.Players[i].IsKicked = true
			room.Broadcast <- room.Game
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "kicked"})
			return
		}
	}

	http.Error(w, "Player not found", http.StatusNotFound)
}

// EndGame - ゲームを終了
func EndGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, ok := rooms[gameID]
	roomsMu.RUnlock()

	if !ok {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	room.Game.Status = "finished"
	room.Broadcast <- room.Game

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

// SubmitHint - ヒントを投稿
func SubmitHint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		Text string `json:"text"`
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

	playerID := r.Header.Get("X-Player-ID")

	for i := range room.Game.Rounds {
		if room.Game.Rounds[i].Status == "hint_phase" {
			hint := Hint{
				PlayerID:  playerID,
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

// SubmitAnswer - 解答を投稿
func SubmitAnswer(w http.ResponseWriter, r *http.Request) {
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

// WebSocketHandler - WebSocket接続
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

	playerID := r.Header.Get("X-Player-ID")

	room.Mu.Lock()
	room.Clients[ws] = true
	room.LastActiveTime[playerID] = time.Now()
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
				room.Mu.Lock()
				room.LastActiveTime[playerID] = time.Now()
				room.Mu.Unlock()
				break
			}
			room.Mu.Lock()
			room.LastActiveTime[playerID] = time.Now()
			room.Mu.Unlock()
		}
	}()
}

// broadcastLoop - ブロードキャスト
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

// HealthCheck
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/health", HealthCheck).Methods("GET")
	r.HandleFunc("/api/games", CreateGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}", GetGame).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/join", JoinGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/start", StartGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/end", EndGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/kick", KickPlayer).Methods("POST", "OPTIONS")
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
