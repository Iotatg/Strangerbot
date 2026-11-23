package bot

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "strangerbot/internal/tgapi"
)

type gameKind int

const (
	gameRPS gameKind = iota
	gameTTT
)

// game is one head-to-head match between two users.
type game struct {
	ID      int64
	Kind    gameKind
	A, B    int64
	ChoiceA string
	ChoiceB string
	Board   [9]int64
	Turn    int64
	Done    bool
	At      time.Time
	MsgIDs  map[int64]int
	mu      sync.Mutex
}

// gameManager holds live games in memory.
type gameManager struct {
	mu    sync.Mutex
	games map[int64]*game
	seq   int64
}

func newGameManager() *gameManager {
	return &gameManager{games: map[int64]*game{}}
}

func (g *gameManager) create(kind gameKind, a, b int64) *game {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	gm := &game{
		ID:     g.seq,
		Kind:   kind,
		A:      a,
		B:      b,
		Turn:   a,
		At:     time.Now(),
		MsgIDs: map[int64]int{},
	}
	g.games[gm.ID] = gm
	return gm
}

func (g *gameManager) get(id int64) (*game, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	gm, ok := g.games[id]
	return gm, ok
}

func (g *gameManager) drop(id int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.games, id)
}

// cancelGamesFor ends every live game of uid.
func (b *Bot) cancelGamesFor(uid int64) {
	b.games.mu.Lock()
	var ended []*game
	for _, gm := range b.games.games {
		if gm.Done || (gm.A != uid && gm.B != uid) {
			continue
		}
		gm.Done = true
		ended = append(ended, gm)
		delete(b.games.games, gm.ID)
	}
	b.games.mu.Unlock()
	for _, gm := range ended {
		b.sendText(gm.A, GameCancelled, nil)
		b.sendText(gm.B, GameCancelled, nil)
	}
}

// sweepGames drops stale games.
func (b *Bot) sweepGames() {
	b.games.mu.Lock()
	var stale []*game
	for _, gm := range b.games.games {
		if !gm.Done && time.Since(gm.At) > 10*time.Minute {
			gm.Done = true
			stale = append(stale, gm)
			delete(b.games.games, gm.ID)
		}
	}
	b.games.mu.Unlock()
	for _, gm := range stale {
		b.sendText(gm.A, GameCancelled, nil)
		b.sendText(gm.B, GameCancelled, nil)
	}
}

// startGamesLoop periodically cleans up abandoned games.
func (b *Bot) startGamesLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		b.sweepGames()
	}
}

func (b *Bot) cmdGames(m *tgbotapi.Message) {
	b.sendText(m.From.ID, GamesMenu, nil)
}

// ── Rock paper scissors ──────────────────────────────────────────────────────

func (b *Bot) cmdRPS(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.soloRPS(uid)
		return
	}
	gm := b.games.create(gameRPS, uid, pid)
	kb := rpsKeyboard(gm.ID)
	b.sendGamePrompt(uid, gm, kb)
	b.sendGamePrompt(pid, gm, kb)
}

func rpsKeyboard(id int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🪨 Rock", fmt.Sprintf("rps:%d:rock", id)),
			tgbotapi.NewInlineKeyboardButtonData("📄 Paper", fmt.Sprintf("rps:%d:paper", id)),
			tgbotapi.NewInlineKeyboardButtonData("✂️ Scissors", fmt.Sprintf("rps:%d:scissors", id)),
		),
	)
}

func (b *Bot) sendGamePrompt(uid int64, gm *game, kb tgbotapi.InlineKeyboardMarkup) {
	text := fmt.Sprintf(GameRPSPrompt, b.partnerName(uid, gm))
	mid := b.sendTextID(uid, text, kb)
	gm.mu.Lock()
	gm.MsgIDs[uid] = mid
	gm.mu.Unlock()
}

func (b *Bot) partnerName(uid int64, gm *game) string {
	other := gm.B
	if uid == gm.B {
		other = gm.A
	}
	u, ok := b.st.GetUser(other)
	if !ok {
		return "your partner"
	}
	if u.FirstName != "" {
		return escapeHTML(u.FirstName)
	}
	return "your partner"
}

