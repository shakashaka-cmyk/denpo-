package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
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
	PlayerID       string `json:"playerId"`
	Name           string `json:"name"`
	TotalScore     int    `json:"totalScore"`
	IsKicked       bool   `json:"isKicked"`
	HintSubmitted  bool   `json:"hintSubmitted"`
}

type Round struct {
	RoundNumber      int            `json:"roundNumber"`
	ParentID         string         `json:"parentId"`
	Answer           string         `json:"answer"`
	Status           string         `json:"status"`
	Hints            []Hint         `json:"hints"`
	CorrectAnswer    bool           `json:"correctAnswer"`
	Scores           map[string]int `json:"scores"`
	CreatedAt        time.Time      `json:"createdAt"`
	AnsweredAt       *time.Time     `json:"answeredAt"`
	RevealedHintIdx  int            `json:"revealedHintIdx"`
	CorrectHintIdx   int            `json:"correctHintIdx"`
}

type Hint struct {
	PlayerID  string    `json:"playerId"`
	PlayerName string   `json:"playerName"`
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

	generalTopics = []string{
		"りんご", "みかん", "バナナ", "いちご", "ぶどう", "スイカ",
		"ねこ", "いぬ", "うさぎ", "ねずみ", "ぱんだ", "らいおん",
		"くるま", "でんしゃ", "ひこうき", "ふね", "じてんしゃ", "けいたい",
		"がっこう", "びょういん", "とうきょう", "きょうと", "おおさか", "ほっかいどう",
		"さくら", "もみじ", "あじさい", "ひまわり", "ばら", "ゆり",
		"ぎゅうにゅう", "たまご", "パン", "こめ", "らーめん", "すし",
		"あつこうやま", "あずきバー", "ようかん", "どらやき", "わらびもち", "みたらし",
		"ぐりーんぴー", "もも", "すもも", "きゅうり", "にんじん", "とうもろこし",
		"ぱそこん", "てれび", "ゲーム", "すまーとふぉん", "かめら", "らじお",
		"せんせい", "けいさんき", "せんまじ", "しょくぎょう", "かいぎ", "べんきょう",
		"あき", "ふゆ", "はる", "なつ", "しろい", "くろい",
		"あかい", "あおい", "きいろい", "ちいさい", "おおきい", "ながい",
	}
)

