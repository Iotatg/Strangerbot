package bot

import (
	"fmt"
	"strings"

	"strangerbot/internal/config"
)

const (
	Welcome = "💜💙 <b>Welcome to Anonymous Chat!</b>\n\n" +
		"Talk to random people anonymously. No names, no pressure.\n" +
		"First, let's quickly set up your profile."

	WelcomeBack = "💙 <b>Welcome back to %s!</b>\n\nUse the buttons below to find a stranger."

	AskAge     = "📅 Send your <b>age</b> (%d-%d):"
	AgeInvalid = "✅ Please send a valid age (%d-%d)."
	AskGender  = "❓ Select your <b>gender</b>:"
	AskCountry = "🌍 Send your <b>country</b> (e.g. India):"
	RegDone    = "✅ <b>Registration complete!</b>"

	Profile = "<b>Your profile</b>\n\n" +
		"Age: <b>%d</b>\n" +
		"Gender: %s <b>%s</b>\n" +
		"Country: <b>%s</b>\n" +
		"Interests: <b>%s</b>\n" +
		"Searches left today: <b>%d/%d</b>\n" +
		"Status: %s <b>%s</b>"

	Searching        = "🌀 Looking for a partner…\n\n/stop — stop searching"
	PartnerFound     = "💙 <b>Partner found</b> | %s\n%s\n/next — skip to a new partner\n/stop — end the dialogue\n/link — share your telegram profile\n/icebreaker — get a conversation starter"
	AlreadySearching = "⏳ Already searching… please wait."
	AlreadyInChat    = "⚠️ You're already chatting! Use /next or /stop."
	NotInChat        = "❗️ You're not in a chat. Use Search to find someone!"
	NoSearchesLeft   = "❗️ <b>No searches left today!</b>\n\nGet more by:\n  • Having partners <b>Like 👍</b> you\n  • Upgrading to <b>Premium</b>\n\nSupport: %s"
	NotRegistered    = "⚠️ Please complete your profile first!\nSend /start to register."

	ChatEndedYou     = "✅ You stopped the chat.\n\n<b>How was your chat?</b>\n\n👍 Like → partner gets +1 search\n👎 Dislike → partner gets -1 search"
	ChatEndedPartner = "👋 Your partner has stopped the chat.\n\n/next — find a new partner"
	NextSearching    = "⏭ Searching for a new partner…"

	RatedLike    = "👍 Liked! Your partner gets +1 search."
	RatedDislike = "👎 Disliked. Partner gets -1 search."
	AlreadyRated = "⚠️ You already rated this conversation."

	InterestsPick    = "🎯 <b>Your interests</b>\n\nPick up to <b>%d</b>. When searching, the bot prefers strangers who share your interests (and still finds you someone if nobody does)."
	InterestsLimit   = "⚠️ You can pick up to %d interests. Untick one first."
	InterestsSaved   = "✅ Interests saved!"
	InterestsCleared = "🗑 Interests cleared."
	CommonInterests  = "🎯 Common interests: <b>%s</b>"
	NoCommon         = "🎯 No shared interests yet — say hi!"

	IcebreakerNotInChat = "❗️ Start a chat first, then use /icebreaker."
	IcebreakerCooldown  = "⏳ Slow down — try again in %ds."
	IcebreakerMsg       = "🧊 <b>Icebreaker</b>\n\n%s"

	DailyOK      = "🎁 <b>Daily bonus claimed!</b>\n\n+<b>%d</b> searches • 🔥 streak: <b>%d</b> day(s)\nCome back tomorrow to grow your streak!"
	DailyAlready = "⏳ You already claimed today's bonus. Come back tomorrow! (🔥 streak: %d)"

	LinkNoUsername = "❗️ <b>No username set.</b>\n\nGo to: Settings → My Profile → Username\nSet a username, then try /link again."
	LinkNotInChat  = "❗️ You need to be in a chat to share your profile."
	LinkOK         = "🔗 <b>Your profile link was sent to your partner.</b>"
	LinkPartner    = "🔗 <b>Your partner shared their profile:</b>\nhttps://t.me/%s"

	BlockNotInChat   = "❗️ You need to be in a chat to block your partner."
	BlockOK          = "🚫 User blocked! You'll never be matched with them again."
	BlockAlready     = "⚠️ You've already blocked this user."
	BlockLimit       = "⚠️ Block limit reached (%d max). Remove someone from /blocklist first."
	BlockPremiumOnly = "🔒 /block is a <b>Premium</b> feature.\n\n" + "Contact support to upgrade."

	BlocklistEmpty  = "📒 Your block list is empty."
	BlocklistHeader = "📒 <b>Block List</b>\n\n"
	BlocklistRow    = "  <b>%d.</b> User <code>%d</code>"

	UnblockUsage    = "Usage: <code>/unblock &lt;id&gt;</code>  (see /blocklist for IDs)"
	UnblockOK       = "🔓 User unblocked!"
	UnblockNotFound = "❗️ No entry with that ID in your block list."

	DelPremiumOnly = "🔒 /del is a <b>Premium</b> feature."
	DelNotInChat   = "❗️ You must be in a chat to use /del."
	DelReplyNeeded = "↩️ Reply to <b>your own message</b> with /del."
	DelNotYourMsg  = "⚠️ You can only delete your own messages."
	DelOK          = "🗑 Message deleted for both sides."

	ReportNotInChat = "❗️ You need to be in a chat to report someone."
	ReportUsage     = "Usage: <code>/report &lt;reason&gt;</code>"
	Unsupported     = "⚠️ That message type can't be sent to strangers."
	PartnerLeftBot  = "👋 Your partner is no longer reachable, so the chat was ended.\n\n/next — find a new partner"
	ReportOK        = "🚨 <b>Report submitted!</b>\n\nOur admins will review the conversation. Fake reports will result in action against your account."

	PremiumGranted      = "🧸 <b>You now have Premium!</b>\n\nEnjoy all benefits!"
	GenderSearchPremium = "🧸 Gender-wise search is a Premium feature.\n\n"

	AdminStats = "📊 <b>Bot Statistics</b>\n" +
		"👥 Total users:    <b>%d</b>\n" +
		"✅ Registered:     <b>%d</b>\n" +
		"💬 Active chats:   <b>%d</b>\n" +
		"⏳ Waiting:        <b>%d</b>\n" +
		"🧸 Premium:        <b>%d</b>\n" +
		"🚨 Open reports:   <b>%d</b> / %d\n" +
		"🚫 Banned:         <b>%d</b>"

	AdminNotAdmin      = "🚫 Not authorised."
	AdminBroadcastDone = "📣 Sent to <b>%d</b> users (%d failed)."

	Banned = "🚫 <b>You have been banned.</b>\n\nReason: <i>%s</i>\n\nAppeal: %s"

	Maintenance = "🛠 <b>Maintenance mode</b>\n\nThe bot is temporarily unavailable. Please try again later."
	Cancelled   = "✅ Cancelled."

	Help = "💬 <b>StrangerBot — Help</b>\n\n" +
		"<b>Basics</b>\n" +
		"/start — register or restart\n" +
		"/profile — view or edit your profile\n" +
		"/search — find a stranger\n" +
		"/next (» Next) — skip to a new partner\n" +
		"/stop (🔴 Stop) — end the current chat\n\n" +
		"<b>Chat tools</b>\n" +
		"/icebreaker — get a conversation starter\n" +
		"/link — share your Telegram profile\n" +
		"/interests — pick interests for better matches\n" +
		"/daily — claim your daily bonus searches\n" +
		"/report &lt;reason&gt; — report your partner\n" +
		"/block — block your partner (Premium)\n" +
		"/blocklist — view blocked users\n" +
		"/unblock &lt;id&gt; — unblock a user\n" +
		"/del — delete a message for both sides (Premium)\n\n" +
		"<b>Extras</b>\n" +
		"/leaderboard — top chatters\n" +
		"/rank — your position\n" +
		"/balance — your search balance\n" +
		"/stats — live bot statistics\n" +
		"/about — about this bot\n" +
		"/feedback &lt;text&gt; — send feedback to admins\n" +
		"/8ball &lt;question&gt; · /dice · /coin · /rps · /choose a b c\n" +
		"/ping — check latency\n" +
		"/id — show your Telegram ID\n" +
		"/cancel — cancel the current action"

	About = "💙 <b>%s</b>\n\nAnonymous stranger chat for Telegram.\n" +
		"Stay private, stay safe, and be kind.\n\n" +
		"Support: %s\n\n" +
		"Credits: <a href=\"https://github.com/rohon-x04\">GitHub</a> • <a href=\"https://t.me/Rohon_x04\">@Rohon_x04</a>\n" +
		"Channel: <a href=\"https://t.me/im_iota\">@im_iota</a>"

	PingPong      = "🏓 Pong! <code>%dms</code>"
	YourID        = "🆔 Your Telegram ID: <code>%d</code>"
	FeedbackUsage = "Usage: <code>/feedback &lt;your message&gt;</code>"
	FeedbackOK    = "✅ Thanks! Your feedback was sent to the admins."

	LeaderboardHeader = "🏆 <b>Top Chatters</b>\n\n"
	LeaderboardRow    = "<b>#%d</b> — %s — <b>%d</b> chats"
	LeaderboardEmpty  = "🏆 No ranked users yet. Start chatting!"

	RankOK   = "📈 You are <b>#%d</b> of <b>%d</b> ranked users."
	RankNone = "📈 You are not ranked yet. Complete your profile and start chatting."

	PublicStats = "📊 <b>Live statistics</b>\n\n👥 Users: <b>%d</b>\n✅ Registered: <b>%d</b>\n💬 Active chats: <b>%d</b>\n⏳ Searching: <b>%d</b>\n🧸 Premium: <b>%d</b>"

	Balance = "💰 <b>Your balance</b>\n\nToday: <b>%d/%d</b> searches left\nBonus pool: <b>%d</b>\n🔥 Daily streak: <b>%d</b>"
)

