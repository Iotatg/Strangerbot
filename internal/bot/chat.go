package bot

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"strangerbot/internal/config"
	"strangerbot/internal/store"

	tgbotapi "strangerbot/internal/tgapi"
)

func (b *Bot) cmdSearch(m *tgbotapi.Message) {
	uid := m.From.ID
	if b.st.IsInChat(uid) {
		b.cmdNext(m)
		return
	}
	if m.Command() == "next" && b.st.IsWaiting(uid) {
		b.st.RemoveWaiting(uid)
	}
	if u, ok := b.st.GetUser(uid); ok && u.Premium {
		b.sendText(uid, SearchRegionPrompt, SearchRegionPicker())
		return
	}
	b.startSearchFor(uid, "")
}

func (b *Bot) findPartner(m *tgbotapi.Message, want string) {
	if b.st.IsInChat(m.From.ID) {
		b.cmdNext(m)
		return
	}
	b.startSearchFor(m.From.ID, want)
}

// cbSearchRegion handles the India / Foreign buttons shown to premium users.
func (b *Bot) cbSearchRegion(cq *tgbotapi.CallbackQuery, region string) {
	uid := cq.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Premium {
		b.answerCallback(cq, "🔒 Premium feature.", true)
		return
	}
	if region != "india" && region != "foreign" {
		b.answerCallback(cq, "", false)
		return
	}
	b.st.SetRegion(uid, region)
	if cq.Message != nil {
		b.deleteMsg(cq.Message.Chat.ID, int64(cq.Message.MessageID))
	}
	b.answerCallback(cq, "", false)
	b.startSearchFor(uid, "")
}

func (b *Bot) startSearchFor(uid int64, want string) {
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	if !b.requireCaptcha(uid) {
		return
	}
	if !b.spam.searchAllowed(uid, time.Duration(b.cfg.SearchCooldownSeconds)*time.Second) {
		b.sendText(uid, SlowDown, nil)
		return
	}
	if want != "" {
		if premiumEnabled && !u.Premium {
			b.sendText(uid, GenderSearchPremium+premiumText(u.Premium, b.cfg), PremiumPayment())
			return
		}
	}
	if b.st.IsInChat(uid) {
		b.sendText(uid, AlreadyInChat, nil)
		return
	}
	if b.st.IsWaiting(uid) {
		b.sendText(uid, AlreadySearching, nil)
		return
	}
	if b.st.SearchesLeft(uid) <= 0 {
		b.sendText(uid, fmt.Sprintf(NoSearchesLeft, b.cfg.SupportLink), nil)
		return
	}
	b.st.AddWaiting(uid, want)
	b.sendText(uid, Searching, nil)
	b.tryMatch(uid)
}

func badge(premium bool) string {
	if premium {
		return "🧸 Premium"
	}
	return "Normal user"
}

func (b *Bot) tryMatch(uid int64) {
	b.matchMu.Lock()
	waiting := b.st.GetWaiting()
	var mine *store.Waiting
	for i := range waiting {
		if waiting[i].ID == uid {
			mine = &waiting[i]
			break
		}
	}
	if mine == nil {
		b.matchMu.Unlock()
		return
	}
	me, ok := b.st.GetUser(uid)
	if !ok {
		b.matchMu.Unlock()
		return
	}
	var eligible []candidate
	for _, entry := range waiting {
		cid := entry.ID
		if cid == uid {
			continue
		}
		if b.st.IsInChat(cid) {
			b.st.RemoveWaiting(cid)
			continue
		}
		cu, ok := b.st.GetUser(cid)
		if !ok || !cu.Registered || cu.Banned {
			b.st.RemoveWaiting(cid)
			continue
		}
		if b.st.IsBlockedBy(uid, cid) || b.st.IsBlockedBy(cid, uid) {
			continue
		}
		if !compatible(me.Gender, mine.Want, cu.Gender, entry.Want) {
			continue
		}
		if !b.filtersAllow(me, cu) {
			continue
		}
		if b.st.SearchesLeft(cid) <= 0 {
			b.st.RemoveWaiting(cid)
			continue
		}
		eligible = append(eligible, candidate{ID: cid, Gender: cu.Gender, Interests: cu.Interests, Want: entry.Want})
	}
	best, ok := pickBest(me.Interests, eligible)
	if !ok {
		b.matchMu.Unlock()
		return
	}
	if !b.st.TryPair(uid, best.ID) {
		b.matchMu.Unlock()
		return
	}
	b.st.ConsumeSearch(uid)
	b.st.ConsumeSearch(best.ID)
	b.matchMu.Unlock()

	b.clearRatedPair(uid, best.ID)
	b.touchActivity(uid)
	b.touchActivity(best.ID)
	partner, _ := b.st.GetUser(best.ID)
	common := sharedInterests(me.Interests, partner.Interests)
	commonLine := NoCommon
	if len(common) > 0 {
		commonLine = fmt.Sprintf(CommonInterests, config.LabelList(common))
	}
	b.sendText(uid, fmt.Sprintf(PartnerFound, badge(partner.Premium), commonLine), nil)
	b.sendText(best.ID, fmt.Sprintf(PartnerFound, badge(me.Premium), commonLine), nil)
	b.sendSafetyOnce(uid)
	b.sendSafetyOnce(best.ID)
}