// Keepalive: Renderの15分スリープ対策
func init() {
	go func() {
		time.Sleep(5 * time.Minute)
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			port := os.Getenv("PORT")
			if port == "" {
				port = "8080"
			}
			healthURL := fmt.Sprintf("http://localhost:%s/api/games", port)
			
			resp, err := http.Get(healthURL)
			if err != nil {
				fmt.Println("[Keepalive] エラー:", err)
				continue
			}
			resp.Body.Close()
			fmt.Println("[Keepalive] Renderを起動状態に保持 -", time.Now().Format("15:04:05"))
		}
	}()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/games", CreateGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games", ListGames).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}", GetGame).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/join", JoinGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/start", StartGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/end", EndGame).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/kick", KickPlayer).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/hints", SubmitHint).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/answer", SubmitAnswer).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/games/{gameId}/reveal", RevealNextHint).Methods("POST", "OPTIONS")
	r.Use(corsMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server starting on port", port)
	http.ListenAndServe(":"+port, r)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:3000" || origin == "https://denpo-ten.vercel.app" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Player-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ListGames(w http.ResponseWriter, r *http.Request) {
	roomsMu.RLock()
	defer roomsMu.RUnlock()

	games := make([]*Game, 0)
	for _, room := range rooms {
		room.Mu.RLock()
		games = append(games, room.Game)
		room.Mu.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func CreateGame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	gameID := generateShortID()
	playerID := uuid.New().String()

	game := &Game{
		GameID: gameID,
		Players: []Player{
			{
				PlayerID:      playerID,
				Name:          req.Name,
				TotalScore:    0,
				IsKicked:      false,
				HintSubmitted: false,
			},
		},
		Status:    "waiting",
		Mode:      req.Mode,
		CreatedAt: time.Now(),
	}

	roomsMu.Lock()
	rooms[gameID] = &GameRoom{
		Game:           game,
		Clients:        make(map[*websocket.Conn]bool),
		Broadcast:      make(chan interface{}, 256),
		LastActiveTime: make(map[string]time.Time),
	}
	roomsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func GetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Game not found\n"))
		return
	}

	room.Mu.RLock()
	defer room.Mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func JoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Game not found\n"))
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	playerID := uuid.New().String()
	player := Player{
		PlayerID:      playerID,
		Name:          req.Name,
		TotalScore:    0,
		IsKicked:      false,
		HintSubmitted: false,
	}
	room.Game.Players = append(room.Game.Players, player)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func StartGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	activePlayers := make([]Player, 0)
	for _, p := range room.Game.Players {
		if !p.IsKicked {
			activePlayers = append(activePlayers, p)
		}
	}

	topics := pokemonTopics
	if room.Game.Mode == "general" {
		topics = generalTopics
	}

	room.Game.Rounds = make([]Round, 0)
	for i := 0; i < 2; i++ {
		for _, player := range activePlayers {
			answer := topics[rand.Intn(len(topics))]
			round := Round{
				RoundNumber:    len(room.Game.Rounds) + 1,
				ParentID:       player.PlayerID,
				Answer:         answer,
				Status:         "hint_phase",
				Hints:          make([]Hint, 0),
				CorrectAnswer:  false,
				Scores:         make(map[string]int),
				CreatedAt:      time.Now(),
				RevealedHintIdx: -1,
				CorrectHintIdx: -1,
			}
			room.Game.Rounds = append(room.Game.Rounds, round)
		}
	}

	// ラウンド開始時に全プレイヤーのHintSubmittedをリセット
	for i := range room.Game.Players {
		room.Game.Players[i].HintSubmitted = false
	}

	room.Game.Status = "playing"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func SubmitHint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	playerID := r.Header.Get("X-Player-ID")

	var req struct {
		Text string `json:"text"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	if len(room.Game.Rounds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentRound := &room.Game.Rounds[len(room.Game.Rounds)-1]

	// プレイヤーの情報を取得
	var playerName string
	for i, p := range room.Game.Players {
		if p.PlayerID == playerID {
			if p.HintSubmitted {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Already submitted hint\n"))
				return
			}
			playerName = p.Name
			room.Game.Players[i].HintSubmitted = true
			break
		}
	}

	hint := Hint{
		PlayerID:   playerID,
		PlayerName: playerName,
		Text:       req.Text,
		CharCount:  len([]rune(req.Text)),
		Order:      len(currentRound.Hints) + 1,
		Score:      0,
		CreatedAt:  time.Now(),
	}
	currentRound.Hints = append(currentRound.Hints, hint)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func RevealNextHint(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	if len(room.Game.Rounds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentRound := &room.Game.Rounds[len(room.Game.Rounds)-1]

	// ヒントを文字数でソート
	sortHints(currentRound.Hints)

	// 次のヒントインデックスを計算
	nextIdx := currentRound.RevealedHintIdx + 1
	if nextIdx >= len(currentRound.Hints) {
		nextIdx = len(currentRound.Hints) - 1
	}

	currentRound.RevealedHintIdx = nextIdx

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	playerID := r.Header.Get("X-Player-ID")

	var req struct {
		Answer string `json:"answer"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	if len(room.Game.Rounds) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentRound := &room.Game.Rounds[len(room.Game.Rounds)-1]

	// 回答を正規化
	normalizedAnswer := normalizeText(req.Answer)
	normalizedCorrectAnswer := normalizeText(currentRound.Answer)

	isCorrect := normalizedAnswer == normalizedCorrectAnswer

	if isCorrect {
		currentRound.CorrectAnswer = true
		currentRound.Status = "finished"
		now := time.Now()
		currentRound.AnsweredAt = &now

		// ヒントを文字数でソート
		sortHints(currentRound.Hints)

		// 開示されたヒントのプレイヤーに点数を与える
		if currentRound.RevealedHintIdx >= 0 && currentRound.RevealedHintIdx < len(currentRound.Hints) {
			correctHint := currentRound.Hints[currentRound.RevealedHintIdx]
			currentRound.CorrectHintIdx = currentRound.RevealedHintIdx

			// スコア計算: (18 - 文字数) ÷ 順番（小数点切り上げ）
			charCount := correctHint.CharCount
			hintOrder := currentRound.RevealedHintIdx + 1
			score := int(math.Ceil(float64(18-charCount) / float64(hintOrder)))

			// 親と正解ヒント提出者に点数を与える
			currentRound.Scores[playerID] = score              // 親
			currentRound.Scores[correctHint.PlayerID] = score  // ヒント提出者

			// 総スコアに加算
			for i := range room.Game.Players {
				if room.Game.Players[i].PlayerID == playerID {
					room.Game.Players[i].TotalScore += score
				}
				if room.Game.Players[i].PlayerID == correctHint.PlayerID {
					room.Game.Players[i].TotalScore += score
				}
			}
		}

		// すべてのラウンドが終了したか確認
		allFinished := true
		for _, round := range room.Game.Rounds {
			if round.Status != "finished" {
				allFinished = false
				break
			}
		}

		if allFinished {
			room.Game.Status = "finished"
		}
	}

	// 新しいラウンドが始まるときにリセット
	resetHintSubmittedForNewRound(room.Game)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func EndGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	room.Game.Status = "finished"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

func KickPlayer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]

	var req struct {
		PlayerId string `json:"playerId"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	roomsMu.RLock()
	room, exists := rooms[gameID]
	roomsMu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	for i := range room.Game.Players {
		if room.Game.Players[i].PlayerID == req.PlayerId {
			room.Game.Players[i].IsKicked = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Game)
}

// resetHintSubmittedForNewRound: 新しいラウンドが始まるときにリセット
func resetHintSubmittedForNewRound(game *Game) {
	if len(game.Rounds) == 0 {
		return
	}
	
	currentRound := &game.Rounds[len(game.Rounds)-1]
	currentRoundNumber := currentRound.RoundNumber
	
	// 同じラウンド番号の他のラウンドが全て終了したかチェック
	roundsOfSameNumber := 0
	finishedRoundsOfSameNumber := 0
	
	for _, r := range game.Rounds {
		if r.RoundNumber == currentRoundNumber {
			roundsOfSameNumber++
			if r.Status == "finished" {
				finishedRoundsOfSameNumber++
			}
		}
	}
	
	// 全て終了したら次のラウンドのために HintSubmitted をリセット
	if roundsOfSameNumber == finishedRoundsOfSameNumber && roundsOfSameNumber > 0 {
		for i := range game.Players {
			game.Players[i].HintSubmitted = false
		}
	}
}

// Helper functions
func generateShortID() string {
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func normalizeText(text string) string {
	text = strings.TrimSpace(text)
	text = hiraganaToKatakana(text)
	text = removeSpecialChars(text)
	return text
}

func hiraganaToKatakana(text string) string {
	result := []rune{}
	for _, r := range text {
		if r >= 'ぁ' && r <= 'ん' {
			result = append(result, r+0x60)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func removeSpecialChars(text string) string {
	result := []rune{}
	for _, r := range text {
		if (r >= 0x3000 && r <= 0x309f) ||
			(r >= 0x30a0 && r <= 0x30ff) ||
			(r >= '0' && r <= '9') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z') ||
			r == ' ' {
			result = append(result, r)
		}
	}
	return string(result)
}

func sortHints(hints []Hint) {
	for i := 0; i < len(hints); i++ {
		for j := i + 1; j < len(hints); j++ {
			if hints[j].CharCount < hints[i].CharCount {
				hints[i], hints[j] = hints[j], hints[i]
			}
		}
	}
}
