package tgbotapi

import (
	"sort"
	"strconv"
	"time"

	"github.com/Iotatg/gotdbot"
)

// additionalCommandPrefixes are characters, beyond Telegram's native "/", that
// this bot also accepts as command prefixes. Telegram only emits a bot_command
// entity for "/", so a synthetic entity is added during conversion for these.
var additionalCommandPrefixes = []byte{'!', '.'}

func isASCIIAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// syntheticCommandLength reports whether text starts with a "!cmd" or ".cmd"
// command token and, if so, its length including the prefix character. The
// token stops at the first byte that is not a command-name character, so the
// returned length is byte-accurate (command names are ASCII).
func syntheticCommandLength(text string) (int, bool) {
	if len(text) < 2 || !containsPrefix(additionalCommandPrefixes, text[0]) {
		return 0, false
	}
	first := text[1]
	if first != '_' && !isASCIIAlpha(first) {
		return 0, false
	}
	i := 1
	for i < len(text) {
		ch := text[i]
		if ch == '_' || ch == '@' || isASCIIAlpha(ch) || (ch >= '0' && ch <= '9') {
			i++
			continue
		}
		break
	}
	return i, true
}

func containsPrefix(prefixes []byte, b byte) bool {
	for _, p := range prefixes {
		if p == b {
			return true
		}
	}
	return false
}

// addSyntheticCommand prepends a bot_command entity when a message uses one of
// the additional prefixes and Telegram did not already mark it as a command.
func (b *BotAPI) addSyntheticCommand(out *Message) {
	if out.Text == "" {
		return
	}
	if len(out.Entities) > 0 {
		first := out.Entities[0]
		if first.Offset == 0 && first.IsCommand() {
			return
		}
	}
	n, ok := syntheticCommandLength(out.Text)
	if !ok {
		return
	}
	cmd := MessageEntity{Type: "bot_command", Offset: 0, Length: n}
	out.Entities = append([]MessageEntity{cmd}, out.Entities...)
}

func fileIDOf(f *gotdbot.File) string {
	if f == nil || f.Id == 0 {
		return ""
	}
	return strconv.FormatInt(int64(f.Id), 10)
}

func fileUIDOf(f *gotdbot.File) string {
	if f == nil || f.Remote == nil {
		return ""
	}
	return f.Remote.UniqueId
}

func fileSizeOf(f *gotdbot.File) int {
	if f == nil {
		return 0
	}
	return int(f.Size)
}

