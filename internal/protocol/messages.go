// Package protocol описывает JSON-сообщения между клиентом и сервером
// backgammon-game. Поле Type — обязательный дискриминатор.
//
// Полная спецификация — в .claude/skills/nardy-protocol/SKILL.md.
package protocol

// ClientMessage — сообщение от клиента серверу.
//
// На данный момент включает поля для JOIN и MOVE. По мере реализации
// последующих циклов сюда будут добавлены поля для ROLL, END_TURN и т.п.
type ClientMessage struct {
	Type   string `json:"type"`
	GameID string `json:"gameId,omitempty"`
	Token  string `json:"token,omitempty"`
	From   uint8  `json:"from,omitempty"`
	To     uint8  `json:"to,omitempty"`
	Text   string `json:"text,omitempty"` // CHAT: текст сообщения
}

// DicePayload — представление Dice в JSON-протоколе.
//
// Remaining — []int, а НЕ []uint8: encoding/json кодирует []uint8 ([]byte)
// в base64-строку, а контракт (nardy-protocol) требует массив чисел number[].
type DicePayload struct {
	A         uint8 `json:"a"`
	B         uint8 `json:"b"`
	IsDouble  bool  `json:"isDouble"`
	Remaining []int `json:"remaining"`
}

// MovePayload — одиночный ход в LEGAL_MOVES.
// To == 0 означает выкид.
type MovePayload struct {
	From uint8 `json:"from"`
	To   uint8 `json:"to"`
	Pip  uint8 `json:"pip"`
}

// ReachPayload — достижимая цель ОДНОЙ шашки в LEGAL_MOVES, в том числе
// составным ходом несколькими кубиками. Дополняет Moves (одиночные шаги),
// не заменяет: Moves остаётся атомарной правдой для валидации MOVE.
//
// From — исходный пункт (1..24). Path — пункты-остановки по порядку (каждый —
// цель отдельного MOVE), последний элемент = финальная цель. Pips — кубики,
// потраченные на каждый шаг, len(Pips) == len(Path). Для одиночной цели длина
// массивов равна 1. Выкид (To == 0) сюда не входит — это отдельная зона UI.
//
// Path/Pips — []int, а НЕ []uint8: encoding/json кодирует []uint8 в base64,
// а контракт требует массив чисел (как Remaining у DicePayload).
type ReachPayload struct {
	From uint8 `json:"from"`
	Path []int `json:"path"`
	Pips []int `json:"pips"`
}

// ChatPayload — одно сообщение в CHAT_HISTORY. Sender — цвет автора строкой
// ("white"/"black").
type ChatPayload struct {
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

// BorneOffPayload — счёт выкинутых шашек в STATE.
type BorneOffPayload struct {
	White uint8 `json:"white"`
	Black uint8 `json:"black"`
}

// IsFirstMovePayload — флаг «ещё не сделал ни одного END_TURN» для
// каждого цвета. Нужен клиенту, чтобы понимать, действует ли
// исключение для дублей 6:6/4:4/3:3 на первом ходу.
type IsFirstMovePayload struct {
	White bool `json:"white"`
	Black bool `json:"black"`
}

// AllHomePayload — флаг «все шашки цвета в его доме» (фаза сброса) для
// каждого цвета. Значение доменного предиката AllInHome. Нужен клиенту,
// чтобы показывать счётчик оставшихся к сбросу шашек только в фазе сброса,
// не дублируя правило «все дома» на клиенте.
type AllHomePayload struct {
	White bool `json:"white"`
	Black bool `json:"black"`
}

// FirstRollPayload — индивидуальные броски обоих цветов при определении
// первого хода (кто больше — ходит первым). Шлётся в FIRST_ROLL, чтобы
// клиент показал «кто сколько бросил».
type FirstRollPayload struct {
	White int `json:"white"`
	Black int `json:"black"`
}

// ServerMessage — сообщение от сервера клиенту.
//
// Общий тип для STATE и ERROR — наименьшее количество разных структур
// упрощает чтение и сериализацию. Когда состав сообщений вырастет, можно
// будет разнести на отдельные типы с json.RawMessage payload.
type ServerMessage struct {
	Type        string              `json:"type"`
	Color       string              `json:"color,omitempty"` // JOINED: цвет присоединившегося
	Board       []int8              `json:"board,omitempty"`
	Turn        string              `json:"turn,omitempty"`
	Status      string              `json:"status,omitempty"`
	Dice        *DicePayload        `json:"dice,omitempty"`
	BorneOff    *BorneOffPayload    `json:"borneOff,omitempty"`
	IsFirstMove *IsFirstMovePayload `json:"isFirstMove,omitempty"`
	AllHome     *AllHomePayload     `json:"allHome,omitempty"`
	FirstRoll   *FirstRollPayload   `json:"firstRoll,omitempty"`
	Moves       []MovePayload       `json:"moves,omitempty"`
	Reach       []ReachPayload      `json:"reach,omitempty"`
	Winner      string              `json:"winner,omitempty"`
	Kind        string              `json:"kind,omitempty"`
	Code        string              `json:"code,omitempty"`
	Message     string              `json:"message,omitempty"`
	Sender      string              `json:"sender,omitempty"` // CHAT: цвет автора ("white"/"black")
	Text        string              `json:"text,omitempty"`   // CHAT: текст сообщения
	Chat        []ChatPayload       `json:"chat,omitempty"`   // CHAT_HISTORY: вся история
}
