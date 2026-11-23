package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Interest is a single selectable interest in the profile picker.
type Interest struct {
	Key   string
	Label string
}

// Config holds every runtime setting. Values come from the environment (or a
// local .env file) so the bot can be deployed without code changes.
type Config struct {
	Token   string
	APIID   int
	APIHash string

	AdminIDs []int64
	BotName  string

	TDLibLibraryPath string
	TDLibDatabaseDir string
	TDLibFilesDir    string

	DBPath string

	DBDriver      string
	MongoURI      string
	MongoDatabase string

	LogLevel string
	Port     string

	SupportLink string

	FreeDailySearches    int
	PremiumDailySearches int
	MaxBlockList         int
	MinAge               int
	MaxAge               int
	MaxInterests         int

	DailyBonusBase int
	DailyBonusStep int
	DailyBonusMax  int
	IcebreakerCD   int

	LikeReward  int
	DislikeCost int

	ChatIdleMinutes      int
	FloodLimit           int
	FloodWindowSeconds   int
	PrivacyDeleteMinutes int
	DigestHour           int
	ReferralReward       int
	GiftPremiumDays      int

	MaxStrikes            int
	MuteMinutes           int
	SearchCooldownSeconds int
	CaptchaRequired       bool
	AutoBanReports        int
	ProtectContent        bool
	BlockForwards         bool

	Maintenance  bool
	NotifyReport bool
	MaxRelaySize int
}

// Interests is the ordered list shown in the picker.
var Interests = []Interest{
	{"music", "🎵 Music"},
	{"gaming", "🎮 Gaming"},
	{"movies", "🎬 Movies"},
	{"anime", "🍥 Anime"},
	{"sports", "⚽ Sports"},
	{"coding", "💻 Coding"},
	{"travel", "✈️ Travel"},
	{"books", "📚 Books"},
	{"memes", "😂 Memes"},
	{"fitness", "💪 Fitness"},
	{"food", "🍕 Food"},
	{"art", "🎨 Art"},
}

// Labels returns a key -> label lookup map.
func Labels() map[string]string {
	m := make(map[string]string, len(Interests))
	for _, in := range Interests {
		m[in.Key] = in.Label
	}
	return m
}

// Valid reports whether key is a known interest.
func Valid(key string) bool {
	_, ok := Labels()[key]
	return ok
}

// Label renders a key as its human label, falling back to the key.
func Label(key string) string {
	if l, ok := Labels()[key]; ok {
		return l
	}
	return key
}

// LabelList renders keys as a comma separated list of labels.
func LabelList(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, Label(k))
	}
	return strings.Join(out, ", ")
}

// Load reads configuration from the environment and an optional .env file.
func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Token:                 strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		APIID:                 parseInt(os.Getenv("API_ID")),
		APIHash:               strings.TrimSpace(os.Getenv("API_HASH")),
		BotName:               envOr("BOT_NAME", "StrangerBot"),
		TDLibLibraryPath:      envOr("TDLIB_LIBRARY_PATH", "lib/libtdjson.so"),
		TDLibDatabaseDir:      envOr("TDLIB_DATABASE_DIR", "data/tdlib/db"),
		TDLibFilesDir:         envOr("TDLIB_FILES_DIR", "data/tdlib/files"),
		DBPath:                envOr("DB_PATH", "data/db.json"),
		DBDriver:              envOr("DB_DRIVER", "json"),
		MongoURI:              firstEnv("MONGO_URI", "MONGO_DB_URI"),
		MongoDatabase:         envOr("MONGO_DB", "strangerbot"),
		LogLevel:              envOr("LOG_LEVEL", "info"),
		Port:                  strings.TrimSpace(os.Getenv("PORT")),
		SupportLink:           envOr("SUPPORT_LINK", "https://t.me/+Q493A_VBAUpkMmRl"),
		FreeDailySearches:     envInt("FREE_DAILY_SEARCHES", 100),
		PremiumDailySearches:  envInt("PREMIUM_DAILY_SEARCHES", 200),
		MaxBlockList:          envInt("MAX_BLOCK_LIST", 30),
		MinAge:                envInt("MIN_AGE", 18),
		MaxAge:                envInt("MAX_AGE", 80),
		MaxInterests:          envInt("MAX_INTERESTS", 5),
		DailyBonusBase:        envInt("DAILY_BONUS_BASE", 3),
		DailyBonusStep:        envInt("DAILY_BONUS_STEP", 1),
		DailyBonusMax:         envInt("DAILY_BONUS_MAX", 10),
		IcebreakerCD:          envInt("ICEBREAKER_COOLDOWN", 20),
		LikeReward:            envInt("LIKE_REWARD", 1),
		DislikeCost:           envInt("DISLIKE_COST", 1),
		ChatIdleMinutes:       envInt("CHAT_IDLE_MINUTES", 15),
		FloodLimit:            envInt("FLOOD_LIMIT", 10),
		FloodWindowSeconds:    envInt("FLOOD_WINDOW_SECONDS", 10),
		PrivacyDeleteMinutes:  envInt("PRIVACY_DELETE_MINUTES", 5),
		DigestHour:            envInt("DIGEST_HOUR", 9),
		ReferralReward:        envInt("REFERRAL_REWARD", 5),
		GiftPremiumDays:       envInt("GIFT_PREMIUM_DAYS", 7),
		MaxStrikes:            envInt("MAX_STRIKES", 3),
		MuteMinutes:           envInt("MUTE_MINUTES", 5),
		SearchCooldownSeconds: envInt("SEARCH_COOLDOWN_SECONDS", 5),
		CaptchaRequired:       envBool("CAPTCHA_REQUIRED", true),
		AutoBanReports:        envInt("AUTO_BAN_REPORTS", 3),
		ProtectContent:        envBool("PROTECT_CONTENT", true),
		BlockForwards:         envBool("BLOCK_FORWARDS", true),
		Maintenance:           envBool("MAINTENANCE", false),
		NotifyReport:          envBool("NOTIFY_ADMINS_ON_REPORT", true),
		MaxRelaySize:          envInt("MAX_RELAY_SIZE", 4096),
	}
	cfg.AdminIDs = parseIDs(os.Getenv("ADMIN_IDS"))
	return cfg
}

// Validate checks that required settings are present.
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("missing required environment variable: BOT_TOKEN")
	}
	if c.APIID == 0 {
		return fmt.Errorf("missing required environment variable: API_ID")
	}
	if c.APIHash == "" {
		return fmt.Errorf("missing required environment variable: API_HASH")
	}
	switch strings.ToLower(strings.TrimSpace(c.DBDriver)) {
	case "", "json", "file":
	case "mongo", "mongodb":
		if c.MongoURI == "" {
			return fmt.Errorf("missing required environment variable: MONGO_URI")
		}
		if c.MongoDatabase == "" {
			return fmt.Errorf("MONGO_DB must not be empty when DB_DRIVER=mongodb")
		}
	default:
		return fmt.Errorf("unsupported DB_DRIVER %q (use json or mongodb)", c.DBDriver)
	}
	return nil
}

// firstEnv returns the first non-empty environment variable value.
func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return def
}

func parseIDs(raw string) []int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []int64
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if n, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			ids = append(ids, n)
		}
	}
	return ids
}

// IsAdmin reports whether uid is configured as an administrator.
func (c *Config) IsAdmin(uid int64) bool {
	for _, id := range c.AdminIDs {
		if id == uid {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
