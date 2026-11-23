package bot

import (
	"strings"

	tgbotapi "strangerbot/internal/tgapi"
)

// onCallback routes every inline button press to its handler.
func (b *Bot) onCallback(cq *tgbotapi.CallbackQuery) {
	if cq == nil || cq.From == nil {
		return
	}
	data := cq.Data
	switch {
	case strings.HasPrefix(data, "reg_gender:"):
		b.cbRegGender(cq, strings.TrimPrefix(data, "reg_gender:"))
	case strings.HasPrefix(data, "edit_gender:"):
		b.cbEditGenderSave(cq, strings.TrimPrefix(data, "edit_gender:"))
	case strings.HasPrefix(data, "int:t:"):
		b.cbInterestToggle(cq, strings.TrimPrefix(data, "int:t:"))
	case strings.HasPrefix(data, "rate:"):
		b.cbRate(cq)
	case strings.HasPrefix(data, "post:"):
		b.routePostCallback(cq)
	case strings.HasPrefix(data, "admin_resolve:"):
		b.cbAdminResolve(cq)
	case strings.HasPrefix(data, "admin_ban:"):
		b.cbAdminBan(cq)
	case data == "captcha:ok":
		b.cbCaptcha(cq)
	case strings.HasPrefix(data, "acct:"):
		if data == "acct:delete" {
			b.cbDeleteAccount(cq)
		} else if data == "acct:cancel" {
			b.cbDeleteCancel(cq)
		} else {
			b.answerCallback(cq, "", false)
		}
	case strings.HasPrefix(data, "appeal:"):
		if strings.HasPrefix(data, "appeal:unban:") {
			b.cbAppeal(cq, true)
		} else if strings.HasPrefix(data, "appeal:keep:") {
			b.cbAppeal(cq, false)
		} else {
			b.answerCallback(cq, "", false)
		}
	case strings.HasPrefix(data, "pay:"):
		if parts := strings.Split(data, ":"); len(parts) == 2 {
			b.cbPay(cq, parts[1])
		} else {
			b.answerCallback(cq, "", false)
		}
	case strings.HasPrefix(data, "rps:"):
		b.cbRPS(cq)
	case strings.HasPrefix(data, "ttt:"):
		b.cbTTT(cq)
	case strings.HasPrefix(data, "pick:"):
		b.cbSearchRegion(cq, strings.TrimPrefix(data, "pick:"))
	case data == "edit:age":
		b.cbEditAgeStart(cq)
	case data == "edit:gender":
		b.cbEditGenderStart(cq)
	case data == "edit:country":
		b.cbEditCountryStart(cq)
	case data == "edit:interests":
		b.cbEditInterestsStart(cq)
	case data == "edit:premium":
		b.cbEditPremium(cq)
	case data == "int:clear":
		b.cbInterestClear(cq)
	case data == "int:done":
		b.cbInterestDone(cq)
	default:
		b.answerCallback(cq, "", false)
	}
}

// routePostCallback splits the post-chat block/report buttons.
func (b *Bot) routePostCallback(cq *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	switch parts[1] {
	case "block":
		b.cbPostBlock(cq)
	case "report":
		b.cbPostReportStart(cq)
	default:
		b.answerCallback(cq, "", false)
	}
}
