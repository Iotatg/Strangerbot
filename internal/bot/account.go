package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"strangerbot/internal/config"

	tgbotapi "strangerbot/internal/tgapi"
)

// cmdMyData shows everything the bot stores about the caller.
func (b *Bot) cmdMyData(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	text := fmt.Sprintf(MyData,
		u.ID, escapeHTML(u.FirstName), u.Username, u.Age, u.Gender, escapeHTML(u.Country),
		config.LabelList(u.Interests), u.TotalChats, u.Likes, u.Dislikes, u.TotalReports,
		u.ReferredBy, u.ReferralCount, u.Joined)
	b.sendText(uid, text, AccountButtons())
}

// cmdDeleteAccount starts the account deletion confirmation.
func (b *Bot) cmdDeleteAccount(m *tgbotapi.Message) {
	uid := m.From.ID
	if _, ok := b.st.GetUser(uid); !ok {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	b.sendText(uid, DeleteAccountPrompt, DeleteConfirm())
}

func (b *Bot) cbDeleteAccount(cq *tgbotapi.CallbackQuery) {
	uid := cq.From.ID
	partner, ok := b.st.DeleteUser(uid)
	if ok && partner != 0 {
		b.sendText(partner, ChatEndedPartner, MainMenu())
	}
	b.states.clear(uid)
	b.activityMu.Lock()
	delete(b.activity, uid)
	b.activityMu.Unlock()
	b.answerCallback(cq, "Account deleted.", true)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, DeleteAccountDone, nil)
	}
}

func (b *Bot) cbDeleteCancel(cq *tgbotapi.CallbackQuery) {
	b.answerCallback(cq, "Cancelled.", false)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, "Cancelled.", nil)
	}
}

// cmdGift lets a premium user gift premium days to someone else.
func (b *Bot) cmdGift(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	me, ok := b.st.GetUser(uid)
	if !ok || !me.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	if !me.Premium {
		b.sendText(uid, GiftPremiumOnly, nil)
		return
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		b.sendText(uid, "Usage: <code>/gift &lt;user_id&gt;</code>", nil)
		return
	}
	target, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || target == uid {
		b.sendText(uid, "Usage: <code>/gift &lt;user_id&gt;</code>", nil)
		return
	}
	if _, ok := b.st.GetUser(target); !ok {
		b.sendText(uid, "❗️ That user is not known to the bot.", nil)
		return
	}
	until := time.Now().UTC().Add(time.Duration(b.cfg.GiftPremiumDays) * 24 * time.Hour)
	b.st.SetPremium(target, true, until.Format(time.RFC3339))
	b.sendText(uid, fmt.Sprintf(GiftSent, b.cfg.GiftPremiumDays, target), nil)
	b.sendText(target, fmt.Sprintf(GiftReceived, escapeHTML(me.FirstName), b.cfg.GiftPremiumDays), MainMenu())
}

// cmdAppeal lets a banned user ask an admin for a second chance.
func (b *Bot) cmdAppeal(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	if !b.st.IsBanned(uid) {
		b.sendText(uid, NotBanned, nil)
		return
	}
	reason := strings.TrimSpace(args)
	if reason == "" {
		b.sendText(uid, AppealUsage, nil)
		return
	}
	text := fmt.Sprintf(AppealAdmin, uid, escapeHTML(reason))
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Unban", fmt.Sprintf("appeal:unban:%d", uid)),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Keep ban", fmt.Sprintf("appeal:keep:%d", uid)),
		),
	)
	for _, admin := range b.cfg.AdminIDs {
		cfg := tgbotapi.NewMessage(admin, text)
		cfg.ParseMode = tgbotapi.ModeHTML
		cfg.ReplyMarkup = kb
		if _, err := b.api.Send(cfg); err != nil {
			b.log.Warn("appeal notify failed", "admin", admin, "err", err)
		}
	}
	b.sendText(uid, AppealSent, nil)
}

func (b *Bot) cbAppeal(cq *tgbotapi.CallbackQuery, unban bool) {
	if !b.requireAdmin(cq.From.ID) {
		b.answerCallback(cq, "Not authorised.", true)
		return
	}
	parts := strings.Split(cq.Data, ":")
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	uid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	if unban {
		b.st.UnbanUser(uid)
		b.sendText(uid, AppealAccepted, MainMenu())
	}
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, cq.Message.Text+"\n\n"+
			map[bool]string{true: "✅ Unbanned", false: "🚫 Ban kept"}[unban], nil)
	}
	b.answerCallback(cq, "Done", false)
}