func (b *Bot) cbRPS(cq *tgbotapi.CallbackQuery) {
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	choice := parts[2]
	if choice != "rock" && choice != "paper" && choice != "scissors" {
		b.answerCallback(cq, "", false)
		return
	}
	gm, ok := b.games.get(id)
	if !ok || gm.Done || gm.Kind != gameRPS {
		b.answerCallback(cq, "This game is over.", true)
		return
	}
	if cq.From.ID != gm.A && cq.From.ID != gm.B {
		b.answerCallback(cq, "Not your game.", true)
		return
	}
	b.games.mu.Lock()
	if cq.From.ID == gm.A {
		gm.ChoiceA = choice
	} else {
		gm.ChoiceB = choice
	}
	ready := gm.ChoiceA != "" && gm.ChoiceB != ""
	b.games.mu.Unlock()
	b.answerCallback(cq, "Locked in ✅", false)
	if !ready {
		b.editText(cq.From.ID, cq.Message.MessageID, "✅ Choice locked. Waiting for your partner…", nil)
		return
	}
	b.resolveRPS(gm)
}

func (b *Bot) resolveRPS(gm *game) {
	winner := rpsWinner(gm.ChoiceA, gm.ChoiceB)
	if winner == 0 {
		b.finishGame(gm, "🤝 It's a draw! Both chose "+rpsLabel(gm.ChoiceA)+".")
		return
	}
	winID := gm.A
	winningChoice, losingChoice := gm.ChoiceA, gm.ChoiceB
	if winner == 2 {
		winID = gm.B
		winningChoice, losingChoice = gm.ChoiceB, gm.ChoiceA
	}
	u, _ := b.st.GetUser(winID)
	winName := displayName(storeUserView{ID: u.ID, FirstName: u.FirstName, Username: u.Username})
	text := fmt.Sprintf(GameRPSResult, winName, rpsLabel(winningChoice), rpsLabel(losingChoice))
	b.finishGame(gm, text)
}

func (b *Bot) finishGame(gm *game, text string) {
	gm.Done = true
	b.games.drop(gm.ID)
	gm.mu.Lock()
	defer gm.mu.Unlock()
	for _, uid := range []int64{gm.A, gm.B} {
		if mid, ok := gm.MsgIDs[uid]; ok && mid != 0 {
			b.editText(uid, mid, text, nil)
		} else {
			b.sendText(uid, text, nil)
		}
	}
}

func rpsWinner(a, b string) int {
	if a == b {
		return 0
	}
	beats := map[string]string{"rock": "scissors", "paper": "rock", "scissors": "paper"}
	if beats[a] == b {
		return 1
	}
	return 2
}

func rpsLabel(c string) string {
	switch c {
	case "rock":
		return "🪨 Rock"
	case "paper":
		return "📄 Paper"
	default:
		return "✂️ Scissors"
	}
}

// soloRPS is the free practice mode.
func (b *Bot) soloRPS(uid int64) {
	choices := []string{"rock", "paper", "scissors"}
	bot := choices[rand.Intn(len(choices))]
	me := choices[rand.Intn(len(choices))]
	winner := rpsWinner(me, bot)
	switch winner {
	case 0:
		b.sendText(uid, fmt.Sprintf(SoloRPSTie, rpsLabel(me), rpsLabel(bot)), nil)
	case 1:
		b.sendText(uid, fmt.Sprintf(SoloRPSWin, rpsLabel(me), rpsLabel(bot)), nil)
	default:
		b.sendText(uid, fmt.Sprintf(SoloRPSLose, rpsLabel(me), rpsLabel(bot)), nil)
	}
}

// ── Tic tac toe ──────────────────────────────────────────────────────────────

func (b *Bot) cmdTTT(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, GameNotInChat, nil)
		return
	}
	gm := b.games.create(gameTTT, uid, pid)
	b.sendTTTBoard(gm)
}

