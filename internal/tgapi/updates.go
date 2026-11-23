package tgbotapi

import (
	"strconv"

	"github.com/Iotatg/gotdbot"
)

// ConvertMessage converts a TDLib message into the classic Bot API shape.
func (b *BotAPI) ConvertMessage(m *gotdbot.Message) *Message {
	return b.convertMessage(m)
}

// ConvertEditedMessage reconstructs an edited message update.
func (b *BotAPI) ConvertEditedMessage(u *gotdbot.UpdateMessageEdited) *Message {
	if u == nil {
		return nil
	}
	m, err := b.client.GetMessage(u.ChatId, u.MessageId)
	if err != nil || m == nil {
		return nil
	}
	return b.convertMessage(m)
}

// ConvertCallbackQuery converts a TDLib callback query.
func (b *BotAPI) ConvertCallbackQuery(u *gotdbot.UpdateNewCallbackQuery) *CallbackQuery {
	if u == nil {
		return nil
	}
	cq := &CallbackQuery{
		ID:           strconv.FormatInt(u.Id, 10),
		ChatInstance: strconv.FormatInt(u.ChatInstance, 10),
	}
	from := b.user(u.SenderUserId)
	cq.From = &from
	if payload, ok := u.Payload.(*gotdbot.CallbackQueryPayloadData); ok && payload != nil {
		cq.Data = string(payload.Data)
	}
	chat := b.chat(u.ChatId)
	cq.Message = &Message{MessageID: int(u.MessageId), Chat: &chat}
	if u.MessageId != 0 {
		if m, err := b.client.GetMessage(u.ChatId, u.MessageId); err == nil && m != nil {
			if full := b.convertMessage(m); full != nil {
				cq.Message = full
			}
		}
	}
	return cq
}

// ConvertChatMemberUpdate converts a TDLib chat member update.
func (b *BotAPI) ConvertChatMemberUpdate(u *gotdbot.UpdateChatMember) *ChatMemberUpdated {
	if u == nil {
		return nil
	}
	actor := b.user(u.ActorUserId)
	upd := &ChatMemberUpdated{
		Chat:          b.chat(u.ChatId),
		From:          actor,
		Date:          int(u.Date),
		OldChatMember: b.convertChatMember(u.OldChatMember),
		NewChatMember: b.convertChatMember(u.NewChatMember),
	}
	return upd
}

// ConvertJoinRequest converts a TDLib join request update.
func (b *BotAPI) ConvertJoinRequest(u *gotdbot.UpdateNewChatJoinRequest) *ChatJoinRequest {
	if u == nil || u.Request == nil {
		return nil
	}
	return &ChatJoinRequest{
		Chat: b.chat(u.ChatId),
		From: b.user(u.Request.UserId),
		Date: int(u.Request.Date),
		Bio:  u.Request.Bio,
	}
}
