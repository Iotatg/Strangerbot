package tgbotapi

import ()

// NewMessage creates a new Message.
//
// chatID is where to send it, text is the message text.
func NewMessage(chatID int64, text string) MessageConfig {
	return MessageConfig{
		BaseChat: BaseChat{
			ChatID:           chatID,
			ReplyToMessageID: 0,
		},
		Text:                  text,
		DisableWebPagePreview: false,
	}
}

// NewDeleteMessage creates a request to delete a message.

// NewMessageToChannel creates a new Message that is sent to a channel
// by username.
//
// username is the username of the channel, text is the message text,
// and the username should be in the form of `@username`.

// NewForward creates a new forward.
//
// chatID is where to send it, fromChatID is the source chat,
// and messageID is the ID of the original message.
func NewForward(chatID int64, fromChatID int64, messageID int) ForwardConfig {
	return ForwardConfig{
		BaseChat:   BaseChat{ChatID: chatID},
		FromChatID: fromChatID,
		MessageID:  messageID,
	}
}

// NewCopyMessage creates a new copy message.
//
// chatID is where to send it, fromChatID is the source chat,
// and messageID is the ID of the original message.

// NewPhoto creates a new sendPhoto request.
//
// chatID is where to send it, file is a string path to the file,
// FileReader, or FileBytes.
//
// Note that you must send animated GIFs as a document.
func NewPhoto(chatID int64, file RequestFileData) PhotoConfig {
	return PhotoConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
	}
}

// NewPhotoToChannel creates a new photo uploader to send a photo to a channel.
//
// Note that you must send animated GIFs as a document.

// NewAudio creates a new sendAudio request.
func NewAudio(chatID int64, file RequestFileData) AudioConfig {
	return AudioConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
	}
}

// NewDocument creates a new sendDocument request.
func NewDocument(chatID int64, file RequestFileData) DocumentConfig {
	return DocumentConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
	}
}

// NewSticker creates a new sendSticker request.
func NewSticker(chatID int64, file RequestFileData) StickerConfig {
	return StickerConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
	}
}

// NewVideo creates a new sendVideo request.

// NewVideo creates a new sendVideo request.
//
// chatID is where to send it, file is a string path to the file,
// FileReader, or FileBytes.
func NewVideo(chatID int64, file RequestFileData) VideoConfig {
	return VideoConfig{BaseFile: BaseFile{BaseChat: BaseChat{ChatID: chatID}, File: file}}
}

// NewAnimation creates a new sendAnimation request.
func NewAnimation(chatID int64, file RequestFileData) AnimationConfig {
	return AnimationConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{ChatID: chatID},
			File:     file,
		},
	}
}

// NewVideoNote creates a new sendVideoNote request.
//
// chatID is where to send it, file is a string path to the file,
// FileReader, or FileBytes.
func NewVideoNote(chatID int64, file RequestFileData) VideoNoteConfig {
	return VideoNoteConfig{BaseFile: BaseFile{BaseChat: BaseChat{ChatID: chatID}, File: file}}
}

// NewVoice creates a new sendVoice request.
//
// chatID is where to send it, file is a string path to the file,
// FileReader, or FileBytes.
func NewVoice(chatID int64, file RequestFileData) VoiceConfig {
	return VoiceConfig{BaseFile: BaseFile{BaseChat: BaseChat{ChatID: chatID}, File: file}}
}

// NewMediaGroup creates a new media group. Files should be an array of
// two to ten InputMediaPhoto or InputMediaVideo.

// NewInputMediaPhoto creates a new InputMediaPhoto.

// NewInputMediaVideo creates a new InputMediaVideo.

// NewInputMediaAnimation creates a new InputMediaAnimation.

// NewInputMediaAudio creates a new InputMediaAudio.

// NewInputMediaDocument creates a new InputMediaDocument.

// NewContact allows you to send a shared contact.

// NewLocation shares your location.
//
// chatID is where to send it, latitude and longitude are coordinates.

