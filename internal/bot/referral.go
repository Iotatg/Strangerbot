package bot

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "strangerbot/internal/tgapi"
)

// parseReferral extracts a referrer id from a /start deep-link payload.
func parseReferral(args string) (int64, bool) {
	args = strings.TrimSpace(args)
	if !strings.HasPrefix(args, "ref_") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(args, "ref_"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// handleReferral rewards both sides the first time a referral is recorded.
func (b *Bot) handleReferral(uid, referrer int64) {
	ref, ok := b.st.AttachReferral(uid, referrer)
	if !ok {
		return
	}
	b.st.AddBonusSearches(uid, 2)
	b.st.AddBonusSearches(ref, b.cfg.ReferralReward)
	b.sendText(uid, ReferralWelcome, nil)
	b.sendText(ref, fmt.Sprintf(ReferralRewarded, b.cfg.ReferralReward), nil)
}

// cmdReferral shows the caller's invite link and referral count.
func (b *Bot) cmdReferral(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	link := fmt.Sprintf("https://t.me/%s?start=ref_%d", b.api.Self.UserName, uid)
	count := b.st.CountReferrals(uid)
	b.sendText(uid, fmt.Sprintf(ReferralCard, escapeHTML(u.FirstName), b.cfg.ReferralReward, link, count), nil)
}
