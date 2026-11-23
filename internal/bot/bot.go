package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Iotatg/gotdbot"

	"strangerbot/internal/config"
	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

// Bot wires the gotdbot transport, JSON store and feature handlers together.
type Bot struct {
	api *tgbotapi.BotAPI
	cfg *config.Config
	st  *store.Store
	log *slog.Logger

	states  *stateStore
	msgs    *MessageMap
	rl      *rateLimiter
	matchMu sync.Mutex

	ratedMu sync.Mutex
	rated   map[string]bool

	iceMu   sync.Mutex
	iceLast map[int64]time.Time

	flood      *rateLimiter
	activity   map[int64]time.Time
	activityMu sync.Mutex
	relayLog   []relayEntry
	relayMu    sync.Mutex

	spam     *spamTracker
	auditMu  sync.Mutex
	auditLog []auditEntry

	safetyMu    sync.Mutex
	safetyShown map[int64]bool

	games *gameManager
}

// New authorizes the TDLib client and returns a ready bot.
func New(cfg *config.Config, st *store.Store, log *slog.Logger) (*Bot, error) {
	if err := os.MkdirAll(cfg.TDLibDatabaseDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.TDLibFilesDir, 0o755); err != nil {
		return nil, err
	}
	api, err := tgbotapi.NewBotAPIWithTDLib(tgbotapi.TDLibConfig{
		APIID:             int32(cfg.APIID),
		APIHash:           cfg.APIHash,
		Token:             cfg.Token,
		LibraryPath:       cfg.TDLibLibraryPath,
		DatabaseDirectory: cfg.TDLibDatabaseDir,
		FilesDirectory:    cfg.TDLibFilesDir,
	})
	if err != nil {
		return nil, err
	}
	b := &Bot{
		api:         api,
		cfg:         cfg,
		st:          st,
		log:         log,
		states:      newStateStore(),
		msgs:        NewMessageMap(20000),
		rl:          newRateLimiter(20, 10*time.Second),
		rated:       map[string]bool{},
		iceLast:     map[int64]time.Time{},
		flood:       newRateLimiter(cfg.FloodLimit, time.Duration(cfg.FloodWindowSeconds)*time.Second),
		activity:    map[int64]time.Time{},
		spam:        newSpamTracker(),
		safetyShown: map[int64]bool{},
		games:       newGameManager(),
	}
	if err := api.Start(); err != nil {
		return nil, err
	}
	b.register()
	log.Info("authorized", "bot", api.Self.UserName, "id", api.Self.ID)
	return b, nil
}

// API exposes the underlying transport.
func (b *Bot) API() *tgbotapi.BotAPI { return b.api }

func (b *Bot) register() {
	cl := b.api.Client()
	cl.OnMessage(func(_ *gotdbot.Client, m *gotdbot.Message) error {
		b.handle(tgbotapi.Update{Message: b.api.ConvertMessage(m)})
		return nil
	}, nil)
	cl.OnCallbackQuery(func(_ *gotdbot.Client, u *gotdbot.UpdateNewCallbackQuery) error {
		if cq := b.api.ConvertCallbackQuery(u); cq != nil {
			b.handle(tgbotapi.Update{CallbackQuery: cq})
		}
		return nil
	}, nil)
	if _, err := b.api.Send(tgbotapi.NewSetMyCommands(botCommands()...)); err != nil {
		b.log.Warn("set commands failed", "err", err)
	}
}

// Run blocks until ctx is cancelled, then shuts the client down.
func (b *Bot) Run(ctx context.Context) error {
	b.startSweepers()
	<-ctx.Done()
	b.api.Close()
	return nil
}

func (b *Bot) handle(upd tgbotapi.Update) {
	switch {
	case upd.CallbackQuery != nil:
		b.onCallback(upd.CallbackQuery)
	case upd.Message != nil:
		b.onMessage(upd.Message)
	}
}

func (b *Bot) onMessage(m *tgbotapi.Message) {
	if m == nil || m.From == nil || m.Chat == nil {
		return
	}
	if m.From.IsBot || !m.Chat.IsPrivate() {
		return
	}
	uid := m.From.ID
	b.st.EnsureUser(uid, m.From.UserName, m.From.FirstName)

	cmd := ""
	if m.IsCommand() {
		cmd = strings.ToLower(m.Command())
	}
	if b.st.IsBanned(uid) && !b.cfg.IsAdmin(uid) && cmd != "appeal" {
		u, _ := b.st.GetUser(uid)
		b.sendText(uid, fmt.Sprintf(Banned, escapeHTML(u.BanReason), b.cfg.SupportLink), nil)
		return
	}
	if b.cfg.Maintenance && !b.cfg.IsAdmin(uid) {
		b.sendText(uid, Maintenance, nil)
		return
	}

	if m.IsCommand() {
		b.onCommand(m)
		return
	}
	b.touchActivity(uid)
	if b.st.IsMuted(uid) && !b.cfg.IsAdmin(uid) {
		b.sendText(uid, StillMuted, nil)
		return
	}
	if b.st.IsInChat(uid) && !b.flood.allow(uid) {
		b.endChat(uid, false)
		b.sendText(uid, FloodWarned, nil)
		return
	}
	if b.handleState(m) {
		return
	}
	if containsLinkOrMention(m) {
		b.sendText(uid, "🚫 Links are not allowed in chat. Your message was not sent.", nil)
		return
	}
	if b.handleKeyboard(m) {
		return
	}
	b.relay(m)
}

