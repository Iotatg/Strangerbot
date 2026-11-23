package bot

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

var (
	phoneCandidate = regexp.MustCompile(`(?:\+?\d[\d\s\-().]{6,}\d)`)
	emailRe        = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	inviteRe       = regexp.MustCompile(`(?i)(?:t|telegram)\.me/\+`)
	deeplinkRe     = regexp.MustCompile(`(?i)\btg://`)
	scamRe         = regexp.MustCompile(`(?i)\b(free\s+(?:bitcoin|crypto|money|airdrop|gift\s*cards?)|double\s+your|guaranteed\s+(?:profit|return|income)|investment\s+opportunity|forex|casino|betting|click\s+here|whatsapp\s+me|earn\s+usd|make\s+money\s+fast|send\s+me\s+your\s+(?:password|otp))\b`)
)

// verdict is the result of scanning a message body.
type verdict struct {
	blocked  bool
	reason   string
	severity int
}

// hasPhone reports whether text contains something that looks like a phone
// number (at least nine digits once separators are removed).
func hasPhone(text string) bool {
	for _, cand := range phoneCandidate.FindAllString(text, -1) {
		digits := 0
		for _, r := range cand {
			if r >= '0' && r <= '9' {
				digits++
			}
		}
		if digits >= 9 {
			return true
		}
	}
	return false
}

// scanContent blocks contact details, private invites and obvious scams.
func scanContent(text string, forwarded bool) verdict {
	if forwarded {
		return verdict{blocked: true, reason: "forwarded messages are not allowed", severity: 1}
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return verdict{}
	}
	switch {
	case inviteRe.MatchString(t) || deeplinkRe.MatchString(t):
		return verdict{blocked: true, reason: "private invite links are not allowed", severity: 3}
	case scamRe.MatchString(t):
		return verdict{blocked: true, reason: "suspicious promotional content", severity: 2}
	case hasPhone(t):
		return verdict{blocked: true, reason: "phone numbers are not allowed", severity: 1}
	case emailRe.MatchString(t):
		return verdict{blocked: true, reason: "email addresses are not allowed", severity: 1}
	}
	return verdict{}
}

type dupRecord struct {
	text  string
	count int
	last  time.Time
}

type spamTracker struct {
	mu     sync.Mutex
	dup    map[int64]dupRecord
	search map[int64]time.Time
}

func newSpamTracker() *spamTracker {
	return &spamTracker{dup: map[int64]dupRecord{}, search: map[int64]time.Time{}}
}

// duplicate reports whether the same non-empty text was sent repeatedly inside
// the lookback window.
func (s *spamTracker) duplicate(uid int64, text string, max int) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	r, ok := s.dup[uid]
	if !ok || now.Sub(r.last) > time.Minute {
		s.dup[uid] = dupRecord{text: text, count: 1, last: now}
		return false
	}
	if r.text == text {
		r.count++
		s.dup[uid] = r
		return r.count > max
	}
	s.dup[uid] = dupRecord{text: text, count: 1, last: now}
	return false
}

// searchAllowed enforces a minimum gap between searches.
func (s *spamTracker) searchAllowed(uid int64, cooldown time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.search[uid]
	if ok && time.Since(last) < cooldown {
		return false
	}
	s.search[uid] = time.Now()
	return true
}

type auditEntry struct {
	at     time.Time
	admin  int64
	action string
}

// audit records an admin action in a bounded in-memory ring.
func (b *Bot) audit(admin int64, action string) {
	b.auditMu.Lock()
	defer b.auditMu.Unlock()
	b.auditLog = append(b.auditLog, auditEntry{at: time.Now().UTC(), admin: admin, action: action})
	if len(b.auditLog) > 100 {
		b.auditLog = b.auditLog[len(b.auditLog)-100:]
	}
}

// strike records a violation and escalates: warn, mute, then ban.
func (b *Bot) strike(uid int64, reason string) {
	n := b.st.AddStrike(uid)
	switch {
	case n >= b.cfg.MaxStrikes:
		b.st.BanUser(uid, reason)
		b.endChat(uid, false)
		b.sendText(uid, fmt.Sprintf(Banned, escapeHTML(reason), b.cfg.SupportLink), nil)
		b.sendToAdmins(fmt.Sprintf("🚨 <b>Auto-banned</b> <code>%d</code>: %s (strikes %d)", uid, escapeHTML(reason), n))
		b.log.Warn("auto-ban", "user", uid, "reason", reason, "strikes", n)
	case n >= 2:
		until := time.Now().UTC().Add(time.Duration(b.cfg.MuteMinutes) * time.Minute)
		b.st.SetMuted(uid, until.Format(time.RFC3339))
		b.endChat(uid, false)
		b.sendText(uid, MutedNotice, nil)
		b.log.Warn("auto-mute", "user", uid, "reason", reason, "strikes", n)
	default:
		b.sendText(uid, WarningNotice, nil)
		b.log.Warn("warning", "user", uid, "reason", reason, "strikes", n)
	}
}

// needCaptcha reports whether the user still has to pass the human check.
func (b *Bot) needCaptcha(uid int64) bool {
	if !b.cfg.CaptchaRequired || b.cfg.IsAdmin(uid) {
		return true
	}
	return b.st.IsVerified(uid)
}

