package bot

import (
	"fmt"
	"math/rand"
	"strings"

	tgbotapi "strangerbot/internal/tgapi"
)

var eightBallAnswers = []string{
	"It is certain.", "It is decidedly so.", "Without a doubt.", "Yes — definitely.",
	"You may rely on it.", "As I see it, yes.", "Most likely.", "Outlook good.",
	"Yes.", "Signs point to yes.", "Reply hazy, try again.", "Ask again later.",
	"Better not tell you now.", "Cannot predict now.", "Concentrate and ask again.",
	"Don't count on it.", "My reply is no.", "My sources say no.",
	"Outlook not so good.", "Very doubtful.",
}

func displayName(u storeUserView) string {
	name := u.FirstName
	if name == "" {
		name = u.Username
	}
	if name == "" {
		return fmt.Sprintf("<code>%d</code>", u.ID)
	}
	if u.Username != "" {
		return escapeHTML(name) + " (@" + escapeHTML(u.Username) + ")"
	}
	return escapeHTML(name)
}

type storeUserView struct {
	ID        int64
	FirstName string
	Username  string
}

func (b *Bot) cmdLeaderboard(m *tgbotapi.Message) {
	uid := m.From.ID
	top := b.st.TopChatters(10)
	if len(top) == 0 {
		b.sendText(uid, LeaderboardEmpty, nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(LeaderboardHeader)
	for i, u := range top {
		if u.TotalChats <= 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s <b>%s</b> — <b>%d</b> %s\n", medal(i), displayName(storeUserView{
			ID: u.ID, FirstName: u.FirstName, Username: u.Username,
		}), u.TotalChats, chatsWord(u.TotalChats)))
	}
	b.sendText(uid, sb.String(), nil)
}

func (b *Bot) cmdRank(m *tgbotapi.Message) {
	uid := m.From.ID
	rank, total := b.st.Rank(uid)
	if rank == 0 {
		b.sendText(uid, RankNone, nil)
		return
	}
	b.sendText(uid, fmt.Sprintf(RankOK, rank, total), nil)
}

func (b *Bot) cmdPublicStats(m *tgbotapi.Message) {
	s := b.st.Stats()
	b.sendText(m.From.ID, fmt.Sprintf(PublicStats, s.TotalUsers, s.Registered, s.ActiveChats, s.Waiting, s.Premium), nil)
}

func (b *Bot) cmdEightBall(m *tgbotapi.Message, args string) {
	_ = args
	b.sendText(m.From.ID, fmt.Sprintf(EightBallTemplate, eightBallAnswers[rand.Intn(len(eightBallAnswers))]), nil)
}

func (b *Bot) cmdDice(m *tgbotapi.Message) {
	b.sendText(m.From.ID, fmt.Sprintf(DiceTemplate, rand.Intn(6)+1), nil)
}

func (b *Bot) cmdCoin(m *tgbotapi.Message) {
	face := "Heads"
	if rand.Intn(2) == 1 {
		face = "Tails"
	}
	b.sendText(m.From.ID, fmt.Sprintf(CoinTemplate, face), nil)
}

func medal(i int) string {
	switch i {
	case 0:
		return "🥇"
	case 1:
		return "🥈"
	case 2:
		return "🥉"
	default:
		return fmt.Sprintf("%d.", i+1)
	}
}

func chatsWord(n int) string {
	if n == 1 {
		return "chat"
	}
	return "chats"
}

func (b *Bot) cmdChoose(m *tgbotapi.Message, args string) {
	parts := strings.FieldsFunc(args, func(r rune) bool { return r == ',' })
	if len(parts) == 1 {
		parts = strings.Fields(args)
	}
	if len(parts) < 2 {
		b.sendText(m.From.ID, "Usage: <code>/choose option1, option2, option3</code>", nil)
		return
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	b.sendText(m.From.ID, fmt.Sprintf(ChooseTemplate, escapeHTML(parts[rand.Intn(len(parts))])), nil)
}
