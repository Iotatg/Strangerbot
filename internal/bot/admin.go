package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"strangerbot/internal/config"
	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

func (b *Bot) requireAdmin(uid int64) bool {
	if b.cfg.IsAdmin(uid) {
		return true
	}
	b.sendText(uid, AdminNotAdmin, nil)
	return false
}

func (b *Bot) adminStats(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	s := b.st.Stats()
	b.sendText(m.From.ID, fmt.Sprintf(AdminStats, s.TotalUsers, s.Registered, s.ActiveChats, s.Waiting, s.Premium, s.OpenReports, s.TotalReports, s.Banned), nil)
}

func (b *Bot) adminBan(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/ban &lt;id&gt; [reason]</code>", nil)
		return
	}
	tid, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		b.sendText(m.From.ID, "Usage: <code>/ban &lt;id&gt; [reason]</code>", nil)
		return
	}
	reason := "Violation of rules"
	if len(fields) > 1 {
		reason = strings.Join(fields[1:], " ")
	}
	b.st.BanUser(tid, reason)
	partner, _ := b.st.RemovePair(tid)
	b.st.RemoveWaiting(tid)
	b.sendText(m.From.ID, fmt.Sprintf("🚫 Banned <code>%d</code>. Reason: <i>%s</i>", tid, escapeHTML(reason)), nil)
	b.sendText(tid, fmt.Sprintf(Banned, escapeHTML(reason), b.cfg.SupportLink), nil)
	if partner != 0 {
		b.sendText(partner, ChatEndedPartner, MainMenu())
	}
}

func (b *Bot) adminUnban(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/unban &lt;id&gt;</code>", nil)
		return
	}
	tid, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		b.sendText(m.From.ID, "Usage: <code>/unban &lt;id&gt;</code>", nil)
		return
	}
	b.st.UnbanUser(tid)
	b.sendText(m.From.ID, fmt.Sprintf("✅ Unbanned <code>%d</code>.", tid), nil)
}

func (b *Bot) adminPremium(m *tgbotapi.Message, args string, grant bool) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		if grant {
			b.sendText(m.From.ID, "Usage: <code>/premium &lt;id&gt;</code>", nil)
		} else {
			b.sendText(m.From.ID, "Usage: <code>/unpremium &lt;id&gt;</code>", nil)
		}
		return
	}
	tid, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		b.sendText(m.From.ID, "Usage: <code>/premium &lt;id&gt;</code>", nil)
		return
	}
	b.st.UpdateUser(tid, func(u *store.User) { u.Premium = grant })
	if grant {
		b.sendText(m.From.ID, fmt.Sprintf("🧸 Premium granted to <code>%d</code>.", tid), nil)
		b.sendText(tid, PremiumGranted, nil)
	} else {
		b.sendText(m.From.ID, fmt.Sprintf("❗️ Premium revoked from <code>%d</code>.", tid), nil)
	}
}

func (b *Bot) adminAddCoin(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) < 2 {
		b.sendText(m.From.ID, "Usage: <code>/addcoin &lt;id&gt; &lt;amount&gt;</code>", nil)
		return
	}
	tid, err1 := strconv.ParseInt(fields[0], 10, 64)
	amount, err2 := strconv.Atoi(fields[1])
	if err1 != nil || err2 != nil {
		b.sendText(m.From.ID, "Usage: <code>/addcoin &lt;id&gt; &lt;amount&gt;</code>", nil)
		return
	}
	b.st.AddBonusSearches(tid, amount)
	u, ok := b.st.GetUser(tid)
	bal := "?"
	if ok {
		bal = strconv.Itoa(u.BonusSearches)
	}
	b.sendText(m.From.ID, fmt.Sprintf("💰 Added %d to <code>%d</code>. Bonus now: <b>%s</b>", amount, tid, bal), nil)
}

func (b *Bot) adminBroadcast(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	text := strings.TrimSpace(args)
	if text == "" {
		b.sendText(m.From.ID, "Usage: <code>/broadcast &lt;message&gt;</code>", nil)
		return
	}
	users := b.st.AllUsers()
	sent, failed := 0, 0
	for _, u := range users {
		if u.Banned {
			continue
		}
		msg := fmt.Sprintf("📣 <b>Announcement</b>\n\n%s", escapeHTML(text))
		cfg := tgbotapi.NewMessage(u.ID, msg)
		cfg.ParseMode = tgbotapi.ModeHTML
		if _, err := b.api.Send(cfg); err != nil {
			failed++
		} else {
			sent++
		}
		time.Sleep(50 * time.Millisecond)
	}
	b.sendText(m.From.ID, fmt.Sprintf(AdminBroadcastDone, sent, failed), nil)
}

