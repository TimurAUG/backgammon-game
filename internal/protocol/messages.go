// Package protocol описывает JSON-сообщения между клиентом и сервером
// backgammon-game. Поле Type — обязательный дискриминатор.
//
// Полная спецификация — в .claude/skills/nardy-protocol/SKILL.md.
package protocol

// ClientMessage — сообщение от клиента серверу.
//
// На данный момент включает только минимально необходимые поля для #33
// (JOIN). По мере реализации последующих циклов сюда будут добавлены
// поля для ROLL, MOVE, END_TURN и т.п.
type ClientMessage struct {
	Type   string `json:"type"`
	GameID string `json:"gameId,omitempty"`
	Token  string `json:"token,omitempty"`
}

// DicePayload — представление Dice в JSON-протоколе.
type DicePayload struct {
	A         uint8   `json:"a"`
	B         uint8   `json:"b"`
	IsDouble  bool    `json:"isDouble"`
	Remaining []uint8 `json:"remaining"`
}

// ServerMessage — сообщение от сервера клиенту.
//
// Общий тип для STATE и ERROR — наименьшее количество разных структур
// упрощает чтение и сериализацию. Когда состав сообщений вырастет, можно
// будет разнести на отдельные типы с json.RawMessage payload.
type ServerMessage struct {
	Type    string       `json:"type"`
	Board   []int8       `json:"board,omitempty"`
	Turn    string       `json:"turn,omitempty"`
	Status  string       `json:"status,omitempty"`
	Dice    *DicePayload `json:"dice,omitempty"`
	Code    string       `json:"code,omitempty"`
	Message string       `json:"message,omitempty"`
}
