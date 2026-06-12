package game_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/stretchr/testify/require"
)

// twoPlayerGame поднимает партию с двумя подключёнными mockConn и возвращает
// сами conn'ы и live-объект игры. Общий сетап для чат-тестов.
func twoPlayerGame(t *testing.T) (white, black *mockConn, g *game.Game) {
	t.Helper()
	mgr := game.NewManagerWithRand(bytes.NewReader([]byte{0, 0}))
	white = &mockConn{}
	black = &mockConn{}
	_, _, err := mgr.JoinGame("g1", "", white)
	require.NoError(t, err)
	_, g, err = mgr.JoinGame("g1", "", black)
	require.NoError(t, err)
	return white, black, g
}

// TestGame_PostChat_AppendsWithSenderAndText — сообщение игрока добавляется
// в историю с правильным отправителем (цветом) и текстом.
// TDD plan #45.
func TestGame_PostChat_AppendsWithSenderAndText(t *testing.T) {
	_, _, g := twoPlayerGame(t)

	g.PostChat(domain.Black, "привет")

	history := g.Chat()
	require.Len(t, history, 1)
	require.Equal(t, domain.Black, history[0].Sender)
	require.Equal(t, "привет", history[0].Text)
}

// TestGame_PostChat_RejectsBlank — пустое или состоящее только из пробелов
// сообщение игнорируется (в историю не попадает).
// TDD plan #45.
func TestGame_PostChat_RejectsBlank(t *testing.T) {
	_, _, g := twoPlayerGame(t)

	g.PostChat(domain.White, "")
	g.PostChat(domain.White, "   \t\n ")

	require.Empty(t, g.Chat(), "пустые сообщения не сохраняются")
}

// TestGame_PostChat_TrimsSurroundingWhitespace — края текста обрезаются.
// TDD plan #45.
func TestGame_PostChat_TrimsSurroundingWhitespace(t *testing.T) {
	_, _, g := twoPlayerGame(t)

	g.PostChat(domain.White, "  ход за тобой  ")

	history := g.Chat()
	require.Len(t, history, 1)
	require.Equal(t, "ход за тобой", history[0].Text)
}

// TestGame_PostChat_TruncatesToMaxRunes — длинный текст усекается до 500
// СИМВОЛОВ (рун), а не байт: кириллица в UTF-8 по 2 байта, обрезка по байтам
// разорвала бы символ.
// TDD plan #45.
func TestGame_PostChat_TruncatesToMaxRunes(t *testing.T) {
	_, _, g := twoPlayerGame(t)

	g.PostChat(domain.White, strings.Repeat("я", 600))

	history := g.Chat()
	require.Len(t, history, 1)
	require.Equal(t, 500, len([]rune(history[0].Text)), "усечение до 500 рун")
}

// TestGame_PostChat_BroadcastsToBothPlayers — сообщение рассылается обоим
// подключённым клиентам как CHAT с цветом-отправителем (строкой) и текстом.
// TDD plan #46.
func TestGame_PostChat_BroadcastsToBothPlayers(t *testing.T) {
	white, black, g := twoPlayerGame(t)

	g.PostChat(domain.White, "гг")

	chatW := findMessage(white.Messages(), "CHAT")
	require.NotNil(t, chatW, "white должен получить CHAT")
	require.Equal(t, "white", chatW.Sender)
	require.Equal(t, "гг", chatW.Text)

	chatB := findMessage(black.Messages(), "CHAT")
	require.NotNil(t, chatB, "black должен получить CHAT (в т.ч. своё эхо для соперника)")
	require.Equal(t, "white", chatB.Sender)
	require.Equal(t, "гг", chatB.Text)
}

// TestGame_ChatHistoryMessage_BuildsHistory — собирает CHAT_HISTORY со всеми
// сообщениями в порядке отправки; sender — строка-цвет.
// TDD plan #47.
func TestGame_ChatHistoryMessage_BuildsHistory(t *testing.T) {
	_, _, g := twoPlayerGame(t)
	g.PostChat(domain.White, "раз")
	g.PostChat(domain.Black, "два")

	msg := g.ChatHistoryMessage()
	require.NotNil(t, msg)
	require.Equal(t, "CHAT_HISTORY", msg.Type)
	require.Len(t, msg.Chat, 2)
	require.Equal(t, "white", msg.Chat[0].Sender)
	require.Equal(t, "раз", msg.Chat[0].Text)
	require.Equal(t, "black", msg.Chat[1].Sender)
	require.Equal(t, "два", msg.Chat[1].Text)
}

// TestGame_ChatHistoryMessage_NilWhenEmpty — при пустой истории возвращает nil:
// досылать при JOIN нечего (по образцу LegalMovesMessageFor).
// TDD plan #47.
func TestGame_ChatHistoryMessage_NilWhenEmpty(t *testing.T) {
	_, _, g := twoPlayerGame(t)
	require.Nil(t, g.ChatHistoryMessage())
}