// Icebreakers is the pool of conversation starters.
var Icebreakers = []string{
	"What's a song you've had on repeat lately?",
	"If you could instantly learn any skill, what would it be?",
	"What's the best thing that happened to you this week?",
	"Which movie or series can you watch over and over?",
	"Where would you travel tomorrow if money wasn't an issue?",
	"What's a small thing that always makes your day better?",
	"Early bird or night owl — and what do you do at that time?",
	"What's the most random fact you know?",
	"What's your go-to comfort food?",
	"If you had a free day with no plans, how would you spend it?",
	"What's something you're proud of but rarely talk about?",
	"Which app can't you live without?",
	"What's the best advice you've ever received?",
	"Which fictional world would you want to live in?",
	"What's a hobby you'd love to pick up?",
	"Two truths and a lie — go!",
	"Would you rather be able to fly or be invisible?",
	"What's your most unpopular opinion (the harmless kind)?",
}

// Game replies.
const (
	EightBallTemplate = "🎱 <b>Magic 8-Ball</b>\n\n%s"
	DiceTemplate      = "🎲 You rolled a <b>%d</b>."
	CoinTemplate      = "🪙 <b>%s</b>"
	ChooseTemplate    = "🎯 I choose: <b>%s</b>"
	FeedbackAdmin     = "📨 <b>New feedback #%d</b>\nFrom: <code>%d</code>\n\n%s"
)

