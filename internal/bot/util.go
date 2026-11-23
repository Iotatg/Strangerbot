package bot

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

// escapeHTML escapes the three characters that matter for Telegram HTML parse
// mode (matching Python's html.escape(quote=False)).
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

var (
	urlRe     = regexp.MustCompile(`(?i)(?:https?://|t\.me/|telegram\.me/|te\.legra\.ph/|www\.)`)
	domainRe  = regexp.MustCompile(`(?i)\b[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+(?:[/?#]\S*)?\b`)
	mentionRe = regexp.MustCompile(`@[a-zA-Z0-9_]{4,32}`)
)

// containsLinkOrMention reports whether the message text/entities carry a link
// or a @mention that would let strangers find each other.
func containsLinkOrMention(m *tgbotapi.Message) bool {
	if m == nil {
		return false
	}
	entities := append(append([]tgbotapi.MessageEntity{}, m.Entities...), m.CaptionEntities...)
	for _, e := range entities {
		switch e.Type {
		case "url", "text_link", "mention", "text_mention":
			return true
		}
	}
	text := m.Text + " " + m.Caption
	if mentionRe.MatchString(text) {
		return true
	}
	if urlRe.MatchString(text) {
		return true
	}
	return domainRe.MatchString(text)
}

type msgKey struct {
	chat int64
	mid  int64
}

// MessageMap remembers relayed message ids so replies can be threaded and
// /del can remove a message on both sides. It is bounded and evicts oldest.
type MessageMap struct {
	mu        sync.Mutex
	limit     int
	sent      map[msgKey]msgKey
	recv      map[msgKey]msgKey
	sentOrder []msgKey
	recvOrder []msgKey
}

// NewMessageMap creates a bounded map.
func NewMessageMap(limit int) *MessageMap {
	return &MessageMap{
		limit: limit,
		sent:  make(map[msgKey]msgKey),
		recv:  make(map[msgKey]msgKey),
	}
}

func (m *MessageMap) put(dst map[msgKey]msgKey, order *[]msgKey, k, v msgKey) {
	if _, exists := dst[k]; exists {
		dst[k] = v
		return
	}
	dst[k] = v
	*order = append(*order, k)
	for len(*order) > m.limit {
		old := (*order)[0]
		*order = (*order)[1:]
		delete(dst, old)
	}
}

// Record links a sender message to the receiver copy.
func (m *MessageMap) Record(sender, senderMid, receiver, receiverMid int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k1 := msgKey{sender, senderMid}
	v1 := msgKey{receiver, receiverMid}
	k2 := msgKey{receiver, receiverMid}
	v2 := msgKey{sender, senderMid}
	m.put(m.sent, &m.sentOrder, k1, v1)
	m.put(m.recv, &m.recvOrder, k2, v2)
}

// OwnMessage returns the partner copy if uid sent msg mid.
func (m *MessageMap) OwnMessage(uid, mid int64) (msgKey, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.sent[msgKey{uid, mid}]
	return v, ok
}

// ReplyTarget resolves the message to reply to when uid replies to repliedMid.
func (m *MessageMap) ReplyTarget(uid, repliedMid int64) (msgKey, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := msgKey{uid, repliedMid}
	if v, ok := m.sent[k]; ok {
		return v, true
	}
	if v, ok := m.recv[k]; ok {
		return v, true
	}
	return msgKey{}, false
}

// Forget drops both directions of a mapping.
func (m *MessageMap) Forget(uid, mid int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := msgKey{uid, mid}
	if v, ok := m.sent[k]; ok {
		delete(m.sent, k)
		delete(m.recv, v)
	}
}

type rlWindow struct {
	count int
	start time.Time
}

// rateLimiter is a per-user fixed-window limiter.
type rateLimiter struct {
	mu       sync.Mutex
	limit    int
	interval time.Duration
	windows  map[int64]rlWindow
}

func newRateLimiter(limit int, interval time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, interval: interval, windows: map[int64]rlWindow{}}
}

func (r *rateLimiter) allow(uid int64) bool {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.windows[uid]
	if !ok || now.Sub(w.start) > r.interval {
		r.windows[uid] = rlWindow{count: 1, start: now}
		return true
	}
	if w.count >= r.limit {
		return false
	}
	w.count++
	r.windows[uid] = w
	return true
}

type state struct {
	Step         string
	Age          int
	AgeMin       int
	AgeMax       int
	Gender       string
	ReportTarget int64
}

