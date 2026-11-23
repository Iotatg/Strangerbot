package tgbotapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Iotatg/gotdbot"
)

func replyTo(id int) gotdbot.InputMessageReplyTo {
	if id == 0 {
		return nil
	}
	return &gotdbot.InputMessageReplyToMessage{MessageId: int64(id)}
}

func (b *BotAPI) formatted(text, parseMode string) *gotdbot.FormattedText {
	if text == "" {
		return nil
	}
	if strings.TrimSpace(parseMode) == "" {
		return &gotdbot.FormattedText{Text: text}
	}
	ft, err := b.client.Format(text, parseMode)
	if err != nil || ft == nil {
		return &gotdbot.FormattedText{Text: text}
	}
	return ft
}

type markupButton struct {
	Text         string `json:"text"`
	URL          string `json:"url"`
	CallbackData string `json:"callback_data"`
	Style        string `json:"style"`
}

type markupJSON struct {
	InlineKeyboard [][]markupButton `json:"inline_keyboard"`
}

func (b *BotAPI) markup(m interface{}) gotdbot.ReplyMarkup {
	if m == nil {
		return nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	var parsed markupJSON
	if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.InlineKeyboard) == 0 {
		return nil
	}
	rows := make([][]gotdbot.InlineKeyboardButton, 0, len(parsed.InlineKeyboard))
	for _, row := range parsed.InlineKeyboard {
		btns := make([]gotdbot.InlineKeyboardButton, 0, len(row))
		for _, btn := range row {
			ib := gotdbot.InlineKeyboardButton{Text: btn.Text}
			switch {
			case btn.URL != "":
				ib.Type = &gotdbot.InlineKeyboardButtonTypeUrl{Url: btn.URL}
			case btn.CallbackData != "":
				ib.Type = &gotdbot.InlineKeyboardButtonTypeCallback{Data: []byte(btn.CallbackData)}
			default:
				continue
			}
			switch strings.ToLower(btn.Style) {
			case "danger":
				ib.Style = &gotdbot.ButtonStyleDanger{}
			case "success":
				ib.Style = &gotdbot.ButtonStyleSuccess{}
			case "primary":
				ib.Style = &gotdbot.ButtonStylePrimary{}
			}
			btns = append(btns, ib)
		}
		if len(btns) > 0 {
			rows = append(rows, btns)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return &gotdbot.ReplyMarkupInlineKeyboard{Rows: rows}
}

func (b *BotAPI) inputFile(f RequestFileData) (gotdbot.InputFile, error) {
	switch v := f.(type) {
	case FileID:
		id, err := strconv.ParseInt(string(v), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid file id %q", string(v))
		}
		return gotdbot.FileFromID(int32(id)), nil
	case FilePath:
		return gotdbot.FileFromPath(string(v)), nil
	case FileURL:
		url := string(v)
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			path, err := downloadURL(url)
			if err != nil {
				return nil, err
			}
			return gotdbot.FileFromPath(path), nil
		}
		return gotdbot.FileFromPath(url), nil
	case FileBytes:
		path, err := writeTemp(v.Name, v.Bytes)
		if err != nil {
			return nil, err
		}
		return gotdbot.FileFromPath(path), nil
	case FileReader:
		data, err := io.ReadAll(v.Reader)
		if err != nil {
			return nil, err
		}
		path, err := writeTemp(v.Name, data)
		if err != nil {
			return nil, err
		}
		return gotdbot.FileFromPath(path), nil
	}
	return nil, fmt.Errorf("unsupported file type %T", f)
}

func writeTemp(name string, data []byte) (string, error) {
	dir, err := os.MkdirTemp("", "meowrobot-upload-*")
	if err != nil {
		return "", err
	}
	if name == "" {
		name = "file"
	}
	path := filepath.Join(dir, filepath.Base(name))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

var uploadHTTPClient = &http.Client{Timeout: 2 * time.Minute}

func downloadURL(url string) (string, error) {
	resp, err := uploadHTTPClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	name := filepath.Base(strings.Split(url, "?")[0])
	if name == "" || name == "." || name == "/" {
		name = "download"
	}
	return writeTemp(name, data)
}

func (b *BotAPI) sendText(cfg MessageConfig) (Message, error) {
	opts := &gotdbot.SendTextMessageOpts{
		ParseMode:             cfg.ParseMode,
		DisableWebPagePreview: cfg.DisableWebPagePreview,
		ReplyTo:               replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:           b.markup(cfg.ReplyMarkup),
		ProtectContent:        cfg.ProtectContent,
	}
	m, err := b.client.SendTextMessage(cfg.ChatID, cfg.Text, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendPhoto(cfg PhotoConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendPhotoOpts{
		Caption:        cfg.Caption,
		ParseMode:      cfg.ParseMode,
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendPhoto(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendDocument(cfg DocumentConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendDocumentOpts{
		Caption:                     cfg.Caption,
		ParseMode:                   cfg.ParseMode,
		DisableContentTypeDetection: cfg.DisableContentTypeDetection,
		ReplyTo:                     replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:                 b.markup(cfg.ReplyMarkup),
		ProtectContent:              cfg.ProtectContent,
	}
	m, err := b.client.SendDocument(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendAudio(cfg AudioConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendAudioOpts{
		Caption:        cfg.Caption,
		ParseMode:      cfg.ParseMode,
		Title:          cfg.Title,
		Performer:      cfg.Performer,
		Duration:       int32(cfg.Duration),
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendAudio(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendAnimation(cfg AnimationConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendAnimationOpts{
		Caption:        cfg.Caption,
		ParseMode:      cfg.ParseMode,
		Duration:       int32(cfg.Duration),
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendAnimation(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendSticker(cfg StickerConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendStickerOpts{
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendSticker(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendVideo(cfg VideoConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendVideoOpts{
		Caption:           cfg.Caption,
		ParseMode:         cfg.ParseMode,
		Duration:          int32(cfg.Duration),
		SupportsStreaming: cfg.SupportsStreaming,
		ReplyTo:           replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:       b.markup(cfg.ReplyMarkup),
		ProtectContent:    cfg.ProtectContent,
	}
	m, err := b.client.SendVideo(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendVoice(cfg VoiceConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendVoiceOpts{
		Caption:        cfg.Caption,
		ParseMode:      cfg.ParseMode,
		Duration:       int32(cfg.Duration),
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendVoice(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) sendVideoNote(cfg VideoNoteConfig) (Message, error) {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return Message{}, err
	}
	opts := &gotdbot.SendVideoNoteOpts{
		Duration:       int32(cfg.Duration),
		Length:         int32(cfg.Length),
		ReplyTo:        replyTo(cfg.ReplyToMessageID),
		ReplyMarkup:    b.markup(cfg.ReplyMarkup),
		ProtectContent: cfg.ProtectContent,
	}
	m, err := b.client.SendVideoNote(cfg.ChatID, f, opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) editText(cfg EditMessageTextConfig) (Message, error) {
	content := &gotdbot.InputMessageText{Text: b.formatted(cfg.Text, cfg.ParseMode)}
	opts := &gotdbot.EditMessageTextOpts{ReplyMarkup: b.markup(cfg.ReplyMarkup)}
	m, err := b.client.EditMessageText(cfg.ChatID, content, int64(cfg.MessageID), opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func (b *BotAPI) editCaption(cfg EditMessageCaptionConfig) (Message, error) {
	opts := &gotdbot.EditMessageCaptionOpts{
		Caption:     b.formatted(cfg.Caption, cfg.ParseMode),
		ReplyMarkup: b.markup(cfg.ReplyMarkup),
	}
	m, err := b.client.EditMessageCaption(cfg.ChatID, int64(cfg.MessageID), opts)
	if err != nil {
		return Message{}, err
	}
	return derefMessage(b.convertMessage(m)), nil
}

func derefMessage(m *Message) Message {
	if m == nil {
		return Message{}
	}
	return *m
}

// dispatch handles send/edit style configs. It returns sent=true when the
// config was executed here.
func (b *BotAPI) dispatch(c Chattable) (Message, bool, error) {
	switch cfg := c.(type) {
	case MessageConfig:
		m, err := b.sendText(cfg)
		return m, true, err
	case PhotoConfig:
		m, err := b.sendPhoto(cfg)
		return m, true, err
	case DocumentConfig:
		m, err := b.sendDocument(cfg)
		return m, true, err
	case AudioConfig:
		m, err := b.sendAudio(cfg)
		return m, true, err
	case AnimationConfig:
		m, err := b.sendAnimation(cfg)
		return m, true, err
	case StickerConfig:
		m, err := b.sendSticker(cfg)
		return m, true, err
	case VideoConfig:
		m, err := b.sendVideo(cfg)
		return m, true, err
	case VoiceConfig:
		m, err := b.sendVoice(cfg)
		return m, true, err
	case VideoNoteConfig:
		m, err := b.sendVideoNote(cfg)
		return m, true, err
	case EditMessageTextConfig:
		m, err := b.editText(cfg)
		return m, true, err
	case EditMessageCaptionConfig:
		m, err := b.editCaption(cfg)
		return m, true, err
	}
	return Message{}, false, nil
}

// Send executes a send-style config and returns the resulting message.
func (b *BotAPI) Send(c Chattable) (Message, error) {
	if m, sent, err := b.dispatch(c); sent {
		return m, err
	}
	if err := b.doRequest(c); err != nil {
		return Message{}, err
	}
	return Message{}, nil
}

// Request executes any config and wraps the result in an APIResponse.
func (b *BotAPI) Request(c Chattable) (*APIResponse, error) {
	if _, sent, err := b.dispatch(c); sent {
		if err != nil {
			return nil, err
		}
		return &APIResponse{Ok: true}, nil
	}
	if err := b.doRequest(c); err != nil {
		return nil, err
	}
	return &APIResponse{Ok: true}, nil
}

func (b *BotAPI) doRequest(c Chattable) error {
	switch cfg := c.(type) {
	case SetMyCommandsConfig:
		cmds := make([]gotdbot.BotCommand, 0, len(cfg.Commands))
		for _, cmd := range cfg.Commands {
			cmds = append(cmds, gotdbot.BotCommand{Command: cmd.Command, Description: cmd.Description})
		}
		return b.client.SetCommands(cmds, "", nil)
	case DeleteMessageConfig:
		return b.client.DeleteMessages(cfg.ChatID, []int64{int64(cfg.MessageID)}, nil)
	case BanChatMemberConfig:
		err := b.client.SetChatMemberStatus(cfg.ChatID, sender(cfg.UserID), &gotdbot.ChatMemberStatusBanned{
			BannedUntilDate: int32(cfg.UntilDate),
		})
		b.invalidateMember(cfg.ChatID, cfg.UserID)
		return err
	case UnbanChatMemberConfig:
		err := b.client.SetChatMemberStatus(cfg.ChatID, sender(cfg.UserID), &gotdbot.ChatMemberStatusLeft{})
		b.invalidateMember(cfg.ChatID, cfg.UserID)
		return err
	case RestrictChatMemberConfig:
		err := b.client.SetChatMemberStatus(cfg.ChatID, sender(cfg.UserID), &gotdbot.ChatMemberStatusRestricted{
			IsMember:            true,
			Permissions:         convertPermissionsToTD(cfg.Permissions),
			RestrictedUntilDate: int32(cfg.UntilDate),
		})
		b.invalidateMember(cfg.ChatID, cfg.UserID)
		return err
	case PromoteChatMemberConfig:
		err := b.promote(cfg)
		b.invalidateMember(cfg.ChatID, cfg.UserID)
		return err
	case SetChatPermissionsConfig:
		return b.client.SetChatPermissions(cfg.ChatID, convertPermissionsToTD(cfg.Permissions))
	case PinChatMessageConfig:
		return b.client.PinChatMessage(cfg.ChatID, int64(cfg.MessageID), &gotdbot.PinChatMessageOpts{
			DisableNotification: cfg.DisableNotification,
		})
	case UnpinChatMessageConfig:
		return b.client.UnpinChatMessage(cfg.ChatID, int64(cfg.MessageID))
	case SetChatTitleConfig:
		err := b.client.SetChatTitle(cfg.ChatID, cfg.Title)
		b.invalidateChat(cfg.ChatID)
		return err
	case SetChatDescriptionConfig:
		return b.client.SetChatDescription(cfg.ChatID, cfg.Description)
	case SetChatStickerSetConfig:
		return b.setChatStickerSet(cfg)
	case SetChatAdministratorCustomTitle:
		return b.client.SetChatMemberTag(cfg.ChatID, cfg.CustomTitle, cfg.UserID)
	case LeaveChatConfig:
		return b.client.LeaveChat(cfg.ChatID)
	case DeleteChatPhotoConfig:
		err := b.client.SetChatPhoto(cfg.ChatID, &gotdbot.SetChatPhotoOpts{})
		b.invalidateChat(cfg.ChatID)
		return err
	case SetChatPhotoConfig:
		return b.setChatPhoto(cfg)
	case ApproveChatJoinRequestConfig:
		return b.client.ApproveJoinRequest(cfg.ChatID, cfg.UserID)
	case DeclineChatJoinRequest:
		return b.client.DeclineJoinRequest(cfg.ChatID, cfg.UserID)
	case CallbackConfig:
		id, _ := strconv.ParseInt(cfg.CallbackQueryID, 10, 64)
		return b.client.AnswerCallbackQuery(0, id, cfg.Text, cfg.URL, &gotdbot.AnswerCallbackQueryOpts{
			ShowAlert: cfg.ShowAlert,
		})
	case ForwardConfig:
		_, err := b.client.ForwardMessages(cfg.ChatID, cfg.FromChatID, []int64{int64(cfg.MessageID)}, nil)
		return err
	}
	return fmt.Errorf("unsupported telegram method %T", c)
}

func sender(userID int64) gotdbot.MessageSender {
	return &gotdbot.MessageSenderUser{UserId: userID}
}

func (b *BotAPI) promote(cfg PromoteChatMemberConfig) error {
	any := cfg.CanManageChat || cfg.CanChangeInfo || cfg.CanPostMessages || cfg.CanEditMessages ||
		cfg.CanDeleteMessages || cfg.CanManageVoiceChats || cfg.CanInviteUsers ||
		cfg.CanRestrictMembers || cfg.CanPinMessages || cfg.CanPromoteMembers
	if !any {
		return b.client.SetChatMemberStatus(cfg.ChatID, sender(cfg.UserID), &gotdbot.ChatMemberStatusMember{})
	}
	rights := &gotdbot.ChatAdministratorRights{
		CanManageChat:       cfg.CanManageChat,
		CanChangeInfo:       cfg.CanChangeInfo,
		CanPostMessages:     cfg.CanPostMessages,
		CanEditMessages:     cfg.CanEditMessages,
		CanDeleteMessages:   cfg.CanDeleteMessages,
		CanManageVideoChats: cfg.CanManageVoiceChats,
		CanInviteUsers:      cfg.CanInviteUsers,
		CanRestrictMembers:  cfg.CanRestrictMembers,
		CanPinMessages:      cfg.CanPinMessages,
		CanPromoteMembers:   cfg.CanPromoteMembers,
		IsAnonymous:         cfg.IsAnonymous,
	}
	return b.client.SetChatMemberStatus(cfg.ChatID, sender(cfg.UserID), &gotdbot.ChatMemberStatusAdministrator{
		Rights: rights,
	})
}

func (b *BotAPI) setChatStickerSet(cfg SetChatStickerSetConfig) error {
	set, err := b.client.SearchStickerSet(cfg.StickerSetName, nil)
	if err != nil {
		return err
	}
	chat, err := b.client.GetChat(cfg.ChatID)
	if err != nil {
		return err
	}
	sg, ok := chat.Type.(*gotdbot.ChatTypeSupergroup)
	if !ok || sg == nil {
		return fmt.Errorf("sticker set can only be set for a supergroup")
	}
	return b.client.SetSupergroupStickerSet(set.Id, sg.SupergroupId)
}

func (b *BotAPI) setChatPhoto(cfg SetChatPhotoConfig) error {
	f, err := b.inputFile(cfg.File)
	if err != nil {
		return err
	}
	local, ok := f.(*gotdbot.InputFileLocal)
	if !ok {
		return fmt.Errorf("chat photo must be an uploaded image")
	}
	err = b.client.SetChatPhoto(cfg.ChatID, &gotdbot.SetChatPhotoOpts{
		Photo: &gotdbot.InputChatPhotoStatic{Photo: gotdbot.FileFromPath(local.Path)},
	})
	b.invalidateChat(cfg.ChatID)
	return err
}