func (b *Bot) adminReports(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	open := b.st.GetReports(false)
	if len(open) == 0 {
		b.sendText(m.From.ID, "✅ No open reports.", nil)
		return
	}
	for _, r := range open {
		text := fmt.Sprintf("🚨 <b>Report #%d</b>\nReporter: <code>%d</code>\nReported: <code>%d</code>\nReason: <i>%s</i>\nTime: %s",
			r.ID, r.Reporter, r.Reported, escapeHTML(r.Reason), r.Time)
		b.sendText(m.From.ID, text, AdminReportBtns(r.ID, r.Reported))
	}
}

func (b *Bot) adminResolve(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/resolve &lt;report_number&gt;</code>", nil)
		return
	}
	id, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		b.sendText(m.From.ID, "Usage: <code>/resolve &lt;report_number&gt;</code>", nil)
		return
	}
	if b.st.ResolveReport(id) {
		b.sendText(m.From.ID, fmt.Sprintf("✅ Report #%d resolved.", id), nil)
	} else {
		b.sendText(m.From.ID, "❗️ No report with that number.", nil)
	}
}

func (b *Bot) adminUserInfo(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/userinfo &lt;id&gt;</code>", nil)
		return
	}
	uid, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		b.sendText(m.From.ID, "Usage: <code>/userinfo &lt;id&gt;</code>", nil)
		return
	}
	u, ok := b.st.GetUser(uid)
	if !ok {
		b.sendText(m.From.ID, "❗️ User not found.", nil)
		return
	}
	left := b.st.SearchesLeft(uid)
	uname := u.Username
	if uname == "" {
		uname = "none"
	}
	age := "N/A"
	if u.Age != 0 {
		age = strconv.Itoa(u.Age)
	}
	gender := u.Gender
	if gender == "" {
		gender = "N/A"
	}
	country := u.Country
	if country == "" {
		country = "N/A"
	}
	text := fmt.Sprintf("👤 <b>User Info</b>\n\n"+
		"ID: <code>%d</code>\nUsername: @%s\nName: %s\nAge: %s\nGender: %s\nCountry: %s\n"+
		"Interests: %s\nRegistered: %v\nPremium: %v\nSearches left: %d\nBonus: %d\n"+
		"Total chats: %d\nLikes: %d | Dislikes: %d\nBanned: %v — %s\nJoined: %s",
		u.ID, uname, escapeHTML(u.FirstName), age, gender, escapeHTML(country),
		config.LabelList(u.Interests), u.Registered, u.Premium, left, u.BonusSearches,
		u.TotalChats, u.Likes, u.Dislikes, u.Banned, escapeHTML(u.BanReason), u.Joined)
	b.sendText(m.From.ID, text, nil)
}

func (b *Bot) cbAdminResolve(cq *tgbotapi.CallbackQuery) {
	if !b.requireAdmin(cq.From.ID) {
		b.answerCallback(cq, "Not authorised.", true)
		return
	}
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 2 {
		b.answerCallback(cq, "", false)
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	b.st.ResolveReport(id)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, cq.Message.Text+"\n\n✅ Resolved", nil)
	}
	b.answerCallback(cq, "Resolved!", false)
}

func (b *Bot) cbAdminBan(cq *tgbotapi.CallbackQuery) {
	if !b.requireAdmin(cq.From.ID) {
		b.answerCallback(cq, "Not authorised.", true)
		return
	}
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	tid, err1 := strconv.ParseInt(parts[1], 10, 64)
	rid, err2 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil {
		b.answerCallback(cq, "", false)
		return
	}
	b.st.BanUser(tid, "Reported by users")
	partner, _ := b.st.RemovePair(tid)
	b.st.RemoveWaiting(tid)
	b.st.ResolveReport(rid)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, cq.Message.Text+fmt.Sprintf("\n\n🚫 User %d banned & resolved.", tid), nil)
	}
	b.answerCallback(cq, "Banned!", false)
	b.sendText(tid, fmt.Sprintf(Banned, "Reported by users", b.cfg.SupportLink), nil)
	if partner != 0 {
		b.sendText(partner, ChatEndedPartner, MainMenu())
	}
}