// genderIcon returns the emoji for a gender.
func genderIcon(g string) string {
	if g == "male" {
		return "👦"
	}
	return "👧"
}

// capFirst upper-cases the first rune for display.
func capFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// profileText renders the profile card.
func profileText(u userView, left, daily int) string {
	interests := config.LabelList(u.Interests)
	if interests == "" {
		interests = "not set"
	}
	status, statusIcon := "Not active", "❗️"
	if u.Premium {
		status, statusIcon = "Premium active", "✅"
	}
	return fmt.Sprintf(Profile, u.Age, genderIcon(u.Gender), capFirst(u.Gender), escapeHTML(u.Country), interests, left, daily, statusIcon, status)
}

// userView is the subset of store.User the text layer needs.
type userView struct {
	Age       int
	Gender    string
	Country   string
	Interests []string
	Premium   bool
}

// premiumText renders the premium info block. The upgrade link is only shown
// to users who are not premium yet.
func premiumText(premium bool, cfg *config.Config) string {
	status := "❗️ Not active"
	if premium {
		status = "✅ Active"
	}
	text := fmt.Sprintf("🧸 <b>Premium Benefits</b>\n\n"+
		"• 👧 Find Girl / 👦 Find Boy: search by gender\n"+
		"• 🔎 <b>%d</b> daily searches (free: %d)\n"+
		"• 🚫 /block, /unblock: block up to %d users\n"+
		"• 🗑 /del: delete your sent messages\n"+
		"• 🧸 Premium badge shown to your partners\n\n"+
		"Status: %s", cfg.PremiumDailySearches, cfg.FreeDailySearches, cfg.MaxBlockList, status)
	if !premium {
		text += "\n\nContact support to upgrade:\n" + cfg.SupportLink
	}
	return text
}