type stateStore struct {
	mu sync.Mutex
	m  map[int64]*state
}

func newStateStore() *stateStore {
	return &stateStore{m: map[int64]*state{}}
}

func (s *stateStore) get(uid int64) *state {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[uid]
	if !ok {
		st = &state{}
		s.m[uid] = st
	}
	return st
}

func (s *stateStore) setStep(uid int64, step string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[uid]
	if !ok {
		st = &state{}
		s.m[uid] = st
	}
	st.Step = step
}

func (s *stateStore) clear(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, uid)
}

// notifyAdminsOfReport DMs every admin about a new report.
func (b *Bot) notifyAdminsOfReport(r store.Report) {
	if !b.cfg.NotifyReport {
		return
	}
	text := "🚨 <b>New report #" + itoa(r.ID) + "</b>\n" +
		"Reporter: <code>" + itoa(r.Reporter) + "</code>\n" +
		"Reported: <code>" + itoa(r.Reported) + "</code>\n" +
		"Reason: <i>" + escapeHTML(r.Reason) + "</i>"
	kb := AdminReportBtns(r.ID, r.Reported)
	for _, admin := range b.cfg.AdminIDs {
		cfg := tgbotapi.NewMessage(admin, text)
		cfg.ParseMode = tgbotapi.ModeHTML
		cfg.ReplyMarkup = kb
		if _, err := b.api.Send(cfg); err != nil {
			b.log.Warn("notify admin failed", "admin", admin, "err", err)
		}
	}
}

// sendToAdmin DMs all admins a plain message.
func (b *Bot) sendToAdmins(text string) {
	for _, admin := range b.cfg.AdminIDs {
		cfg := tgbotapi.NewMessage(admin, text)
		cfg.ParseMode = tgbotapi.ModeHTML
		if _, err := b.api.Send(cfg); err != nil {
			b.log.Warn("admin dm failed", "admin", admin, "err", err)
		}
	}
}

// sendText sends an HTML message to a chat and logs failures.
func (b *Bot) sendText(chatID int64, text string, markup interface{}) {
	cfg := tgbotapi.NewMessage(chatID, text)
	cfg.ParseMode = tgbotapi.ModeHTML
	cfg.ReplyMarkup = markup
	if _, err := b.api.Send(cfg); err != nil {
		b.log.Warn("send failed", "chat", chatID, "err", err)
	}
}

// sendTextID sends an HTML message and returns its message id (0 on failure).
func (b *Bot) sendTextID(chatID int64, text string, markup interface{}) int {
	cfg := tgbotapi.NewMessage(chatID, text)
	cfg.ParseMode = tgbotapi.ModeHTML
	cfg.ReplyMarkup = markup
	msg, err := b.api.Send(cfg)
	if err != nil {
		b.log.Warn("send failed", "chat", chatID, "err", err)
		return 0
	}
	return msg.MessageID
}

// editText edits a message with HTML parse mode.
func (b *Bot) editText(chatID int64, messageID int, text string, markup *tgbotapi.InlineKeyboardMarkup) {
	cfg := tgbotapi.EditMessageTextConfig{
		BaseEdit:  tgbotapi.BaseEdit{ChatID: chatID, MessageID: messageID, ReplyMarkup: markup},
		Text:      text,
		ParseMode: tgbotapi.ModeHTML,
	}
	if _, err := b.api.Send(cfg); err != nil {
		b.log.Debug("edit failed", "chat", chatID, "msg", messageID, "err", err)
	}
}

// answerCallback answers a callback query.
func (b *Bot) answerCallback(cq *tgbotapi.CallbackQuery, text string, alert bool) {
	cfg := tgbotapi.CallbackConfig{CallbackQueryID: cq.ID, Text: text, ShowAlert: alert}
	if _, err := b.api.Send(cfg); err != nil {
		b.log.Debug("answer callback failed", "err", err)
	}
}

// dailyLimit returns the user's daily search allowance.
func (b *Bot) dailyLimit(premium bool) int {
	if premium {
		return b.cfg.PremiumDailySearches
	}
	return b.cfg.FreeDailySearches
}

// isPremium reports premium access (always true if premium is disabled).
func (b *Bot) isPremium(u store.User) bool {
	if !premiumEnabled {
		return true
	}
	return u.Premium
}

// premiumEnabled mirrors config.PREMIUM_ENABLED from the source.
var premiumEnabled = true

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