func (b *Bot) cmdNext(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !u.Registered {
		b.sendText(uid, NotRegistered, nil)
		return
	}
	if pid, ok := b.st.RemovePair(uid); ok {
		b.clearRatedPair(uid, pid)
		b.sendText(pid, ChatEndedPartner, nil)
	}
	if b.st.SearchesLeft(uid) <= 0 {
		b.st.RemoveWaiting(uid)
		b.sendText(uid, fmt.Sprintf(NoSearchesLeft, b.cfg.SupportLink), nil)
		return
	}
	b.sendText(uid, NextSearching, nil)
	b.st.AddWaiting(uid, "")
	b.tryMatch(uid)
}

func (b *Bot) cmdStop(m *tgbotapi.Message) {
	uid := m.From.ID
	b.states.clear(uid)
	if !b.st.IsInChat(uid) {
		b.st.RemoveWaiting(uid)
		b.sendText(uid, NotInChat, MainMenu())
		return
	}
	b.endChat(uid, true)
}

func (b *Bot) endChat(uid int64, notifySelf bool) {
	b.cancelGamesFor(uid)
	pid, ok := b.st.RemovePair(uid)
	b.st.RemoveWaiting(uid)
	if ok {
		b.clearRatedPair(uid, pid)
		b.sendText(uid, ChatEndedYou, PostChat(pid, true))
		b.sendText(pid, ChatEndedPartner, nil)
		return
	}
	if notifySelf {
		b.sendText(uid, ChatEndedYou, MainMenu())
	}
}

func (b *Bot) cmdIcebreaker(m *tgbotapi.Message) {
	uid := m.From.ID
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, IcebreakerNotInChat, nil)
		return
	}
	b.iceMu.Lock()
	last := b.iceLast[uid]
	wait := 0
	if !last.IsZero() {
		wait = b.cfg.IcebreakerCD - int(time.Since(last).Seconds())
	}
	if wait > 0 {
		b.iceMu.Unlock()
		b.sendText(uid, fmt.Sprintf(IcebreakerCooldown, wait), nil)
		return
	}
	b.iceLast[uid] = time.Now()
	b.iceMu.Unlock()

	text := fmt.Sprintf(IcebreakerMsg, Icebreakers[rand.Intn(len(Icebreakers))])
	b.sendText(uid, text, nil)
	b.sendText(pid, text, nil)
}

func isRelayable(m *tgbotapi.Message) bool {
	return m.Text != "" || len(m.Photo) > 0 || m.Voice != nil || m.VideoNote != nil ||
		m.Audio != nil || m.Animation != nil || m.Video != nil || m.Sticker != nil || m.Document != nil
}

func (b *Bot) relay(m *tgbotapi.Message) {
	uid := m.From.ID
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		return
	}
	if m.Text != "" && strings.HasPrefix(m.Text, "/") {
		return
	}
	if b.cfg.BlockForwards && (m.ForwardFrom != nil || m.ForwardFromChat != nil) {
		b.strike(uid, "forwarded messages are not allowed")
		return
	}
	if b.st.IsMuted(uid) {
		b.sendText(uid, StillMuted, nil)
		return
	}
	if v := scanContent(m.Text+" "+m.Caption, false); v.blocked {
		b.strike(uid, v.reason)
		return
	}
	if b.spam.duplicate(uid, m.Text, 3) {
		b.strike(uid, "repeated messages")
		return
	}
	if !isRelayable(m) {
		b.sendText(uid, Unsupported, nil)
		return
	}
	if !b.rl.allow(uid) {
		return
	}
	replyTo := int64(0)
	if m.ReplyToMessage != nil {
		if tgt, ok := b.msgs.ReplyTarget(uid, int64(m.ReplyToMessage.MessageID)); ok && tgt.chat == pid {
			replyTo = tgt.mid
		}
	}
	msg, err := b.sendRelay(pid, m, replyTo)
	if err != nil {
		b.log.Warn("relay failed", "from", uid, "to", pid, "err", err)
		b.st.RemovePair(uid)
		b.sendText(uid, PartnerLeftBot, MainMenu())
		return
	}
	if msg != nil {
		b.msgs.Record(uid, int64(m.MessageID), pid, int64(msg.MessageID))
		b.recordRelay(uid, pid, int64(m.MessageID), int64(msg.MessageID))
	}
	b.touchActivity(uid)
	b.touchActivity(pid)
}

