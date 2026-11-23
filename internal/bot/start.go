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

func (b *Bot) cmdStart(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	b.states.clear(uid)
	u := b.st.EnsureUser(uid, m.From.UserName, m.From.FirstName)
	if refID, ok := parseReferral(args); ok {
		b.handleReferral(uid, refID)
	}
	if u.Registered {
		b.sendText(uid, fmt.Sprintf(WelcomeBack, b.cfg.BotName), MainMenu())
		return
	}
	b.sendText(uid, Welcome, nil)
	b.sendText(uid, fmt.Sprintf(AskAge, b.cfg.MinAge, b.cfg.MaxAge), nil)
	b.states.setStep(uid, "reg_age")
}

func (b *Bot) cmdHelp(m *tgbotapi.Message) {
	b.sendText(m.From.ID, Help, nil)
}

func (b *Bot) cmdAbout(m *tgbotapi.Message) {
	b.sendText(m.From.ID, fmt.Sprintf(About, b.cfg.BotName, b.cfg.SupportLink), nil)
}

func (b *Bot) cmdPing(m *tgbotapi.Message) {
	lag := int64(0)
	if m.Date > 0 {
		lag = time.Now().Unix() - int64(m.Date)
		if lag < 0 {
			lag = 0
		}
	}
	b.sendText(m.From.ID, fmt.Sprintf(PingPong, lag*1000), nil)
}

func (b *Bot) cmdID(m *tgbotapi.Message) {
	b.sendText(m.From.ID, fmt.Sprintf(YourID, m.From.ID), nil)
}

func (b *Bot) cmdFeedback(m *tgbotapi.Message, args string) {
	args = strings.TrimSpace(args)
	if args == "" {
		b.sendText(m.From.ID, FeedbackUsage, nil)
		return
	}
	f := b.st.AddFeedback(m.From.ID, args)
	b.sendText(m.From.ID, FeedbackOK, nil)
	b.sendToAdmins(fmt.Sprintf(FeedbackAdmin, f.ID, f.UserID, escapeHTML(f.Text)))
}

func (b *Bot) cmdBalance(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	left := b.st.SearchesLeft(uid)
	b.sendText(uid, fmt.Sprintf(Balance, left, b.dailyLimit(u.Premium), u.BonusSearches, u.DailyStreak), nil)
}

func (b *Bot) sendProfile(uid int64) {
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	left := b.st.SearchesLeft(uid)
	text := profileText(userView{
		Age: u.Age, Gender: u.Gender, Country: u.Country, Interests: u.Interests, Premium: u.Premium,
	}, left, b.dailyLimit(u.Premium))
	b.sendText(uid, text, ProfileEdit())
}

func (b *Bot) cmdProfile(m *tgbotapi.Message) {
	b.states.clear(m.From.ID)
	b.sendProfile(m.From.ID)
}

func (b *Bot) sendInterestsPicker(uid int64) {
	sel := b.st.GetInterests(uid)
	b.sendText(uid, fmt.Sprintf(InterestsPick, b.cfg.MaxInterests), InterestsPicker(sel))
}

func (b *Bot) cmdInterests(m *tgbotapi.Message) {
	b.states.clear(m.From.ID)
	u, ok := b.st.GetUser(m.From.ID)
	if !ok || !u.Registered {
		b.sendText(m.From.ID, NotRegistered, nil)
		return
	}
	b.sendInterestsPicker(m.From.ID)
}

func (b *Bot) cmdDaily(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	amount, streak, ok := b.st.ClaimDaily(uid)
	if !ok {
		b.sendText(uid, fmt.Sprintf(DailyAlready, streak), nil)
		return
	}
	b.sendText(uid, fmt.Sprintf(DailyOK, amount, streak), nil)
}

func (b *Bot) regAge(m *tgbotapi.Message, st *state) {
	text := strings.TrimSpace(m.Text)
	n, err := strconv.Atoi(text)
	if err != nil || n < b.cfg.MinAge || n > b.cfg.MaxAge {
		b.sendText(m.From.ID, fmt.Sprintf(AgeInvalid, b.cfg.MinAge, b.cfg.MaxAge), nil)
		return
	}
	st.Age = n
	b.states.setStep(m.From.ID, "reg_gender")
	b.sendText(m.From.ID, AskGender, GenderSelect("reg_gender:"))
}

func (b *Bot) regCountry(m *tgbotapi.Message, st *state) {
	country := strings.TrimSpace(m.Text)
	if country == "" {
		b.sendText(m.From.ID, AskCountry, nil)
		return
	}
	if len(country) > 60 {
		country = country[:60]
	}
	if st.Age == 0 || st.Gender == "" {
		b.states.clear(m.From.ID)
		b.sendText(m.From.ID, NotRegistered, nil)
		return
	}
	uid := m.From.ID
	b.st.UpdateUser(uid, func(u *store.User) {
		u.Age = st.Age
		u.Gender = st.Gender
		u.Country = country
		u.Registered = true
	})
	b.states.clear(uid)
	b.sendText(uid, RegDone, MainMenu())
	b.sendInterestsPicker(uid)
}