// NewVenue allows you to send a venue and its location.

// NewChatAction sets a chat action.
// Actions last for 5 seconds, or until your next action.
//
// chatID is where to send it, action should be set via Chat constants.

// NewUserProfilePhotos gets user profile photos.
//
// userID is the ID of the user you wish to get profile photos from.

// NewUpdate gets updates since the last Offset.
//
// offset is the last Update ID to include.
// You likely want to set this to the last Update ID plus 1.

// NewWebhook creates a new webhook.
//
// link is the url parsable link you wish to get the updates.

// NewWebhookWithCert creates a new webhook with a certificate.
//
// link is the url you wish to get webhooks,
// file contains a string to a file, FileReader, or FileBytes.

// NewInlineQueryResultArticle creates a new inline query article.

// NewInlineQueryResultArticleMarkdown creates a new inline query article with Markdown parsing.

// NewInlineQueryResultArticleMarkdownV2 creates a new inline query article with MarkdownV2 parsing.

// NewInlineQueryResultArticleHTML creates a new inline query article with HTML parsing.

// NewInlineQueryResultGIF creates a new inline query GIF.

// NewInlineQueryResultCachedGIF create a new inline query with cached photo.

// NewInlineQueryResultMPEG4GIF creates a new inline query MPEG4 GIF.

// NewInlineQueryResultCachedMPEG4GIF create a new inline query with cached MPEG4 GIF.

// NewInlineQueryResultPhoto creates a new inline query photo.

// NewInlineQueryResultPhotoWithThumb creates a new inline query photo.

// NewInlineQueryResultCachedPhoto create a new inline query with cached photo.

// NewInlineQueryResultVideo creates a new inline query video.

// NewInlineQueryResultCachedVideo create a new inline query with cached video.

// NewInlineQueryResultCachedSticker create a new inline query with cached sticker.

// NewInlineQueryResultAudio creates a new inline query audio.

// NewInlineQueryResultCachedAudio create a new inline query with cached photo.

// NewInlineQueryResultVoice creates a new inline query voice.

// NewInlineQueryResultCachedVoice create a new inline query with cached photo.

// NewInlineQueryResultDocument creates a new inline query document.

// NewInlineQueryResultCachedDocument create a new inline query with cached photo.

// NewInlineQueryResultLocation creates a new inline query location.

// NewInlineQueryResultVenue creates a new inline query venue.

// NewEditMessageText allows you to edit the text of a message.
func NewEditMessageText(chatID int64, messageID int, text string) EditMessageTextConfig {
	return EditMessageTextConfig{
		BaseEdit: BaseEdit{
			ChatID:    chatID,
			MessageID: messageID,
		},
		Text: text,
	}
}

// NewEditMessageTextAndMarkup allows you to edit the text and replymarkup of a message.
func NewEditMessageTextAndMarkup(chatID int64, messageID int, text string, replyMarkup InlineKeyboardMarkup) EditMessageTextConfig {
	return EditMessageTextConfig{
		BaseEdit: BaseEdit{
			ChatID:      chatID,
			MessageID:   messageID,
			ReplyMarkup: &replyMarkup,
		},
		Text: text,
	}
}

// NewEditMessageCaption allows you to edit the caption of a message.

// NewEditMessageReplyMarkup allows you to edit the inline
// keyboard markup.

// NewRemoveKeyboard hides the keyboard, with the option for being selective
// or hiding for everyone.

// NewKeyboardButton creates a regular keyboard button.

// NewKeyboardButtonContact creates a keyboard button that requests
// user contact information upon click.

// NewKeyboardButtonLocation creates a keyboard button that requests
// user location information upon click.

// NewKeyboardButtonRow creates a row of keyboard buttons.

// NewReplyKeyboard creates a new regular keyboard with sane defaults.

// NewOneTimeReplyKeyboard creates a new one time keyboard.

// NewInlineKeyboardButtonData creates an inline keyboard button with text
// and data for a callback.
func NewInlineKeyboardButtonData(text, data string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:         text,
		CallbackData: &data,
	}
}

