package bot

import (
	"fmt"
	"time"

	tgbotapi "strangerbot/internal/tgapi"
)

// relayEntry is a relayed message pair kept for privacy auto-deletion.
type relayEntry struct {
	from     int64
	to       int64
	sentMid  int64
	recvMid  int64
	at       time.Time
	deleteAt time.Time
}

// touchActivity records that uid was active right now.
func (b *Bot) touchActivity(uid int64) {
	b.activityMu.Lock()
	b.activity[uid] = time.Now()
	b.activityMu.Unlock()
}

// recordRelay stores a relayed message for the privacy sweeper.
func (b *Bot) recordRelay(from, to, sentMid, recvMid int64) {
	u1, ok1 := b.st.GetUser(from)
	u2, ok2 := b.st.GetUser(to)
	if (!ok1 || !u1.Privacy) && (!ok2 || !u2.Privacy) {
		return
	}
	b.relayMu.Lock()
	defer b.relayMu.Unlock()
	b.relayLog = append(b.relayLog, relayEntry{
		from: from, to: to, sentMid: sentMid, recvMid: recvMid,
		at:       time.Now(),
		deleteAt: time.Now().Add(time.Duration(b.cfg.PrivacyDeleteMinutes) * time.Minute),
	})
}

// startSweepers runs the idle-chat and privacy cleanup loops.
func (b *Bot) startSweepers() {
	go b.idleLoop()
	go b.privacyLoop()
	go b.digestLoop()
	go b.premiumLoop()
	go b.startGamesLoop()
}

func (b *Bot) idleLoop() {
	interval := time.Duration(max(1, b.cfg.ChatIdleMinutes)) * time.Minute / 2
	if interval < time.Minute {
		interval = time.Minute
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for range t.C {
		b.sweepIdle()
	}
}

func (b *Bot) sweepIdle() {
	limit := time.Duration(b.cfg.ChatIdleMinutes) * time.Minute
	now := time.Now()

	b.activityMu.Lock()
	snapshot := make(map[int64]time.Time, len(b.activity))
	for k, v := range b.activity {
		snapshot[k] = v
	}
	b.activityMu.Unlock()

	for _, pair := range b.st.AllPairs() {
		last := snapshot[pair[0]]
		if other, ok := snapshot[pair[1]]; ok && other.After(last) {
			last = other
		}
		if !last.IsZero() && now.Sub(last) < limit {
			continue
		}
		b.st.RemovePair(pair[0])
		b.clearRatedPair(pair[0], pair[1])
		b.sendText(pair[0], ChatIdleEnded, MainMenu())
		b.sendText(pair[1], ChatIdleEnded, MainMenu())
	}
}

func (b *Bot) privacyLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		b.sweepPrivacy()
	}
}

func (b *Bot) sweepPrivacy() {
	now := time.Now()

	b.relayMu.Lock()
	kept := b.relayLog[:0]
	var expired []relayEntry
	for _, e := range b.relayLog {
		if now.After(e.deleteAt) {
			expired = append(expired, e)
			continue
		}
		kept = append(kept, e)
	}
	b.relayLog = kept
	b.relayMu.Unlock()

	for _, e := range expired {
		b.deleteMsg(e.from, e.sentMid)
		b.deleteMsg(e.to, e.recvMid)
		b.msgs.Forget(e.from, e.sentMid)
	}
}

// cmdPrivacy toggles auto-deletion of relayed messages.
func (b *Bot) cmdPrivacy(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	switch normalizeToggle(args) {
	case "on":
		b.st.SetPrivacy(uid, true)
		b.sendText(uid, PrivacyOn, nil)
	case "off":
		b.st.SetPrivacy(uid, false)
		b.sendText(uid, PrivacyOff, nil)
	default:
		state := "off"
		if u.Privacy {
			state = "on"
		}
		b.sendText(uid, fmt.Sprintf(PrivacyStatus, state, b.cfg.PrivacyDeleteMinutes), nil)
	}
}

func normalizeToggle(s string) string {
	switch s {
	case "on", "1", "true", "yes", "enable":
		return "on"
	case "off", "0", "false", "no", "disable":
		return "off"
	}
	return ""
}