func (b *Bot) editAge(m *tgbotapi.Message) {
	text := strings.TrimSpace(m.Text)
	n, err := strconv.Atoi(text)
	if err != nil || n < b.cfg.MinAge || n > b.cfg.MaxAge {
		b.sendText(m.From.ID, fmt.Sprintf(AgeInvalid, b.cfg.MinAge, b.cfg.MaxAge), nil)
		return
	}
	uid := m.From.ID
	b.st.UpdateUser(uid, func(u *store.User) { u.Age = n })
	b.states.clear(uid)
	b.sendText(uid, "✅ Age updated!", nil)
	b.sendProfile(uid)
}

func (b *Bot) editCountry(m *tgbotapi.Message) {
	country := strings.TrimSpace(m.Text)
	if country == "" {
		b.sendText(m.From.ID, AskCountry, nil)
		return
	}
	if len(country) > 60 {
		country = country[:60]
	}
	uid := m.From.ID
	b.st.UpdateUser(uid, func(u *store.User) { u.Country = country })
	b.states.clear(uid)
	b.sendText(uid, "✅ Country updated!", nil)
	b.sendProfile(uid)
}

func (b *Bot) cbRegGender(cq *tgbotapi.CallbackQuery, gender string) {
	if gender != "male" && gender != "female" {
		b.answerCallback(cq, "", false)
		return
	}
	uid := cq.From.ID
	st := b.states.get(uid)
	st.Gender = gender
	b.states.setStep(uid, "reg_country")
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, AskCountry, nil)
	} else {
		b.sendText(uid, AskCountry, nil)
	}
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditGenderStart(cq *tgbotapi.CallbackQuery) {
	uid := cq.From.ID
	b.states.setStep(uid, "edit_gender")
	b.sendText(uid, AskGender, GenderSelect("edit_gender:"))
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditGenderSave(cq *tgbotapi.CallbackQuery, gender string) {
	if gender != "male" && gender != "female" {
		b.answerCallback(cq, "", false)
		return
	}
	uid := cq.From.ID
	b.st.UpdateUser(uid, func(u *store.User) { u.Gender = gender })
	b.states.clear(uid)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, "✅ Gender updated!", nil)
	}
	b.sendProfile(uid)
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditAgeStart(cq *tgbotapi.CallbackQuery) {
	b.states.setStep(cq.From.ID, "edit_age")
	b.sendText(cq.From.ID, fmt.Sprintf(AskAge, b.cfg.MinAge, b.cfg.MaxAge), nil)
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditCountryStart(cq *tgbotapi.CallbackQuery) {
	b.states.setStep(cq.From.ID, "edit_country")
	b.sendText(cq.From.ID, AskCountry, nil)
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditInterestsStart(cq *tgbotapi.CallbackQuery) {
	uid := cq.From.ID
	sel := b.st.GetInterests(uid)
	b.sendText(uid, fmt.Sprintf(InterestsPick, b.cfg.MaxInterests), InterestsPicker(sel))
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbEditPremium(cq *tgbotapi.CallbackQuery) {
	u, _ := b.st.GetUser(cq.From.ID)
	premium := false
	if u.Premium {
		premium = true
	}
	var markup interface{}
	if !premium {
		markup = PremiumPayment()
	}
	b.sendText(cq.From.ID, premiumText(premium, b.cfg), markup)
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbInterestToggle(cq *tgbotapi.CallbackQuery, key string) {
	result := b.st.ToggleInterest(cq.From.ID, key)
	if result == "limit" {
		b.answerCallback(cq, fmt.Sprintf(InterestsLimit, b.cfg.MaxInterests), true)
		return
	}
	sel := b.st.GetInterests(cq.From.ID)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, fmt.Sprintf(InterestsPick, b.cfg.MaxInterests), ptrMarkup(InterestsPicker(sel)))
	}
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbInterestClear(cq *tgbotapi.CallbackQuery) {
	b.st.ClearInterests(cq.From.ID)
	sel := b.st.GetInterests(cq.From.ID)
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, fmt.Sprintf(InterestsPick, b.cfg.MaxInterests), ptrMarkup(InterestsPicker(sel)))
	}
	b.answerCallback(cq, InterestsCleared, false)
}

func (b *Bot) cbInterestDone(cq *tgbotapi.CallbackQuery) {
	sel := b.st.GetInterests(cq.From.ID)
	label := "not set"
	if l := config.LabelList(sel); l != "" {
		label = l
	}
	if cq.Message != nil {
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, InterestsSaved+"\n\n🎯 <b>"+label+"</b>", nil)
	}
	b.answerCallback(cq, "", false)
}

func (b *Bot) cbPay(cq *tgbotapi.CallbackQuery, region string) {
	if u, _ := b.st.GetUser(cq.From.ID); u.Premium {
		b.sendText(cq.From.ID, PremiumGranted, nil)
		b.answerCallback(cq, "", false)
		return
	}
	text := "💳 <b>International Payment</b>\n\n" + premiumText(false, b.cfg) +
		"\n\nTell them your Telegram ID:\n<code>" + itoa(cq.From.ID) + "</code>"
	b.sendText(cq.From.ID, text, nil)
	b.answerCallback(cq, "", false)
}

func ptrMarkup(k tgbotapi.InlineKeyboardMarkup) *tgbotapi.InlineKeyboardMarkup {
	return &k
}