const (
	FindAgePrompt       = "🎯 <b>Match filters</b>\n\nSend the <b>minimum</b> age you want to meet (e.g. <code>18</code>), or <code>-</code> to skip."
	FindAgeInvalid      = "❗️ Use an age like <code>18-25</code> or <code>18</code>."
	FindAgeInvalidRange = "❗️ Enter an age between %d and %d."
	FindMaxPrompt       = "🎯 Send the <b>maximum</b> age you want to meet (e.g. <code>25</code>)."
	FindCountryPrompt   = "🌍 Send a <b>country</b> (e.g. <code>India</code>), or <code>any</code> to skip."
	FindSaved           = "✅ Filters saved: <b>%s</b>. Use /search to find a partner."
	SearchRegionPrompt  = "🌍 <b>Who do you want to meet?</b>\n\nChoose a region to start searching."
)

const (
	TrustCard = "🛡 <b>Your trust score</b>\n\n" +
		"Score: <b>%d/100</b> %s\n" +
		"Level: <b>%s</b>\n\n" +
		"Chats: %d\n👍 %d · 👎 %d\nOpen reports: %d\nReferrals: %d\n\n" +
		"Build trust by chatting politely and not getting reported."
)

const (
	MyData = "📦 <b>Your data</b>\n\n" +
		"ID: <code>%d</code>\nName: %s\nUsername: @%s\nAge: %d\nGender: %s\nCountry: %s\n" +
		"Interests: %s\nChats: %d\n👍 %d · 👎 %d\nReports: %d\n" +
		"Referred by: <code>%d</code> · Referrals: %d\nJoined: %s\n\n" +
		"You can delete everything with the button below."

	DeleteAccountPrompt = "⚠️ <b>Delete your account?</b>\n\nThis removes your profile, interests, stats and chat pairing. This cannot be undone."
	DeleteAccountDone   = "🗑 Your account and data have been deleted."
)

const (
	ReferralCard     = "🎁 <b>Invite friends</b>\n\nHey %s, share your link and earn <b>%d</b> bonus searches for every friend who joins:\n\n<code>%s</code>\n\nInvited so far: <b>%d</b>"
	ReferralWelcome  = "🎁 You joined via a referral. <b>+2 bonus searches</b> added!"
	ReferralRewarded = "🎁 Someone you invited joined! <b>+%d bonus searches</b>."
)

const (
	PrivacyOn     = "🔒 <b>Privacy mode enabled</b>\n\nMessages you exchange will be deleted automatically after a few minutes."
	PrivacyOff    = "🔓 <b>Privacy mode disabled</b>\n\nYour chat history stays until you or your partner delete it."
	PrivacyStatus = "🔐 <b>Privacy mode: %s</b>\n\nAuto-delete after <b>%d</b> minutes.\nUse <code>/privacy on</code> or <code>/privacy off</code>."
)

