package tgbotapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/Iotatg/gotdbot"
)

type fileServer struct {
	mu    sync.RWMutex
	files map[string]string
	addr  string
}

var (
	globalFiles    *fileServer
	globalFilesErr error
	globalFilesOne sync.Once
)

func startFileServer() (*fileServer, error) {
	globalFilesOne.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			globalFilesErr = err
			return
		}
		fs := &fileServer{files: make(map[string]string), addr: ln.Addr().String()}
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.URL.Path, "/")
			fs.mu.RLock()
			path := fs.files[token]
			fs.mu.RUnlock()
			if path == "" {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, path)
		})
		go func() {
			_ = (&http.Server{Handler: mux}).Serve(ln)
		}()
		globalFiles = fs
	})
	return globalFiles, globalFilesErr
}

func (fs *fileServer) add(path string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)
	fs.mu.Lock()
	fs.files[token] = path
	fs.mu.Unlock()
	return "http://" + fs.addr + "/" + token
}

func parseFileID(fileID string) (int32, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(fileID), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid file id %q", fileID)
	}
	return int32(id), nil
}

// GetFile returns metadata for a file identifier produced by this transport.
func (b *BotAPI) GetFile(config FileConfig) (File, error) {
	id, err := parseFileID(config.FileID)
	if err != nil {
		return File{}, err
	}
	f, err := b.client.GetFile(id)
	if err != nil {
		return File{}, err
	}
	path := config.FileID
	if f.Local != nil && f.Local.IsDownloadingCompleted && f.Local.Path != "" {
		path = f.Local.Path
	}
	return File{
		FileID:   config.FileID,
		FileSize: int(f.Size),
		FilePath: path,
	}, nil
}