func (b *Bot) sendRelay(pid int64, m *tgbotapi.Message, replyTo int64) (*tgbotapi.Message, error) {
	var cfg tgbotapi.Chattable
	switch {
	case m.Text != "":
		c := tgbotapi.NewMessage(pid, m.Text)
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case len(m.Photo) > 0:
		c := tgbotapi.NewPhoto(pid, tgbotapi.FileID(m.Photo[len(m.Photo)-1].FileID))
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.VideoNote != nil:
		c := tgbotapi.NewVideoNote(pid, tgbotapi.FileID(m.VideoNote.FileID))
		c.Duration = m.VideoNote.Duration
		c.Length = m.VideoNote.Length
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Voice != nil:
		c := tgbotapi.NewVoice(pid, tgbotapi.FileID(m.Voice.FileID))
		c.Duration = m.Voice.Duration
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Animation != nil:
		c := tgbotapi.NewAnimation(pid, tgbotapi.FileID(m.Animation.FileID))
		c.Duration = m.Animation.Duration
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Audio != nil:
		c := tgbotapi.NewAudio(pid, tgbotapi.FileID(m.Audio.FileID))
		c.Duration = m.Audio.Duration
		c.Title = m.Audio.Title
		c.Performer = m.Audio.Performer
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Video != nil:
		c := tgbotapi.NewVideo(pid, tgbotapi.FileID(m.Video.FileID))
		c.Duration = m.Video.Duration
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Sticker != nil:
		c := tgbotapi.NewSticker(pid, tgbotapi.FileID(m.Sticker.FileID))
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	case m.Document != nil:
		c := tgbotapi.NewDocument(pid, tgbotapi.FileID(m.Document.FileID))
		c.Caption = m.Caption
		c.ReplyToMessageID = int(replyTo)
		c.ProtectContent = b.cfg.ProtectContent
		cfg = c
	default:
		return nil, nil
	}
	msg, err := b.api.Send(cfg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (b *Bot) cmdLink(m *tgbotapi.Message) {
	uid := m.From.ID
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, LinkNotInChat, nil)
		return
	}
	if m.From.UserName == "" {
		b.sendText(uid, LinkNoUsername, nil)
		return
	}
	b.sendText(pid, fmt.Sprintf(LinkPartner, m.From.UserName), nil)
	b.sendText(uid, LinkOK, nil)
}

func (b *Bot) cmdBlock(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok {
		return
	}
	if !b.isPremium(u) {
		b.sendText(uid, BlockPremiumOnly, nil)
		return
	}
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, BlockNotInChat, nil)
		return
	}
	switch b.st.BlockUser(uid, pid) {
	case "already":
		b.sendText(uid, BlockAlready, nil)
	case "limit":
		b.sendText(uid, fmt.Sprintf(BlockLimit, b.cfg.MaxBlockList), nil)
	default:
		b.sendText(uid, BlockOK, nil)
	}
}

func (b *Bot) cmdBlocklist(m *tgbotapi.Message) {
	uid := m.From.ID
	bl := b.st.GetBlocklist(uid)
	if len(bl) == 0 {
		b.sendText(uid, BlocklistEmpty, nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(BlocklistHeader)
	for i, id := range bl {
		sb.WriteString(fmt.Sprintf(BlocklistRow+"\n", i+1, id))
	}
	sb.WriteString("\n<i>Use /unblock &lt;id&gt; to remove.</i>")
	b.sendText(uid, sb.String(), nil)
}

func (b *Bot) cmdUnblock(m *tgbotapi.Message, args string) {
	args = strings.TrimSpace(args)
	if args == "" {
		b.sendText(m.From.ID, UnblockUsage, nil)
		return
	}
	n, err := strconv.Atoi(args)
	if err != nil {
		b.sendText(m.From.ID, UnblockUsage, nil)
		return
	}
	if b.st.UnblockByIndex(m.From.ID, n) == "ok" {
		b.sendText(m.From.ID, UnblockOK, nil)
	} else {
		b.sendText(m.From.ID, UnblockNotFound, nil)
	}
}

func (b *Bot) cmdDel(m *tgbotapi.Message) {
	uid := m.From.ID
	u, ok := b.st.GetUser(uid)
	if !ok || !b.isPremium(u) {
		b.sendText(uid, DelPremiumOnly, nil)
		return
	}
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, DelNotInChat, nil)
		return
	}
	if m.ReplyToMessage == nil {
		b.sendText(uid, DelReplyNeeded, nil)
		return
	}
	own, ok := b.msgs.OwnMessage(uid, int64(m.ReplyToMessage.MessageID))
	if !ok || own.chat != pid {
		b.sendText(uid, DelNotYourMsg, nil)
		return
	}
	b.deleteMsg(pid, own.mid)
	b.deleteMsg(uid, int64(m.ReplyToMessage.MessageID))
	b.deleteMsg(uid, int64(m.MessageID))
	b.msgs.Forget(uid, int64(m.ReplyToMessage.MessageID))
	b.sendText(uid, DelOK, nil)
}