const (
	GiftPremiumOnly = "🧸 Only premium users can gift premium."
	GiftSent        = "🎁 Premium gifted! <b>%d day(s)</b> added to <code>%d</code>."
	GiftReceived    = "🎁 <b>%s</b> gifted you <b>%d day(s)</b> of premium!"
	PremiumExpired  = "🧸 Your gifted premium has expired. Thanks for chatting!"
)

const (
	NotBanned      = "✅ You are not banned."
	AppealUsage    = "📨 Tell the admins why you should be unbanned: <code>/appeal your reason</code>"
	AppealAdmin    = "📨 <b>Appeal from <code>%d</code></b>\n\nReason: <i>%s</i>"
	AppealSent     = "📨 Your appeal has been sent to the admins."
	AppealAccepted = "✅ Your appeal was accepted. Your ban has been lifted."
)

const (
	FloodWarned   = "⏳ You are sending messages too fast. Your chat was ended to keep things fair. Slow down a little."
	ChatIdleEnded = "⌛ This chat was ended because of inactivity. Send /search to find someone new."
)

const (
	DigestTemplate = "📊 <b>Daily digest</b>\n\n" +
		"Users: <b>%d</b> (registered %d)\nActive chats: %d\nWaiting: %d\n" +
		"Premium: %d\nBanned: %d\nOpen reports: %d (total %d)"
)

const (
	WarningNotice = "⚠️ <b>Warning</b>\n\nSharing personal contact details or repeating messages is not allowed. Further violations will mute or ban you."
	MutedNotice   = "🔇 You have been muted for repeated violations. Please read /safety and try again later."
	StillMuted    = "🔇 You are muted right now. Please wait before sending more messages."
	SlowDown      = "⏳ Please wait a moment before searching again."
	CaptchaPrompt = "🤖 <b>Quick check</b>\n\nTap the button below to prove you are human and start chatting."
	CaptchaDone   = "✅ You are verified. Send /search to find a partner."
	SafetyNotice  = "🛡 <b>Safety first</b>\n\n" +
		"• Never share phone numbers, emails or social handles.\n" +
		"• Do not send money, gift cards or crypto to anyone.\n" +
		"• Forwarded messages and private invite links are blocked.\n" +
		"• Use /report or the 🚩 button if someone misbehaves.\n" +
		"• Messages are never shown with your name or username."

	SecurityStatus = "🛡 <b>Security status</b>\n\n" +
		"Max strikes: <b>%d</b>\nMute after: <b>%d min</b>\nAuto-ban after reports: <b>%d</b>\n" +
		"Human check: <b>%s</b>\nProtect content: <b>%s</b>\nBlock forwards: <b>%s</b>\n" +
		"Search cooldown: <b>%ds</b>\nPrivacy delete: <b>%d min</b>"

	StrikesInfo = "🚦 User <code>%d</code>: <b>%d</b> / %d strikes"
)

const (
	GamesMenu     = "🎮 <b>Games</b>\n\n/rps — rock paper scissors vs your partner\n/ttt — tic tac toe vs your partner\n\nNo partner yet? Use /search first."
	GameNotInChat = "🎮 You need an active chat partner first. Use /search."
	GameCancelled = "↩️ Your game was cancelled."

	GameRPSPrompt = "✊✋✌️ <b>Rock Paper Scissors</b>\n\nvs %s\n\nChoose your move:"
	GameRPSResult = "🏆 <b>%s wins!</b>\n\n%s beats %s"

	SoloRPSTie  = "🤝 <b>Draw!</b> You chose %s, I chose %s."
	SoloRPSWin  = "🏆 <b>You win!</b> You chose %s, I chose %s."
	SoloRPSLose = "💀 <b>I win!</b> You chose %s, I chose %s."

	GameTTTWin = "🏆 <b>%s wins!</b> %s completes the line."
)
