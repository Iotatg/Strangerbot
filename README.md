# Strangerbot

Anonymous one-to-one chat bot for Telegram, built in Go on the [gotdbot](https://github.com/Iotatg/gotdbot) TDLib transport.

Strangers are matched by shared interests, then exchange messages anonymously. The bot relays text and media, never leaks usernames or links, and ships a full admin panel.

## Credits

- GitHub: [rohon-x04](https://github.com/rohon-x04)
- Telegram: [@Rohon_x04](https://t.me/Rohon_x04)
- Channel: [@im_iota](https://t.me/im_iota)

## Features

- Interest based matching with deterministic tie-breaking
- Anonymous relay for text, photo, sticker, voice, video, video note, animation, audio and documents
- Profile: age, gender, country, interests (with limits)
- Daily bonus with streak, free/premium daily search limits
- Like / dislike, block list, premium users, bonus search pool
- Reply threading (`/del` and reply-to-message deletion for premium users)
- Reports with admin resolve / ban actions
- Icebreakers, games, leaderboard, rank, balance, live stats
- Maintenance mode, rate limiting, admin broadcast, feedback inbox
- 🛡 Trust score / reputation leaderboard (`/trust`, `/trustlist`)
- 🎯 Age and country match filters (`/find`)
- 🌍 Premium users pick India or foreign matches from `/search` buttons
- 🎁 Referral program with bonus searches (`/referral`)
- 🔒 Privacy mode with auto-deletion of relayed messages (`/privacy`)
- ⏳ Flood protection and idle-chat auto-end
- 🎁 Gift premium to another user (`/gift`)
- 📦 Data export and account deletion (`/mydata`, `/deleteaccount`)
- 📨 Ban appeals (`/appeal`)
- 📊 Daily admin digest and premium expiry sweeper
- 🛡 Security layer: strike system, auto-mute / auto-ban, human check, content scanning (phone, email, invite links, scam phrases), forward blocking, protected content, duplicate-message detection, search cooldown, report-triggered auto-ban and an admin audit log
- JSON storage with atomic writes, or MongoDB (one document per user)

## Commands

| User | Description |
| --- | --- |
| `/start` | Restart or register |
| `/help` | Usage guide |
| `/profile` | View or edit your profile |
| `/interests` | Pick interests for better matches |
| `/daily` | Claim daily bonus searches |
| `/search` `/next` | Find a stranger |
| `/stop` | End the current chat |
| `/icebreaker` | Send a conversation starter |
| `/link` | Share your Telegram profile |
| `/block` `/blocklist` `/unblock` | Manage blocked users |
| `/del` | Delete a message for both sides (premium) |
| `/report` | Report the current partner |
| `/leaderboard` `/rank` `/balance` | Leaderboard and balance |
| `/trust` | Your trust score |
| `/find` | Age / country match filters |
| `/referral` | Invite link and referral count |
| `/privacy` | Auto-delete chat messages |
| `/gift` | Gift premium to a user |
| `/mydata` `/deleteaccount` | Data export and deletion |
| `/appeal` | Appeal a ban |
| `/safety` | Community safety rules |
| `/stats` `/about` `/feedback` | Bot info |
| `/8ball` `/dice` `/coin` `/rps` `/ttt` `/choose` | Games |
| `/ping` `/id` `/cancel` | Utilities |

| Admin | Description |
| --- | --- |
| `/adminstats` | Bot statistics |
| `/ban` `/unban` | Ban or unban a user |
| `/premium` `/unpremium` | Grant or revoke premium |
| `/addcoin` | Add bonus searches |
| `/broadcast` | Message all users |
| `/reports` `/resolve` | Review and close reports |
| `/userinfo` | Inspect a user record |
| `/trustlist` | Trust leaderboard |
| `/maintenance` | Toggle maintenance mode |
| `/massban` | Ban several users at once |
| `/purge` | Clear all reports |
| `/digest` | Send the stats digest now |
| `/security` | Security configuration status |
| `/strikes` | Show or reset a user's strikes |
| `/unmute` | Lift a mute |
| `/audit` | Recent admin actions |

## Configuration

Copy `.env.example` to `.env` and fill it in, or export the same variables.

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `BOT_TOKEN` | yes | - | Bot token |
| `API_ID` | yes | - | Telegram app id |
| `API_HASH` | yes | - | Telegram app hash |
| `ADMIN_IDS` | no | - | Comma separated admin ids |
| `PORT` | no | - | Health server port (`/healthz`) |
| `DB_DRIVER` | no | `json` | Storage backend: `json` or `mongodb` |
| `DB_PATH` | no | `data/db.json` | JSON store path (`json` driver) |
| `MONGO_URI` | when `mongodb` | - | MongoDB connection string |
| `MONGO_DB` | no | `strangerbot` | MongoDB database name |
| `TDLIB_LIBRARY_PATH` | no | `lib/libtdjson.so` | TDLib shared library |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, `error` |
| `CHAT_IDLE_MINUTES` | no | `15` | End chats after this much inactivity |
| `FLOOD_LIMIT` | no | `10` | Messages allowed per flood window |
| `FLOOD_WINDOW_SECONDS` | no | `10` | Flood window length |
| `PRIVACY_DELETE_MINUTES` | no | `5` | Delay before privacy-mode messages are deleted |
| `DIGEST_HOUR` | no | `9` | UTC hour for the admin digest |
| `REFERRAL_REWARD` | no | `5` | Bonus searches per referral |
| `GIFT_PREMIUM_DAYS` | no | `7` | Days granted by `/gift` |
| `MAX_STRIKES` | no | `3` | Violations before auto-ban |
| `MUTE_MINUTES` | no | `5` | Mute duration on the second strike |
| `SEARCH_COOLDOWN_SECONDS` | no | `5` | Minimum gap between searches |
| `CAPTCHA_REQUIRED` | no | `true` | Require a one-tap human check before the first chat |
| `AUTO_BAN_REPORTS` | no | `3` | Open reports that trigger an auto-ban |
| `PROTECT_CONTENT` | no | `true` | Mark relayed messages as protected from forwarding |
| `BLOCK_FORWARDS` | no | `true` | Reject forwarded messages |

See `.env.example` for all tuning knobs (search limits, bonus steps, maintenance mode).

## Running

Requires the TDLib shared library (`lib/libtdjson.so`). Point `TDLIB_LIBRARY_PATH` at your copy.

```bash
# Build
go build ./cmd/strangerbot

# Run
BOT_TOKEN=... API_ID=... API_HASH=... ./strangerbot
```

Docker:

```bash
docker build -t strangerbot .
docker run -d --name strangerbot \
  -v "$PWD/data:/app/data" \
  --env-file .env \
  strangerbot
```

## Layout

```
cmd/strangerbot      entrypoint, signal handling, health server
internal/bot         dispatch, commands, matching, relay, admin, games
internal/config      environment configuration
internal/store       JSON store or MongoDB backend
internal/tgapi       minimal tgbotapi-compatible layer over gotdbot
```

## Tests

```bash
go test ./...
```