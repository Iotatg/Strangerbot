package bot

import tgbotapi "strangerbot/internal/tgapi"

// commandSpec is one entry in the public command menu.
type commandSpec struct {
	Name        string
	Description string
}

// commandList is the full command menu exposed to Telegram.
var commandList = []commandSpec{
	{"start", "Restart / register"},
	{"help", "How to use the bot"},
	{"profile", "View / edit your profile"},
	{"interests", "Pick interests for better matches"},
	{"daily", "Claim your daily bonus searches"},
	{"search", "Find a stranger"},
	{"next", "Skip to a new partner"},
	{"stop", "End current chat"},
	{"icebreaker", "Send a conversation starter"},
	{"link", "Send your Telegram profile to your partner"},
	{"block", "Block current partner (premium)"},
	{"blocklist", "View blocked users"},
	{"unblock", "Unblock a user by list number"},
	{"del", "Delete a message for both sides (premium)"},
	{"report", "Report current partner"},
	{"leaderboard", "Top chatters"},
	{"rank", "Your position on the leaderboard"},
	{"balance", "Your search balance"},
	{"stats", "Live bot statistics"},
	{"about", "About this bot"},
	{"feedback", "Send feedback to the admins"},
	{"8ball", "Ask the magic 8-ball"},
	{"dice", "Roll a dice"},
	{"coin", "Flip a coin"},
	{"rps", "Rock paper scissors vs partner"},
	{"ttt", "Tic tac toe vs partner"},
	{"games", "List games"},
	{"choose", "Let the bot choose"},
	{"ping", "Check latency"},
	{"id", "Show your Telegram ID"},
	{"cancel", "Cancel the current action"},
	{"trust", "Your reputation score"},
	{"trustlist", "Trust leaderboard (admin)"},
	{"find", "Set age / country match filters"},
	{"referral", "Invite friends and earn searches"},
	{"privacy", "Auto-delete chat messages"},
	{"mydata", "Show the data stored about you"},
	{"deleteaccount", "Delete your account"},
	{"gift", "Gift premium to a user"},
	{"appeal", "Appeal a ban"},
	{"safety", "Community safety rules"},
	{"security", "Security status (admin)"},
	{"strikes", "Show user strikes (admin)"},
	{"resetstrikes", "Reset user strikes (admin)"},
	{"unmute", "Unmute a user (admin)"},
	{"audit", "Admin audit log (admin)"},
	{"maintenance", "Toggle maintenance (admin)"},
	{"purge", "Clear reports (admin)"},
	{"massban", "Ban multiple users (admin)"},
	{"digest", "Send stats digest (admin)"},
	{"ban", "Ban a user (admin)"},
	{"unban", "Unban a user (admin)"},
	{"premium", "Grant premium (admin)"},
	{"unpremium", "Revoke premium (admin)"},
	{"addcoin", "Add bonus searches (admin)"},
	{"broadcast", "Message all users (admin)"},
	{"reports", "Open reports (admin)"},
	{"resolve", "Resolve a report (admin)"},
	{"userinfo", "Show a user record (admin)"},
	{"adminstats", "Bot statistics (admin)"},
}

// botCommands converts the menu to the transport type.
func botCommands() []tgbotapi.BotCommand {
	out := make([]tgbotapi.BotCommand, 0, len(commandList))
	for _, c := range commandList {
		out = append(out, tgbotapi.BotCommand{Command: c.Name, Description: c.Description})
	}
	return out
}

// KnownCommands returns the registered command names (used by tests).
func KnownCommands() []string {
	out := make([]string, 0, len(commandList))
	for _, c := range commandList {
		out = append(out, c.Name)
	}
	return out
}