func (b *Bot) deleteMsg(chatID, mid int64) {
	if _, err := b.api.Send(tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: int(mid)}); err != nil {
		b.log.Debug("delete failed", "chat", chatID, "msg", mid, "err", err)
	}
}

func (b *Bot) cmdReport(m *tgbotapi.Message, args string) {
	uid := m.From.ID
	pid, ok := b.st.GetPartner(uid)
	if !ok {
		b.sendText(uid, ReportNotInChat, nil)
		return
	}
	reason := strings.TrimSpace(args)
	if reason == "" {
		b.sendText(uid, ReportUsage, nil)
		return
	}
	if len(reason) > 300 {
		reason = reason[:300]
	}
	r := b.st.AddReport(uid, pid, reason)
	b.sendText(uid, ReportOK, nil)
	b.notifyAdminsOfReport(r)
	b.afterReport(pid, r)
}

func (b *Bot) reportReason(m *tgbotapi.Message) {
	uid := m.From.ID
	st := b.states.get(uid)
	target := st.ReportTarget
	reason := strings.TrimSpace(m.Text)
	if reason == "" || target == 0 {
		b.sendText(uid, "Please send a reason as text.", nil)
		return
	}
	if len(reason) > 300 {
		reason = reason[:300]
	}
	r := b.st.AddReport(uid, target, reason)
	b.states.clear(uid)
	b.sendText(uid, ReportOK, nil)
	b.notifyAdminsOfReport(r)
	b.afterReport(target, r)
}

func (b *Bot) ratedKey(a, c int64) string { return itoa(a) + ":" + itoa(c) }

func (b *Bot) isRated(a, c int64) bool {
	b.ratedMu.Lock()
	defer b.ratedMu.Unlock()
	return b.rated[b.ratedKey(a, c)]
}

func (b *Bot) markRated(a, c int64) {
	b.ratedMu.Lock()
	b.rated[b.ratedKey(a, c)] = true
	b.ratedMu.Unlock()
}

func (b *Bot) clearRatedPair(a, c int64) {
	b.ratedMu.Lock()
	delete(b.rated, b.ratedKey(a, c))
	delete(b.rated, b.ratedKey(c, a))
	b.ratedMu.Unlock()
}

func (b *Bot) cbRate(cq *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	pid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	uid := cq.From.ID
	if b.isRated(uid, pid) {
		b.answerCallback(cq, AlreadyRated, true)
		return
	}
	b.markRated(uid, pid)
	if parts[1] == "like" {
		b.st.ApplyLike(uid, pid)
		b.answerCallback(cq, RatedLike, true)
	} else {
		b.st.ApplyDislike(uid, pid)
		b.answerCallback(cq, RatedDislike, true)
	}
	if cq.Message != nil {
		kb := PostChatMinimal(pid)
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, cq.Message.Text, &kb)
	}
}

func (b *Bot) cbPostBlock(cq *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	pid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	uid := cq.From.ID
	u, _ := b.st.GetUser(uid)
	if premiumEnabled && (u.ID == 0 || !u.Premium) {
		b.answerCallback(cq, BlockPremiumOnly, true)
		return
	}
	switch b.st.BlockUser(uid, pid) {
	case "ok":
		b.answerCallback(cq, BlockOK, true)
	case "already":
		b.answerCallback(cq, BlockAlready, true)
	case "limit":
		b.answerCallback(cq, fmt.Sprintf(BlockLimit, b.cfg.MaxBlockList), true)
	default:
		b.answerCallback(cq, "", false)
	}
	if cq.Message != nil {
		kb := PostChatReportOnly(pid)
		b.editText(cq.Message.Chat.ID, cq.Message.MessageID, cq.Message.Text, &kb)
	}
}

func (b *Bot) cbPostReportStart(cq *tgbotapi.CallbackQuery) {
	parts := strings.SplitN(cq.Data, ":", 3)
	if len(parts) != 3 {
		b.answerCallback(cq, "", false)
		return
	}
	pid, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		b.answerCallback(cq, "", false)
		return
	}
	st := b.states.get(cq.From.ID)
	st.ReportTarget = pid
	b.states.setStep(cq.From.ID, "report_reason")
	b.sendText(cq.From.ID, "🚨 <b>Report</b>\n\nSend a brief reason (e.g. spam, harassment):", nil)
	b.answerCallback(cq, "", false)
}