// DeleteMessagesInRange deletes every message in the inclusive TDLib id range
// [fromID, toID] for the given chat, returning how many were removed. TDLib
// message ids are large and non-contiguous per chat, and bots may not call
// getChatHistory, so the ids are resolved from the per-chat cache of messages
// the bot has observed.
func (b *BotAPI) DeleteMessagesInRange(chatID int64, fromID, toID int) (int, error) {
	if fromID <= 0 || toID <= 0 || toID < fromID {
		return 0, nil
	}
	ids := b.cachedIDsInRange(chatID, int64(fromID), int64(toID))
	if len(ids) == 0 {
		return 0, nil
	}
	if err := b.client.DeleteMessages(chatID, ids, nil); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// GetFileDirectURL downloads the file and exposes it over a loopback HTTP
// server so callers that use http.Get keep working.
func (b *BotAPI) GetFileDirectURL(fileID string) (string, error) {
	var path string
	if id, err := parseFileID(fileID); err == nil {
		f, err := b.client.GetFile(id)
		if err != nil {
			return "", err
		}
		if f.Local == nil || !f.Local.IsDownloadingCompleted || f.Local.Path == "" {
			f, err = b.client.DownloadFile(id, 0, 0, 1, &gotdbot.DownloadFileOpts{Synchronous: true})
			if err != nil {
				return "", err
			}
		}
		if f.Local == nil || f.Local.Path == "" {
			return "", fmt.Errorf("file %s is not available locally", fileID)
		}
		path = f.Local.Path
	} else {
		path = fileID
	}
	fs, err := startFileServer()
	if err != nil {
		return "", err
	}
	return fs.add(path), nil
}

// GetChat resolves a chat or username to a Chat.
func (b *BotAPI) GetChat(config ChatInfoConfig) (Chat, error) {
	if config.SuperGroupUsername != "" {
		username := strings.TrimPrefix(config.SuperGroupUsername, "@")
		raw, err := b.client.ResolveUsername(username)
		if err != nil {
			return Chat{}, err
		}
		return b.chatFromRaw(raw), nil
	}
	raw, err := b.client.GetChat(config.ChatID)
	if err != nil {
		return Chat{}, err
	}
	return b.chatFromRaw(raw), nil
}

// GetChatMember returns a member of a chat.
func (b *BotAPI) GetChatMember(config GetChatMemberConfig) (ChatMember, error) {
	if m, ok := b.cachedMember(config.ChatID, config.UserID); ok {
		return m, nil
	}
	cm, err := b.client.GetChatMember(config.ChatID, sender(config.UserID))
	if err != nil {
		return ChatMember{}, err
	}
	out := b.convertChatMember(cm)
	b.cacheMember(config.ChatID, config.UserID, out)
	return out, nil
}

// GetChatAdministrators returns the administrators of a chat.
func (b *BotAPI) GetChatAdministrators(config ChatAdministratorsConfig) ([]ChatMember, error) {
	res, err := b.client.GetChatAdministrators(config.ChatID)
	if err != nil {
		return nil, err
	}
	out := make([]ChatMember, 0, len(res.Administrators))
	for _, admin := range res.Administrators {
		u := b.user(admin.UserId)
		status := "administrator"
		if admin.IsOwner {
			status = "creator"
		}
		out = append(out, ChatMember{
			User:        &u,
			Status:      status,
			CustomTitle: admin.CustomTitle,
			CanBeEdited: admin.CanBeEdited,
		})
	}
	return out, nil
}

// GetChatMembersCount returns the number of members in a chat.
func (b *BotAPI) GetChatMembersCount(config ChatMemberCountConfig) (int, error) {
	raw, err := b.client.GetChat(config.ChatID)
	if err != nil {
		return 0, err
	}
	switch t := raw.Type.(type) {
	case *gotdbot.ChatTypeSupergroup:
		info, err := b.client.GetSupergroupFullInfo(t.SupergroupId)
		if err != nil {
			return 0, err
		}
		return int(info.MemberCount), nil
	case *gotdbot.ChatTypeBasicGroup:
		info, err := b.client.GetBasicGroupFullInfo(t.BasicGroupId)
		if err != nil {
			return 0, err
		}
		return len(info.Members), nil
	case *gotdbot.ChatTypePrivate, *gotdbot.ChatTypeSecret:
		return 1, nil
	}
	return 0, fmt.Errorf("unsupported chat type")
}

// GetInviteLink creates and returns an invite link for a chat.
func (b *BotAPI) GetInviteLink(config ChatInviteLinkConfig) (string, error) {
	link, err := b.client.CreateChatInviteLink(config.ChatID, 0, 0, "", nil)
	if err != nil {
		return "", err
	}
	return link.InviteLink, nil
}

// MakeRequest supports the small set of raw endpoints used by this project.
func (b *BotAPI) MakeRequest(endpoint string, params Params) (*APIResponse, error) {
	switch endpoint {
	case "editMessageText":
		chatID, _ := strconv.ParseInt(params["chat_id"], 10, 64)
		messageID, _ := strconv.Atoi(params["message_id"])
		content := &gotdbot.InputMessageText{Text: b.formatted(params["text"], params["parse_mode"])}
		opts := &gotdbot.EditMessageTextOpts{}
		if raw := params["reply_markup"]; raw != "" {
			var mk markupJSON
			if err := json.Unmarshal([]byte(raw), &mk); err == nil {
				opts.ReplyMarkup = b.markup(mk)
			}
		}
		if _, err := b.client.EditMessageText(chatID, content, int64(messageID), opts); err != nil {
			return nil, err
		}
		return &APIResponse{Ok: true}, nil
	case "editMessageCaption":
		chatID, _ := strconv.ParseInt(params["chat_id"], 10, 64)
		messageID, _ := strconv.Atoi(params["message_id"])
		opts := &gotdbot.EditMessageCaptionOpts{
			Caption: b.formatted(params["caption"], params["parse_mode"]),
		}
		if raw := params["reply_markup"]; raw != "" {
			var mk markupJSON
			if err := json.Unmarshal([]byte(raw), &mk); err == nil {
				opts.ReplyMarkup = b.markup(mk)
			}
		}
		if _, err := b.client.EditMessageCaption(chatID, int64(messageID), opts); err != nil {
			return nil, err
		}
		return &APIResponse{Ok: true}, nil
	}
	return nil, fmt.Errorf("unsupported endpoint %q", endpoint)
}

// StopReceivingUpdates is a no-op kept for compatibility.
func (b *BotAPI) StopReceivingUpdates() {}

// IsMessageToMe reports whether the message mentions this bot.
func (b *BotAPI) IsMessageToMe(m Message) bool {
	if b.Self.UserName == "" {
		return false
	}
	return strings.Contains(m.Text, "@"+b.Self.UserName)
}
