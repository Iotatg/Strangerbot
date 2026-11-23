package tgbotapi

import (
	"strconv"
	"sync"
	"time"

	"github.com/Iotatg/gotdbot"
)

// TDLibConfig configures the gotdbot/TDLib backed transport.
type TDLibConfig struct {
	APIID             int32
	APIHash           string
	Token             string
	LibraryPath       string
	DatabaseDirectory string
	FilesDirectory    string
}

type cachedChat struct {
	c   Chat
	exp time.Time
}

type cachedUser struct {
	u   User
	exp time.Time
}

type cachedStickerSet struct {
	name string
	exp  time.Time
}

type cachedMember struct {
	m   ChatMember
	exp time.Time
}

type cachedMessage struct {
	m   *gotdbot.Message
	exp time.Time
}

// BotAPI is a tgbotapi-shaped facade implemented on top of gotdbot (TDLib).
type BotAPI struct {
	Token string `json:"token"`
	Debug bool   `json:"debug"`

	Self User `json:"-"`

	client *gotdbot.Client

	mu          sync.Mutex
	chatCache   map[int64]cachedChat
	userCache   map[int64]cachedUser
	stickerSets map[int64]cachedStickerSet
	memberCache map[string]cachedMember
	msgCache    map[string]cachedMessage

	msgMu  sync.Mutex
	msgIDs map[int64][]int64
}

const (
	cacheTTL       = 10 * time.Minute
	memberCacheTTL = time.Minute
	msgCacheTTL    = 30 * time.Second
	msgCacheMax    = 1024
)

// NewBotAPIWithTDLib creates a BotAPI backed by a gotdbot client.
func NewBotAPIWithTDLib(cfg TDLibConfig) (*BotAPI, error) {
	opts := &gotdbot.ClientOpts{
		LibraryPath:       cfg.LibraryPath,
		DatabaseDirectory: cfg.DatabaseDirectory,
		FilesDirectory:    cfg.FilesDirectory,
	}
	client, err := gotdbot.NewClient(cfg.APIID, cfg.APIHash, cfg.Token, opts)
	if err != nil {
		return nil, err
	}
	return &BotAPI{
		Token:       cfg.Token,
		client:      client,
		chatCache:   make(map[int64]cachedChat),
		userCache:   make(map[int64]cachedUser),
		stickerSets: make(map[int64]cachedStickerSet),
		memberCache: make(map[string]cachedMember),
		msgCache:    make(map[string]cachedMessage),
		msgIDs:      make(map[int64][]int64),
	}, nil
}

// Client exposes the underlying gotdbot client for update registration.
func (b *BotAPI) Client() *gotdbot.Client { return b.client }

// Start authorizes the underlying TDLib client and fills Self.
func (b *BotAPI) Start() error {
	if err := b.client.Start(); err != nil {
		return err
	}
	if b.client.Me != nil {
		b.Self = convertUser(b.client.Me)
	}
	return nil
}

// Close shuts down the underlying TDLib client.
func (b *BotAPI) Close() {
	if b.client != nil {
		b.client.Close()
	}
}

func memberKey(chatID, userID int64) string {
	return strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(userID, 10)
}

// noteMessage records a message id seen in a chat so commands such as purge can
// resolve an id range without getChatHistory (which TDLib forbids for bots).
func (b *BotAPI) noteMessage(chatID, id int64) {
	if chatID == 0 || id == 0 {
		return
	}
	b.msgMu.Lock()
	defer b.msgMu.Unlock()
	list := b.msgIDs[chatID]
	if n := len(list); n > 0 {
		if list[n-1] == id {
			return
		}
		if list[n-1] > id {
			for i := n - 1; i >= 0; i-- {
				if list[i] == id {
					return
				}
				if list[i] < id {
					list = append(list[:i+1], append([]int64{id}, list[i+1:]...)...)
					break
				}
				if i == 0 {
					list = append([]int64{id}, list...)
				}
			}
			b.msgIDs[chatID] = list
			return
		}
	}
	list = append(list, id)
	if len(list) > 1000 {
		list = list[len(list)-1000:]
	}
	b.msgIDs[chatID] = list
}

func (b *BotAPI) cachedIDsInRange(chatID, lo, hi int64) []int64 {
	b.msgMu.Lock()
	defer b.msgMu.Unlock()
	var out []int64
	for _, id := range b.msgIDs[chatID] {
		if id >= lo && id <= hi {
			out = append(out, id)
		}
	}
	return out
}

func (b *BotAPI) cachedMember(chatID, userID int64) (ChatMember, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	c, ok := b.memberCache[memberKey(chatID, userID)]
	if !ok || time.Now().After(c.exp) {
		return ChatMember{}, false
	}
	return c.m, true
}

func (b *BotAPI) cacheMember(chatID, userID int64, m ChatMember) {
	b.mu.Lock()
	b.memberCache[memberKey(chatID, userID)] = cachedMember{m: m, exp: time.Now().Add(memberCacheTTL)}
	b.mu.Unlock()
}

func (b *BotAPI) invalidateMember(chatID, userID int64) {
	b.mu.Lock()
	delete(b.memberCache, memberKey(chatID, userID))
	b.mu.Unlock()
}

// rawMessage returns the gotdbot message for (chatID, msgID), served from a
// short-lived bounded cache so that resolving reply/pin targets does not issue
// a blocking GetMessage round-trip for every referencing message.
func (b *BotAPI) rawMessage(chatID, msgID int64) *gotdbot.Message {
	if chatID == 0 || msgID == 0 {
		return nil
	}
	key := memberKey(chatID, msgID)
	now := time.Now()
	b.mu.Lock()
	if c, ok := b.msgCache[key]; ok && now.Before(c.exp) {
		b.mu.Unlock()
		return c.m
	}
	b.mu.Unlock()

	m, err := b.client.GetMessage(chatID, msgID)
	if err != nil || m == nil {
		return nil
	}
	b.mu.Lock()
	if len(b.msgCache) >= msgCacheMax {
		b.msgCache = make(map[string]cachedMessage)
	}
	b.msgCache[key] = cachedMessage{m: m, exp: now.Add(msgCacheTTL)}
	b.mu.Unlock()
	return m
}

// GetMe returns the bot's own user.
func (b *BotAPI) GetMe() (User, error) {
	if b.Self.ID == 0 {
		u, err := b.client.GetMe()
		if err != nil {
			return User{}, err
		}
		b.Self = convertUser(u)
	}
	return b.Self, nil
}