// NewInlineKeyboardButtonLoginURL creates an inline keyboard button with text
// which goes to a LoginURL.

// NewInlineKeyboardButtonURL creates an inline keyboard button with text
// which goes to a URL.
func NewInlineKeyboardButtonURL(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text: text,
		URL:  &url,
	}
}

// NewInlineKeyboardButtonSwitch creates an inline keyboard button with
// text which allows the user to switch to a chat or return to a chat.

// NewInlineKeyboardRow creates an inline keyboard row with buttons.
func NewInlineKeyboardRow(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	var row []InlineKeyboardButton

	row = append(row, buttons...)

	return row
}

// NewInlineKeyboardMarkup creates a new inline keyboard.
func NewInlineKeyboardMarkup(rows ...[]InlineKeyboardButton) InlineKeyboardMarkup {
	var keyboard [][]InlineKeyboardButton

	keyboard = append(keyboard, rows...)

	return InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}
}

// NewCallback creates a new callback message.
func NewCallback(id, text string) CallbackConfig {
	return CallbackConfig{
		CallbackQueryID: id,
		Text:            text,
		ShowAlert:       false,
	}
}

// NewCallbackWithAlert creates a new callback message that alerts
// the user.

// NewInvoice creates a new Invoice request to the user.

// NewChatTitle allows you to update the title of a chat.
func NewChatTitle(chatID int64, title string) SetChatTitleConfig {
	return SetChatTitleConfig{
		ChatID: chatID,
		Title:  title,
	}
}

// NewChatDescription allows you to update the description of a chat.

// NewChatPhoto allows you to update the photo for a chat.
func NewChatPhoto(chatID int64, photo RequestFileData) SetChatPhotoConfig {
	return SetChatPhotoConfig{
		BaseFile: BaseFile{
			BaseChat: BaseChat{
				ChatID: chatID,
			},
			File: photo,
		},
	}
}

// NewDeleteChatPhoto allows you to delete the photo for a chat.

// NewPoll allows you to create a new poll.

// NewStopPoll allows you to stop a poll.

// NewDice allows you to send a random dice roll.

// NewDiceWithEmoji allows you to send a random roll of one of many types.
//
// Emoji may be 🎲 (1-6), 🎯 (1-6), or 🏀 (1-5).

// NewBotCommandScopeDefault represents the default scope of bot commands.

// NewBotCommandScopeAllPrivateChats represents the scope of bot commands,
// covering all private chats.

// NewBotCommandScopeAllGroupChats represents the scope of bot commands,
// covering all group and supergroup chats.

// NewBotCommandScopeAllChatAdministrators represents the scope of bot commands,
// covering all group and supergroup chat administrators.

// NewBotCommandScopeChat represents the scope of bot commands, covering a
// specific chat.

// NewBotCommandScopeChatAdministrators represents the scope of bot commands,
// covering all administrators of a specific group or supergroup chat.

// NewBotCommandScopeChatMember represents the scope of bot commands, covering a
// specific member of a group or supergroup chat.

// NewGetMyCommandsWithScope allows you to set the registered commands for a
// given scope.

// NewGetMyCommandsWithScopeAndLanguage allows you to set the registered
// commands for a given scope and language code.

// NewSetMyCommands allows you to set the registered commands.
func NewSetMyCommands(commands ...BotCommand) SetMyCommandsConfig {
	return SetMyCommandsConfig{Commands: commands}
}

// NewSetMyCommandsWithScope allows you to set the registered commands for a given scope.

// NewSetMyCommandsWithScopeAndLanguage allows you to set the registered commands for a given scope
// and language code.

// NewDeleteMyCommands allows you to delete the registered commands.

// NewDeleteMyCommandsWithScope allows you to delete the registered commands for a given
// scope.

// NewDeleteMyCommandsWithScopeAndLanguage allows you to delete the registered commands for a given
// scope and language code.