func convertUser(u *gotdbot.User) User {
	if u == nil {
		return User{}
	}
	out := User{
		ID:        u.Id,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
	if u.Usernames != nil {
		if len(u.Usernames.ActiveUsernames) > 0 {
			out.UserName = u.Usernames.ActiveUsernames[0]
		} else {
			out.UserName = u.Usernames.EditableUsername
		}
	}
	if bot, ok := u.Type.(*gotdbot.UserTypeBot); ok && bot != nil {
		out.IsBot = true
		out.CanJoinGroups = bot.CanJoinGroups
		out.CanReadAllGroupMessages = bot.CanReadAllGroupMessages
	}
	return out
}

func (b *BotAPI) user(id int64) User {
	if id == 0 {
		return User{}
	}
	now := time.Now()
	b.mu.Lock()
	if c, ok := b.userCache[id]; ok && now.Before(c.exp) {
		b.mu.Unlock()
		return c.u
	}
	b.mu.Unlock()

	u, err := b.client.GetUser(id)
	var out User
	if err != nil || u == nil {
		out = User{ID: id}
	} else {
		out = convertUser(u)
	}
	b.mu.Lock()
	b.userCache[id] = cachedUser{u: out, exp: now.Add(cacheTTL)}
	b.mu.Unlock()
	return out
}

func chatTypeString(ct gotdbot.ChatType) string {
	switch t := ct.(type) {
	case *gotdbot.ChatTypePrivate:
		return "private"
	case *gotdbot.ChatTypeSecret:
		return "private"
	case *gotdbot.ChatTypeBasicGroup:
		return "group"
	case *gotdbot.ChatTypeSupergroup:
		if t != nil && t.IsChannel {
			return "channel"
		}
		return "supergroup"
	}
	return ""
}

func convertChatPhoto(p *gotdbot.ChatPhotoInfo) *ChatPhoto {
	if p == nil {
		return nil
	}
	return &ChatPhoto{
		SmallFileID:       fileIDOf(p.Small),
		SmallFileUniqueID: fileUIDOf(p.Small),
		BigFileID:         fileIDOf(p.Big),
		BigFileUniqueID:   fileUIDOf(p.Big),
	}
}

func (b *BotAPI) chatFromRaw(c *gotdbot.Chat) Chat {
	if c == nil {
		return Chat{}
	}
	out := Chat{
		ID:    c.Id,
		Type:  chatTypeString(c.Type),
		Title: c.Title,
		Photo: convertChatPhoto(c.Photo),
	}
	if t, ok := c.Type.(*gotdbot.ChatTypePrivate); ok && t != nil {
		u := b.user(t.UserId)
		out.FirstName = u.FirstName
		out.LastName = u.LastName
		out.UserName = u.UserName
	}
	return out
}

func (b *BotAPI) chat(id int64) Chat {
	if id == 0 {
		return Chat{}
	}
	now := time.Now()
	b.mu.Lock()
	if c, ok := b.chatCache[id]; ok && now.Before(c.exp) {
		b.mu.Unlock()
		return c.c
	}
	b.mu.Unlock()

	raw, err := b.client.GetChat(id)
	var out Chat
	if err != nil || raw == nil {
		out = Chat{ID: id}
	} else {
		out = b.chatFromRaw(raw)
	}
	b.mu.Lock()
	b.chatCache[id] = cachedChat{c: out, exp: now.Add(cacheTTL)}
	b.mu.Unlock()
	return out
}

func (b *BotAPI) invalidateChat(id int64) {
	b.mu.Lock()
	delete(b.chatCache, id)
	b.mu.Unlock()
}

func (b *BotAPI) convertMessage(m *gotdbot.Message) *Message {
	return b.convertMessageDepth(m, 0)
}

func (b *BotAPI) convertMessageDepth(m *gotdbot.Message, depth int) *Message {
	if m == nil {
		return nil
	}
	b.noteMessage(m.ChatId, m.Id)
	ch := b.chat(m.ChatId)
	out := &Message{
		MessageID: int(m.Id),
		Date:      int(m.Date),
		EditDate:  int(m.EditDate),
		Chat:      &ch,
	}
	out.AuthorSignature = m.AuthorSignature
	if fi := m.ForwardInfo; fi != nil {
		out.ForwardDate = int(fi.Date)
		switch o := fi.Origin.(type) {
		case *gotdbot.MessageOriginUser:
			if o != nil {
				u := b.user(o.SenderUserId)
				out.ForwardFrom = &u
			}
		case *gotdbot.MessageOriginHiddenUser:
			if o != nil {
				out.ForwardSenderName = o.SenderName
			}
		case *gotdbot.MessageOriginChat:
			if o != nil {
				c := b.chat(o.SenderChatId)
				out.ForwardFromChat = &c
				out.ForwardSignature = o.AuthorSignature
			}
		case *gotdbot.MessageOriginChannel:
			if o != nil {
				c := b.chat(o.ChatId)
				out.ForwardFromChat = &c
				out.ForwardFromMessageID = int(o.MessageId)
				out.ForwardSignature = o.AuthorSignature
			}
		}
	}
	switch s := m.SenderId.(type) {
	case *gotdbot.MessageSenderUser:
		if s != nil {
			u := b.user(s.UserId)
			out.From = &u
		}
	case *gotdbot.MessageSenderChat:
		if s != nil {
			c := b.chat(s.ChatId)
			out.SenderChat = &c
		}
	}

	if depth < 3 {
		if r, ok := m.ReplyTo.(*gotdbot.MessageReplyToMessage); ok && r != nil && r.MessageId != 0 {
			if rm := b.rawMessage(m.ChatId, r.MessageId); rm != nil {
				out.ReplyToMessage = b.convertMessageDepth(rm, depth+1)
			}
		}
	}

	b.fillContent(out, m.Content, depth)
	b.addSyntheticCommand(out)
	return out
}

func (b *BotAPI) fillContent(out *Message, content gotdbot.MessageContent, depth int) {
	switch c := content.(type) {
	case *gotdbot.MessageText:
		if c.Text != nil {
			out.Text = c.Text.Text
			out.Entities = b.convertEntities(c.Text.Entities)
		}
	case *gotdbot.MessagePhoto:
		out.Photo = convertPhoto(c.Photo)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageVideo:
		out.Video = convertVideo(c.Video)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageDocument:
		out.Document = convertDocument(c.Document)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageAudio:
		out.Audio = convertAudio(c.Audio)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageAnimation:
		out.Animation = convertAnimation(c.Animation)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageVoiceNote:
		out.Voice = convertVoice(c.VoiceNote)
		out.Caption, out.CaptionEntities = b.caption(c.Caption)
	case *gotdbot.MessageVideoNote:
		out.VideoNote = convertVideoNote(c.VideoNote)
	case *gotdbot.MessageSticker:
		out.Sticker = b.convertSticker(c.Sticker)
	case *gotdbot.MessageContact:
		if c.Contact != nil {
			out.Contact = &Contact{
				PhoneNumber: c.Contact.PhoneNumber,
				FirstName:   c.Contact.FirstName,
				LastName:    c.Contact.LastName,
				UserID:      c.Contact.UserId,
			}
		}
	case *gotdbot.MessageLocation:
		if c.Location != nil {
			out.Location = &Location{Longitude: c.Location.Longitude, Latitude: c.Location.Latitude}
		}
	case *gotdbot.MessageVenue:
		if c.Venue != nil {
			out.Venue = convertVenue(c.Venue)
		}
	case *gotdbot.MessagePoll:
		if c.Poll != nil {
			out.Poll = convertPoll(c.Poll)
		}
	case *gotdbot.MessageDice:
		out.Dice = &Dice{Emoji: c.Emoji, Value: int(c.Value)}
	case *gotdbot.MessageGame:
		if c.Game != nil {
			out.Game = &Game{
				Title:       c.Game.Title,
				Description: c.Game.Description,
				Photo:       convertPhoto(c.Game.Photo),
			}
			if c.Game.Text != nil {
				out.Game.Text = c.Game.Text.Text
			}
		}
	case *gotdbot.MessageChatAddMembers:
		for _, id := range c.MemberUserIds {
			out.NewChatMembers = append(out.NewChatMembers, b.user(id))
		}
	case *gotdbot.MessageChatJoinByLink:
		if out.From != nil {
			out.NewChatMembers = append(out.NewChatMembers, *out.From)
		}
	case *gotdbot.MessageChatJoinByRequest:
		if out.From != nil {
			out.NewChatMembers = append(out.NewChatMembers, *out.From)
		}
	case *gotdbot.MessageChatDeleteMember:
		u := b.user(c.UserId)
		out.LeftChatMember = &u
	case *gotdbot.MessagePinMessage:
		if c.MessageId != 0 && depth < 3 {
			if pm := b.rawMessage(out.Chat.ID, c.MessageId); pm != nil {
				out.PinnedMessage = b.convertMessageDepth(pm, depth+1)
			}
		}
	case *gotdbot.MessageChatChangeTitle:
		out.NewChatTitle = c.Title
	case *gotdbot.MessageChatChangePhoto:
		if c.Photo != nil {
			out.NewChatPhoto = convertPhotoSizes(c.Photo.Sizes)
		}
	}
}

func (b *BotAPI) caption(ft *gotdbot.FormattedText) (string, []MessageEntity) {
	if ft == nil {
		return "", nil
	}
	return ft.Text, b.convertEntities(ft.Entities)
}

func convertPhoto(p *gotdbot.Photo) []PhotoSize {
	if p == nil {
		return nil
	}
	return convertPhotoSizes(p.Sizes)
}

func convertPhotoSizes(sizes []gotdbot.PhotoSize) []PhotoSize {
	out := make([]PhotoSize, 0, len(sizes))
	for _, s := range sizes {
		out = append(out, PhotoSize{
			FileID:       fileIDOf(s.Photo),
			FileUniqueID: fileUIDOf(s.Photo),
			Width:        int(s.Width),
			Height:       int(s.Height),
			FileSize:     fileSizeOf(s.Photo),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Width < out[j].Width })
	return out
}

func convertThumb(t *gotdbot.Thumbnail) *PhotoSize {
	if t == nil || t.File == nil {
		return nil
	}
	return &PhotoSize{
		FileID:       fileIDOf(t.File),
		FileUniqueID: fileUIDOf(t.File),
		Width:        int(t.Width),
		Height:       int(t.Height),
		FileSize:     fileSizeOf(t.File),
	}
}

func convertDocument(d *gotdbot.Document) *Document {
	if d == nil {
		return nil
	}
	return &Document{
		FileID:       fileIDOf(d.Document),
		FileUniqueID: fileUIDOf(d.Document),
		FileName:     d.FileName,
		MimeType:     d.MimeType,
		FileSize:     fileSizeOf(d.Document),
		Thumbnail:    convertThumb(d.Thumbnail),
	}
}

func convertVideo(v *gotdbot.Video) *Video {
	if v == nil {
		return nil
	}
	return &Video{
		FileID:       fileIDOf(v.Video),
		FileUniqueID: fileUIDOf(v.Video),
		Width:        int(v.Width),
		Height:       int(v.Height),
		Duration:     int(v.Duration),
		MimeType:     v.MimeType,
		FileName:     v.FileName,
		FileSize:     fileSizeOf(v.Video),
		Thumbnail:    convertThumb(v.Thumbnail),
	}
}

func convertAudio(a *gotdbot.Audio) *Audio {
	if a == nil {
		return nil
	}
	return &Audio{
		FileID:       fileIDOf(a.Audio),
		FileUniqueID: fileUIDOf(a.Audio),
		Duration:     int(a.Duration),
		Performer:    a.Performer,
		Title:        a.Title,
		MimeType:     a.MimeType,
		FileName:     a.FileName,
		FileSize:     fileSizeOf(a.Audio),
		Thumbnail:    convertThumb(a.AlbumCoverThumbnail),
	}
}

func convertAnimation(a *gotdbot.Animation) *Animation {
	if a == nil {
		return nil
	}
	return &Animation{
		FileID:       fileIDOf(a.Animation),
		FileUniqueID: fileUIDOf(a.Animation),
		Width:        int(a.Width),
		Height:       int(a.Height),
		Duration:     int(a.Duration),
		MimeType:     a.MimeType,
		FileName:     a.FileName,
		FileSize:     fileSizeOf(a.Animation),
		Thumbnail:    convertThumb(a.Thumbnail),
	}
}

func convertVoice(v *gotdbot.VoiceNote) *Voice {
	if v == nil {
		return nil
	}
	return &Voice{
		FileID:       fileIDOf(v.Voice),
		FileUniqueID: fileUIDOf(v.Voice),
		Duration:     int(v.Duration),
		MimeType:     v.MimeType,
		FileSize:     fileSizeOf(v.Voice),
	}
}

func convertVideoNote(v *gotdbot.VideoNote) *VideoNote {
	if v == nil {
		return nil
	}
	return &VideoNote{
		FileID:       fileIDOf(v.Video),
		FileUniqueID: fileUIDOf(v.Video),
		Length:       int(v.Length),
		Duration:     int(v.Duration),
		FileSize:     fileSizeOf(v.Video),
		Thumbnail:    convertThumb(v.Thumbnail),
	}
}

func (b *BotAPI) convertSticker(s *gotdbot.Sticker) *Sticker {
	if s == nil {
		return nil
	}
	out := &Sticker{
		FileID:       fileIDOf(s.Sticker),
		FileUniqueID: fileUIDOf(s.Sticker),
		Width:        int(s.Width),
		Height:       int(s.Height),
		Emoji:        s.Emoji,
		FileSize:     fileSizeOf(s.Sticker),
		Thumbnail:    convertThumb(s.Thumbnail),
	}
	switch s.Format.(type) {
	case *gotdbot.StickerFormatTgs, *gotdbot.StickerFormatWebm:
		out.IsAnimated = true
	}
	if s.SetId != 0 {
		out.SetName = b.stickerSetName(s.SetId)
	}
	return out
}

func (b *BotAPI) stickerSetName(setID int64) string {
	now := time.Now()
	b.mu.Lock()
	if c, ok := b.stickerSets[setID]; ok && now.Before(c.exp) {
		b.mu.Unlock()
		return c.name
	}
	b.mu.Unlock()

	set, err := b.client.GetStickerSet(setID)
	name := ""
	if err == nil && set != nil {
		name = set.Name
	}
	b.mu.Lock()
	b.stickerSets[setID] = cachedStickerSet{name: name, exp: now.Add(cacheTTL)}
	b.mu.Unlock()
	return name
}

func convertVenue(v *gotdbot.Venue) *Venue {
	if v == nil {
		return nil
	}
	out := &Venue{
		Title:   v.Title,
		Address: v.Address,
	}
	if v.Location != nil {
		out.Location = Location{Longitude: v.Location.Longitude, Latitude: v.Location.Latitude}
	}
	if v.Provider == "foursquare" {
		out.FoursquareID = v.Id
		out.FoursquareType = v.Type
	} else {
		out.GooglePlaceID = v.Id
		out.GooglePlaceType = v.Type
	}
	return out
}

func convertPoll(p *gotdbot.Poll) *Poll {
	if p == nil {
		return nil
	}
	out := &Poll{
		ID:                    strconv.FormatInt(p.Id, 10),
		TotalVoterCount:       int(p.TotalVoterCount),
		IsClosed:              p.IsClosed,
		IsAnonymous:           p.IsAnonymous,
		AllowsMultipleAnswers: p.AllowsMultipleAnswers,
	}
	if p.Question != nil {
		out.Question = p.Question.Text
	}
	switch p.Type.(type) {
	case *gotdbot.PollTypeQuiz:
		out.Type = "quiz"
	default:
		out.Type = "regular"
	}
	for _, o := range p.Options {
		text := ""
		if o.Text != nil {
			text = o.Text.Text
		}
		out.Options = append(out.Options, PollOption{Text: text, VoterCount: int(o.VoterCount)})
	}
	return out
}

func (b *BotAPI) convertEntities(entities []gotdbot.TextEntity) []MessageEntity {
	if len(entities) == 0 {
		return nil
	}
	out := make([]MessageEntity, 0, len(entities))
	for _, e := range entities {
		me := MessageEntity{Offset: int(e.Offset), Length: int(e.Length)}
		switch t := e.Type.(type) {
		case *gotdbot.TextEntityTypeBold:
			me.Type = "bold"
		case *gotdbot.TextEntityTypeItalic:
			me.Type = "italic"
		case *gotdbot.TextEntityTypeUnderline:
			me.Type = "underline"
		case *gotdbot.TextEntityTypeStrikethrough:
			me.Type = "strikethrough"
		case *gotdbot.TextEntityTypeCode:
			me.Type = "code"
		case *gotdbot.TextEntityTypePre:
			me.Type = "pre"
		case *gotdbot.TextEntityTypePreCode:
			me.Type = "pre"
			me.Language = t.Language
		case *gotdbot.TextEntityTypeTextUrl:
			me.Type = "text_link"
			me.URL = t.Url
		case *gotdbot.TextEntityTypeUrl:
			me.Type = "url"
		case *gotdbot.TextEntityTypeEmailAddress:
			me.Type = "email"
		case *gotdbot.TextEntityTypePhoneNumber:
			me.Type = "phone_number"
		case *gotdbot.TextEntityTypeMention:
			me.Type = "mention"
		case *gotdbot.TextEntityTypeHashtag:
			me.Type = "hashtag"
		case *gotdbot.TextEntityTypeCashtag:
			me.Type = "cashtag"
		case *gotdbot.TextEntityTypeBotCommand:
			me.Type = "bot_command"
		case *gotdbot.TextEntityTypeMentionName:
			me.Type = "text_mention"
			u := b.user(t.UserId)
			me.User = &u
		case *gotdbot.TextEntityTypeBlockQuote:
			me.Type = "blockquote"
		case *gotdbot.TextEntityTypeExpandableBlockQuote:
			me.Type = "expandable_blockquote"
		case *gotdbot.TextEntityTypeSpoiler:
			me.Type = "spoiler"
		case *gotdbot.TextEntityTypeCustomEmoji:
			me.Type = "custom_emoji"
		case *gotdbot.TextEntityTypeBankCardNumber:
			me.Type = "bank_card"
		default:
			continue
		}
		out = append(out, me)
	}
	return out
}

func (b *BotAPI) convertChatMember(cm *gotdbot.ChatMember) ChatMember {
	if cm == nil {
		return ChatMember{}
	}
	out := ChatMember{}
	if u, ok := cm.MemberId.(*gotdbot.MessageSenderUser); ok && u != nil {
		user := b.user(u.UserId)
		out.User = &user
	}
	switch s := cm.Status.(type) {
	case *gotdbot.ChatMemberStatusCreator:
		out.Status = "creator"
		if s != nil && !s.IsMember {
			out.Status = "left"
		}
	case *gotdbot.ChatMemberStatusAdministrator:
		out.Status = "administrator"
		if s != nil && s.Rights != nil {
			out.CanBeEdited = s.CanBeEdited
			out.CanManageChat = s.Rights.CanManageChat
			out.CanChangeInfo = s.Rights.CanChangeInfo
			out.CanDeleteMessages = s.Rights.CanDeleteMessages
			out.CanInviteUsers = s.Rights.CanInviteUsers
			out.CanRestrictMembers = s.Rights.CanRestrictMembers
			out.CanPinMessages = s.Rights.CanPinMessages
			out.CanPromoteMembers = s.Rights.CanPromoteMembers
			out.CanManageVoiceChats = s.Rights.CanManageVideoChats
			out.CanPostMessages = s.Rights.CanPostMessages
			out.CanEditMessages = s.Rights.CanEditMessages
			out.CustomTitle = cm.Tag
		}
	case *gotdbot.ChatMemberStatusBanned:
		out.Status = "kicked"
		if s != nil {
			out.UntilDate = int64(s.BannedUntilDate)
		}
	case *gotdbot.ChatMemberStatusRestricted:
		out.Status = "restricted"
		if s != nil {
			out.UntilDate = int64(s.RestrictedUntilDate)
		}
	case *gotdbot.ChatMemberStatusLeft:
		out.Status = "left"
	case *gotdbot.ChatMemberStatusMember:
		out.Status = "member"
	}
	return out
}

func convertPermissionsToTD(p *ChatPermissions) *gotdbot.ChatPermissions {
	if p == nil {
		return &gotdbot.ChatPermissions{}
	}
	return &gotdbot.ChatPermissions{
		CanSendBasicMessages: p.CanSendMessages,
		CanSendPhotos:        p.CanSendMediaMessages,
		CanSendVideos:        p.CanSendMediaMessages,
		CanSendDocuments:     p.CanSendMediaMessages,
		CanSendAudios:        p.CanSendMediaMessages,
		CanSendVoiceNotes:    p.CanSendMediaMessages,
		CanSendVideoNotes:    p.CanSendMediaMessages,
		CanSendPolls:         p.CanSendPolls,
		CanSendOtherMessages: p.CanSendOtherMessages,
		CanAddLinkPreviews:   p.CanAddWebPagePreviews,
		CanChangeInfo:        p.CanChangeInfo,
		CanInviteUsers:       p.CanInviteUsers,
		CanPinMessages:       p.CanPinMessages,
	}
}