func (b *Bot) sendTTTBoard(gm *game) {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	kb := tttKeyboard(gm.ID, gm.Board, gm.Turn)
	for _, uid := range []int64{gm.A, gm.B} {
		turn := ""
		if gm.Turn == uid {
			turn = "🎯 <b>Your turn!</b>\n"
		}
		full := fmt.Sprintf("🎮 <b>Tic Tac Toe</b>\n%s%s", turn, tttBoard(gm.Board))
		if mid, ok := gm.MsgIDs[uid]; ok && mid != 0 {
			b.editText(uid, mid, full, &kb)
		} else {
			gm.MsgIDs[uid] = b.sendTextID(uid, full, kb)
		}
	}
}

func tttBoard(board [9]int64) string {
	cell := func(v int64) string {
		switch v {
		case 1:
			return "❌"
		case 2:
			return "⭕"
		default:
			return "⬜"
		}
	}
	return cell(board[0]) + cell(board[1]) + cell(board[2]) + "\n" +
		cell(board[3]) + cell(board[4]) + cell(board[5]) + "\n" +
		cell(board[6]) + cell(board[7]) + cell(board[8])
}

func tttKeyboard(id int64, board [9]int64, turn int64) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for r := 0; r < 3; r++ {
		var row []tgbotapi.InlineKeyboardButton
		for c := 0; c < 3; c++ {
			i := r*3 + c
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(tttCellLabel(board[i]), fmt.Sprintf("ttt:%d:%d", id, i)))
		}
		rows = append(rows, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func tttMark(v int64) string {
	if v == 2 {
		return "⭕"
	}
	return "❌"
}

func tttCellLabel(v int64) string {
	switch v {
	case 1:
		return "❌"
	case 2:
		return "⭕"
	default:
		return "⬜"
	}
}

func (b *Bot) cbTTT(cq *tgbotapi.CallbackQuery) {
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	cell, err := strconv.Atoi(parts[2])
	if err != nil || cell < 0 || cell > 8 {
		b.answerCallback(cq, "", false)
		return
	}
	gm, ok := b.games.get(id)
	if !ok || gm.Done || gm.Kind != gameTTT {
		b.answerCallback(cq, "This game is over.", true)
		return
	}
	if cq.From.ID != gm.A && cq.From.ID != gm.B {
		b.answerCallback(cq, "Not your game.", true)
		return
	}
	b.games.mu.Lock()
	if gm.Turn != cq.From.ID {
		b.games.mu.Unlock()
		b.answerCallback(cq, "Wait for your turn.", true)
		return
	}
	if gm.Board[cell] != 0 {
		b.games.mu.Unlock()
		b.answerCallback(cq, "That cell is taken.", true)
		return
	}
	mark := int64(1)
	if cq.From.ID == gm.B {
		mark = 2
	}
	gm.Board[cell] = mark
	winner := tttWinner(gm.Board)
	draw := tttFull(gm.Board)
	if winner != 0 || draw {
		gm.Done = true
	}
	if !gm.Done {
		if gm.Turn == gm.A {
			gm.Turn = gm.B
		} else {
			gm.Turn = gm.A
		}
	}
	b.games.mu.Unlock()
	b.answerCallback(cq, "", false)

	if winner != 0 {
		winID := gm.A
		if winner == 2 {
			winID = gm.B
		}
		u, _ := b.st.GetUser(winID)
		winName := displayName(storeUserView{ID: u.ID, FirstName: u.FirstName, Username: u.Username})
		b.finishGame(gm, fmt.Sprintf(GameTTTWin, winName, tttMark(winner)))
		return
	}
	if draw {
		b.finishGame(gm, "🤝 It's a draw!")
		return
	}
	b.sendTTTBoard(gm)
}

func tttWinner(b [9]int64) int64 {
	lines := [8][3]int{
		{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
		{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
		{0, 4, 8}, {2, 4, 6},
	}
	for _, l := range lines {
		if b[l[0]] != 0 && b[l[0]] == b[l[1]] && b[l[1]] == b[l[2]] {
			return b[l[0]]
		}
	}
	return 0
}

func tttFull(b [9]int64) bool {
	for _, v := range b {
		if v == 0 {
			return false
		}
	}
	return true
}
