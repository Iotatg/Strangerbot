package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "strangerbot/internal/tgapi"
)

// cmdMaintenance toggles maintenance mode (admins only).
func (b *Bot) cmdMaintenance(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	switch normalizeToggle(strings.TrimSpace(args)) {
	case "on":
		b.cfg.Maintenance = true
		b.sendText(m.From.ID, "🛠 Maintenance mode <b>enabled</b>.", nil)
	case "off":
		b.cfg.Maintenance = false
		b.sendText(m.From.ID, "✅ Maintenance mode <b>disabled</b>.", nil)
	default:
		state := "off"
		if b.cfg.Maintenance {
			state = "on"
		}
		b.sendText(m.From.ID, "Maintenance mode: <b>"+state+"</b>", nil)
	}
}

// cmdPurge drops every report.
func (b *Bot) cmdPurge(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	n := b.st.ClearReports()
	b.sendText(m.From.ID, fmt.Sprintf("🧹 Purged <b>%d</b> report(s).", n), nil)
}

// cmdMassban bans a comma separated list of user ids.
func (b *Bot) cmdMassban(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.FieldsFunc(args, func(r rune) bool { return r == ',' || r == ' ' })
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/massban &lt;id,id,...&gt; [reason]</code>", nil)
		return
	}
	banned, invalid := 0, 0
	var reason string
	ids := make([]int64, 0, len(fields))
	for _, f := range fields {
		id, err := strconv.ParseInt(strings.TrimSpace(f), 10, 64)
		if err != nil {
			invalid++
			continue
		}
		ids = append(ids, id)
	}
	if len(fields) > len(ids) {
		reason = strings.Join(fields[len(ids):], " ")
	}
	for _, id := range ids {
		if id < 0 {
			invalid++
			continue
		}
		b.st.BanUser(id, reason)
		b.sendText(id, fmt.Sprintf(Banned, escapeHTML(reason), b.cfg.SupportLink), nil)
		banned++
	}
	b.sendText(m.From.ID, fmt.Sprintf("🚫 Banned <b>%d</b> user(s), <b>%d</b> invalid.", banned, invalid), nil)
}

// cmdBroadcastNow sends the daily digest immediately.
func (b *Bot) cmdDigestNow(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	b.sendDigest()
	b.sendText(m.From.ID, "📊 Digest sent.", nil)
}

// digestLoop sends the daily stats digest to admins once a day.
func (b *Bot) digestLoop() {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	last := ""
	for range t.C {
		now := time.Now().UTC()
		day := now.Format("2006-01-02")
		if now.Hour() == b.cfg.DigestHour && last != day {
			last = day
			b.sendDigest()
		}
	}
}

func (b *Bot) sendDigest() {
	s := b.st.Stats()
	text := fmt.Sprintf(DigestTemplate,
		s.TotalUsers, s.Registered, s.ActiveChats, s.Waiting, s.Premium, s.Banned,
		s.OpenReports, s.TotalReports)
	b.sendToAdmins(text)
}

// premiumLoop revokes expired premium and notifies the user.
func (b *Bot) premiumLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for range t.C {
		for _, uid := range b.st.ExpirePremium(time.Now().UTC()) {
			b.sendText(uid, PremiumExpired, MainMenu())
		}
	}
}
