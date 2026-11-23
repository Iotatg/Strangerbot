package bot

import (
	"fmt"
	"strings"

	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

// trustBar renders a 10-cell ASCII progress bar.
func trustBar(score int) string {
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	filled := (score + 5) / 10
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

// trustLevel maps a score to a human label.
func trustLevel(score int) string {
	switch {
	case score >= 85:
		return "Trusted ⭐⭐⭐"
	case score >= 70:
		return "Reliable ⭐⭐"
	case score >= 50:
		return "Fair ⭐"
	case score >= 30:
		return "Low trust ⚠️"
	default:
		return "Risky 🚫"
	}
}

func (b *Bot) cmdTrust(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	score := store.TrustScore(u)
	open := b.st.CountOpenReportsAgainst(uid)
	b.sendText(uid, fmt.Sprintf(TrustCard,
		score, trustBar(score), trustLevel(score),
		u.TotalChats, u.Likes, u.Dislikes, open, u.ReferralCount), nil)
}

func (b *Bot) adminTrustlist(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	top := b.st.TopTrust(15)
	if len(top) == 0 {
		b.sendText(m.From.ID, "No users yet.", nil)
		return
	}
	var sb strings.Builder
	sb.WriteString("🏆 <b>Trust leaderboard</b>\n\n")
	for i, u := range top {
		score := store.TrustScore(u)
		sb.WriteString(fmt.Sprintf("%d. <b>%s</b> — %d %s\n", i+1, displayName(storeUserView{
			ID: u.ID, FirstName: u.FirstName, Username: u.Username,
		}), score, trustBar(score)))
	}
	b.sendText(m.From.ID, sb.String(), nil)
}