// requireCaptcha sends the verification prompt and reports whether the caller
// may continue.
func (b *Bot) requireCaptcha(uid int64) bool {
	if b.needCaptcha(uid) {
		return true
	}
	b.sendText(uid, CaptchaPrompt, CaptchaButton())
	return false
}

func (b *Bot) cbCaptcha(cq *tgbotapi.CallbackQuery) {
	b.st.SetVerified(cq.From.ID, true)
	b.answerCallback(cq, "Verified ✅", true)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, CaptchaDone, nil)
	}
}

func (b *Bot) sendSafety(uid int64) {
	b.sendText(uid, SafetyNotice, nil)
}

func (b *Bot) cmdSecurity(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	text := fmt.Sprintf(SecurityStatus,
		b.cfg.MaxStrikes, b.cfg.MuteMinutes, b.cfg.AutoBanReports,
		onOff(b.cfg.CaptchaRequired), onOff(b.cfg.ProtectContent), onOff(b.cfg.BlockForwards),
		b.cfg.SearchCooldownSeconds, b.cfg.PrivacyDeleteMinutes)
	b.sendText(m.From.ID, text, nil)
}

func onOff(v bool) string {
	if v {
		return "✅ on"
	}
	return "off"
}

func (b *Bot) cmdStrikes(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/strikes &lt;id&gt;</code> or <code>/resetstrikes &lt;id&gt;</code>", nil)
		return
	}
	uid := fields[0]
	if len(fields) > 1 && fields[1] == "reset" {
		b.st.ResetStrikes(parseID(uid))
		b.audit(m.From.ID, "reset strikes for "+uid)
		b.sendText(m.From.ID, "✅ Strikes reset.", nil)
		return
	}
	u, ok := b.st.GetUser(parseID(uid))
	if !ok {
		b.sendText(m.From.ID, "❗️ User not found.", nil)
		return
	}
	b.sendText(m.From.ID, fmt.Sprintf(StrikesInfo, u.ID, u.Strikes, b.cfg.MaxStrikes), nil)
}

func parseID(s string) int64 {
	var id int64
	fmt.Sscan(s, &id)
	return id
}

// sendSafetyOnce shows the safety notice only the first time.
func (b *Bot) sendSafetyOnce(uid int64) {
	b.safetyMu.Lock()
	if b.safetyShown[uid] {
		b.safetyMu.Unlock()
		return
	}
	b.safetyShown[uid] = true
	b.safetyMu.Unlock()
	b.sendSafety(uid)
}

// cmdResetStrikes clears a user's violation counter.
func (b *Bot) cmdResetStrikes(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/resetstrikes &lt;id&gt;</code>", nil)
		return
	}
	uid := parseID(fields[0])
	b.st.ResetStrikes(uid)
	b.audit(m.From.ID, "reset strikes for "+fields[0])
	b.sendText(m.From.ID, "✅ Strikes reset.", nil)
}

func (b *Bot) cmdAudit(m *tgbotapi.Message) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	b.auditMu.Lock()
	entries := append([]auditEntry(nil), b.auditLog...)
	b.auditMu.Unlock()
	if len(entries) == 0 {
		b.sendText(m.From.ID, "No admin actions recorded yet.", nil)
		return
	}
	var sb strings.Builder
	sb.WriteString("🧾 <b>Recent admin actions</b>\n\n")
	for i := len(entries) - 1; i >= 0 && len(sb.String()) < 3500; i-- {
		e := entries[i]
		sb.WriteString(fmt.Sprintf("• %s — <code>%d</code>: %s\n", e.at.Format("15:04"), e.admin, escapeHTML(e.action)))
	}
	b.sendText(m.From.ID, sb.String(), nil)
}

func (b *Bot) cmdUnmute(m *tgbotapi.Message, args string) {
	if !b.requireAdmin(m.From.ID) {
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(m.From.ID, "Usage: <code>/unmute &lt;id&gt;</code>", nil)
		return
	}
	uid := parseID(fields[0])
	b.st.SetMuted(uid, "")
	b.sendText(m.From.ID, "✅ User unmuted.", nil)
}

// cmdSafety shows the community rules to any user.
func (b *Bot) cmdSafety(m *tgbotapi.Message) {
	b.sendSafety(m.From.ID)
}

// afterReport applies automatic moderation after a report is filed.
func (b *Bot) afterReport(reported int64, r store.Report) {
	if b.cfg.AutoBanReports <= 0 {
		return
	}
	if b.st.OpenReportsAgainst(reported) < b.cfg.AutoBanReports {
		return
	}
	b.st.BanUser(reported, "Repeatedly reported by users")
	pid, _ := b.st.RemovePair(reported)
	b.st.RemoveWaiting(reported)
	b.sendText(reported, fmt.Sprintf(Banned, "Repeated user reports", b.cfg.SupportLink), nil)
	if pid != 0 {
		b.sendText(pid, ChatEndedPartner, MainMenu())
	}
	b.sendToAdmins(fmt.Sprintf("🚨 <b>Auto-banned</b> <code>%d</code> after %d reports (latest #%d)", reported, b.cfg.AutoBanReports, r.ID))
}
