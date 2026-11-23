package bot

import (
	"fmt"

	"strangerbot/internal/config"
	tgbotapi "strangerbot/internal/tgapi"
)

// MainMenu is the persistent reply keyboard shown to registered users.
func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{{Text: "» Next"}, {Text: "🔴 Stop"}},
			{{Text: "👧 Find Girl"}, {Text: "👦 Find Boy"}},
			{{Text: "👤 Profile"}, {Text: "🎯 Interests"}},
		},
		ResizeKeyboard:        true,
		InputFieldPlaceholder: "Type a message or pick an action…",
	}
}

// RemoveKeyboard hides the reply keyboard.
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.ReplyKeyboardRemove{RemoveKeyboard: true}
}

// GenderSelect returns the gender picker. prefix is "reg_gender:" or "edit_gender:".
func GenderSelect(prefix string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👦 Male", prefix+"male"),
			tgbotapi.NewInlineKeyboardButtonData("👧 Female", prefix+"female"),
		),
	)
}

// ProfileEdit is the inline keyboard under the profile card.
func ProfileEdit() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Age", "edit:age"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Gender", "edit:gender"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Country", "edit:country"),
			tgbotapi.NewInlineKeyboardButtonData("🎯 Interests", "edit:interests"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧸 Premium", "edit:premium"),
		),
	)
}

// InterestsPicker renders the interest toggle grid.
func InterestsPicker(selected []string) tgbotapi.InlineKeyboardMarkup {
	picked := make(map[string]bool, len(selected))
	for _, s := range selected {
		picked[s] = true
	}
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, in := range config.Interests {
		mark := ""
		if picked[in.Key] {
			mark = "✅ "
		}
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(mark+in.Label, "int:t:"+in.Key))
		if len(row) == 2 {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🗑 Clear", "int:clear"),
		tgbotapi.NewInlineKeyboardButtonData("💾 Done", "int:done"),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PostChat is the inline keyboard after a chat ends.
func PostChat(partnerID int64, showBlock bool) tgbotapi.InlineKeyboardMarkup {
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("👍 Like", fmt.Sprintf("rate:like:%d", partnerID)),
		tgbotapi.NewInlineKeyboardButtonData("👎 Dislike", fmt.Sprintf("rate:dislike:%d", partnerID)),
	)
	rows := [][]tgbotapi.InlineKeyboardButton{row}
	if showBlock {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Block user", fmt.Sprintf("post:block:%d", partnerID)),
		))
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🚩 Report user", fmt.Sprintf("post:report:%d", partnerID)),
	))
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PostChatMinimal keeps only Block/Report after a rating.
func PostChatMinimal(partnerID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚫 Block user", fmt.Sprintf("post:block:%d", partnerID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚩 Report user", fmt.Sprintf("post:report:%d", partnerID)),
		),
	)
}

// PostChatReportOnly keeps only Report.
func PostChatReportOnly(partnerID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚩 Report user", fmt.Sprintf("post:report:%d", partnerID)),
		),
	)
}

// AdminReportBtns is the resolve/ban pair under an admin report card.
func AdminReportBtns(reportID, reportedID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Resolve", fmt.Sprintf("admin_resolve:%d", reportID)),
			tgbotapi.NewInlineKeyboardButtonData("🚫 Ban", fmt.Sprintf("admin_ban:%d:%d", reportedID, reportID)),
		),
	)
}

// SearchRegionPicker renders the India / Foreign buttons shown on /search.
func SearchRegionPicker() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇮🇳 India", "pick:india"),
			tgbotapi.NewInlineKeyboardButtonData("🌍 Foreign", "pick:foreign"),
		),
	)
}

// PremiumPayment is the premium purchase button. Premium is offered to
// foreign users only.
func PremiumPayment() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🌍 Foreign Users", "pay:foreign"),
		),
	)
}

// AccountButtons is the inline keyboard under /mydata.
func AccountButtons() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑 Delete my account", "acct:delete"),
		),
	)
}

// DeleteConfirm asks the user to confirm account deletion.
func DeleteConfirm() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❗️ Yes, delete", "acct:delete"),
			tgbotapi.NewInlineKeyboardButtonData("↩️ Cancel", "acct:cancel"),
		),
	)
}

// CaptchaButton is the one-tap human verification prompt.
func CaptchaButton() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ I'm human", "captcha:ok"),
		),
	)
}