func (b *Bot) handleState(m *tgbotapi.Message) bool {
	st := b.states.get(m.From.ID)
	switch st.Step {
	case "reg_age":
		b.regAge(m, st)
		return true
	case "reg_country":
		b.regCountry(m, st)
		return true
	case "edit_age":
		b.editAge(m)
		return true
	case "edit_country":
		b.editCountry(m)
		return true
	case "report_reason":
		b.reportReason(m)
		return true
	case "find_age":
		b.findAge(m, st)
		return true
	case "find_max":
		b.findMax(m, st)
		return true
	case "find_country":
		b.findCountry(m, st)
		return true
	}
	return false
}

func (b *Bot) handleKeyboard(m *tgbotapi.Message) bool {
	switch m.Text {
	case "» Next":
		b.cmdSearch(m)
		return true
	case "🔴 Stop":
		b.cmdStop(m)
		return true
	case "👧 Find Girl":
		b.findPartner(m, "female")
		return true
	case "👦 Find Boy":
		b.findPartner(m, "male")
		return true
	case "👤 Profile":
		b.cmdProfile(m)
		return true
	case "🎯 Interests":
		b.cmdInterests(m)
		return true
	}
	return false
}

func (b *Bot) onCommand(m *tgbotapi.Message) {
	cmd := strings.ToLower(m.Command())
	args := m.CommandArguments()
	switch cmd {
	case "start":
		b.cmdStart(m, m.CommandArguments())
	case "help":
		b.cmdHelp(m)
	case "profile":
		b.cmdProfile(m)
	case "interests":
		b.cmdInterests(m)
	case "daily":
		b.cmdDaily(m)
	case "search", "next":
		b.cmdSearch(m)
	case "stop":
		b.cmdStop(m)
	case "icebreaker":
		b.cmdIcebreaker(m)
	case "link":
		b.cmdLink(m)
	case "block":
		b.cmdBlock(m)
	case "blocklist":
		b.cmdBlocklist(m)
	case "unblock":
		b.cmdUnblock(m, args)
	case "del":
		b.cmdDel(m)
	case "report":
		b.cmdReport(m, args)
	case "leaderboard", "top":
		b.cmdLeaderboard(m)
	case "rank":
		b.cmdRank(m)
	case "balance":
		b.cmdBalance(m)
	case "stats":
		b.cmdPublicStats(m)
	case "trust":
		b.cmdTrust(m)
	case "trustlist":
		b.adminTrustlist(m)
	case "find":
		b.cmdFind(m, args)
	case "referral":
		b.cmdReferral(m)
	case "privacy":
		b.cmdPrivacy(m, args)
	case "mydata":
		b.cmdMyData(m)
	case "deleteaccount":
		b.cmdDeleteAccount(m)
	case "gift":
		b.cmdGift(m, args)
	case "appeal":
		b.cmdAppeal(m, args)
	case "maintenance":
		b.cmdMaintenance(m, args)
	case "purge":
		b.cmdPurge(m)
	case "massban":
		b.cmdMassban(m, args)
	case "digest":
		b.cmdDigestNow(m)
	case "security":
		b.cmdSecurity(m)
	case "strikes":
		b.cmdStrikes(m, args)
	case "resetstrikes":
		b.cmdResetStrikes(m, args)
	case "unmute":
		b.cmdUnmute(m, args)
	case "audit":
		b.cmdAudit(m)
	case "safety":
		b.cmdSafety(m)
	case "adminstats":
		b.adminStats(m)
	case "about":
		b.cmdAbout(m)
	case "feedback":
		b.cmdFeedback(m, args)
	case "8ball":
		b.cmdEightBall(m, args)
	case "dice":
		b.cmdDice(m)
	case "coin":
		b.cmdCoin(m)
	case "rps":
		b.cmdRPS(m, args)
	case "ttt":
		b.cmdTTT(m, args)
	case "games":
		b.cmdGames(m)
	case "choose":
		b.cmdChoose(m, args)
	case "ping":
		b.cmdPing(m)
	case "id":
		b.cmdID(m)
	case "cancel":
		b.states.clear(m.From.ID)
		b.st.RemoveWaiting(m.From.ID)
		b.sendText(m.From.ID, Cancelled, nil)
	case "ban":
		b.adminBan(m, args)
	case "unban":
		b.adminUnban(m, args)
	case "premium":
		b.adminPremium(m, args, true)
	case "unpremium":
		b.adminPremium(m, args, false)
	case "addcoin":
		b.adminAddCoin(m, args)
	case "broadcast":
		b.adminBroadcast(m, args)
	case "reports":
		b.adminReports(m)
	case "resolve":
		b.adminResolve(m, args)
	case "userinfo":
		b.adminUserInfo(m, args)
	default:
		// Unknown command: ignore silently while in a chat so it is never
		// leaked to the partner.
	}
}
