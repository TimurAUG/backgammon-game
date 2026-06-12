package game

import (
	"strings"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
)

const (
	// maxChatLen — предел длины одного сообщения в СИМВОЛАХ (рунах). Длиннее —
	// усекается. Защита от мусорных «простыней» в истории/Storage.
	maxChatLen = 500
	// maxChatHistory — сколько последних сообщений держим в памяти/Storage.
	// Старые вытесняются: чат партии не растёт безгранично.
	maxChatHistory = 200
)

// ChatMessage — одно сообщение чата партии. Sender — цвет автора; в протокол
// он конвертируется в строку "white"/"black" при рассылке/сериализации.
type ChatMessage struct {
	Sender domain.Color
	Text   string
}

// Chat возвращает копию истории чата. Для инспекции (тесты, persistence).
func (g *Game) Chat() []ChatMessage {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]ChatMessage, len(g.chat))
	copy(out, g.chat)
	return out
}

// PostChat добавляет сообщение игрока c в историю чата. Текст обрезается по
// краям; пустой (после trim) — игнорируется без ошибки; длина усекается до
// maxChatLen рун. История кэпится последними maxChatHistory сообщениями.
//
// Отправителя задаёт вызывающий по цвету слота — клиент его не передаёт.
func (g *Game) PostChat(c domain.Color, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if r := []rune(text); len(r) > maxChatLen {
		text = string(r[:maxChatLen])
	}

	g.mu.Lock()
	defer g.maybePersist()
	defer g.mu.Unlock()
	m := ChatMessage{Sender: c, Text: text}
	g.chat = append(g.chat, m)
	if len(g.chat) > maxChatHistory {
		g.chat = g.chat[len(g.chat)-maxChatHistory:]
	}
	g.broadcastChatLocked(m)
}

// broadcastChatLocked рассылает сообщение чата обоим подключённым клиентам.
// Вызывается с захваченным mu (как остальные broadcast*Locked).
func (g *Game) broadcastChatLocked(m ChatMessage) {
	msg := protocol.ServerMessage{
		Type:   "CHAT",
		Sender: colorString(m.Sender),
		Text:   m.Text,
	}
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

// ChatHistoryMessage собирает CHAT_HISTORY со всей историей чата для досылки
// при (ре)коннекте. Возвращает nil при пустой истории (досылать нечего) — по
// образцу LegalMovesMessageFor.
func (g *Game) ChatHistoryMessage() *protocol.ServerMessage {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.chat) == 0 {
		return nil
	}
	payload := make([]protocol.ChatPayload, len(g.chat))
	for i, m := range g.chat {
		payload[i] = protocol.ChatPayload{Sender: colorString(m.Sender), Text: m.Text}
	}
	return &protocol.ServerMessage{Type: "CHAT_HISTORY", Chat: payload}
}
