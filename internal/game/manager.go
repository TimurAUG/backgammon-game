// Package game содержит менеджер игровых сессий: создание игр, регистрация
// игроков, доступ к состоянию.
//
// Хранение пока in-memory; persistence через Postgres появится на #36.
package game

import (
	crand "crypto/rand"
	"errors"
	"io"
	"sync"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/protocol"
)

// Conn — абстрактный «канал» к одному клиенту. Реализуется на уровне
// транспорта (WS) и используется в game для рассылки сообщений.
type Conn interface {
	Send(msg protocol.ServerMessage) error
}

// ErrRoomFull возвращается из JoinGame, если в игре уже зарегистрированы оба
// игрока.
var ErrRoomFull = errors.New("room full")

// ErrMustUsePip возвращается из EndTurn, если у текущего игрока ещё есть
// пипсы, которые можно использовать.
var ErrMustUsePip = errors.New("must use pip")

// ErrNotYourTurn возвращается из методов хода, если действие пришло не от
// игрока, чей сейчас ход.
var ErrNotYourTurn = errors.New("not your turn")

// ErrRuleOfSix возвращается из EndTurn, если финальная позиция нарушает
// правило шести (блок 6+ у игрока + соперник 0 в своём доме).
var ErrRuleOfSix = errors.New("rule of six violation")

// ErrInvalidState возвращается, если действие не подходит к текущему
// статусу партии (например, ROLL при WaitingForMove).
var ErrInvalidState = errors.New("invalid state")

// ErrGameNotFound возвращается из JoinByID, если игры с заданным id нет
// (вход по приглашению в несуществующую/устаревшую игру).
var ErrGameNotFound = errors.New("game not found")

// Game — одна партия в памяти.
//
// Содержит идентификатор, доменное состояние, источник случайности для
// бросков и две позиции для соединений игроков (индексируются по
// domain.Color). Параллельно хранятся токены — для реконнекта.
//
// Поля conns/tokens/rolledForFirst защищены mu.
type Game struct {
	ID    string
	State domain.GameState

	mu             sync.Mutex
	conns          [2]Conn
	tokens         [2]string
	rng            io.Reader
	rolledForFirst [2]bool
	chat           []ChatMessage

	// storage — куда персистится игра. nil → in-memory only (тесты домена,
	// memoryStorage). Установлен Manager.JoinGame после Load/Create.
	storage Storage
}

// maybePersist сохраняет текущее состояние в Storage, если он задан.
// Обычно вызывается через defer после g.mu.Unlock() (LIFO порядок defer'ов
// гарантирует, что Unlock сработает раньше).
func (g *Game) maybePersist() {
	if g.storage == nil {
		return
	}
	_ = g.storage.SaveGame(g)
}

// Tokens возвращает копию токенов игроков. Для инспекции (тесты,
// сериализация в Storage).
func (g *Game) Tokens() [2]string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.tokens
}

// RolledForFirst возвращает копию счётчиков «бросал ли для определения
// первого хода». Для инспекции.
func (g *Game) RolledForFirst() [2]bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.rolledForFirst
}

// Opponent возвращает соединение соперника для цвета c или nil, если соперник
// не подключён.
func (g *Game) Opponent(c domain.Color) Conn {
	g.mu.Lock()
	defer g.mu.Unlock()
	if c == domain.White {
		return g.conns[domain.Black]
	}
	return g.conns[domain.White]
}

// Detach снимает регистрацию соединения conn цвета c и сообщает, отключены ли
// теперь оба игрока (тогда Manager выгрузит партию из реестра). Идемпотентно.
//
// Слот обнуляется ТОЛЬКО если в нём именно это conn. Иначе горутина старого
// (отвалившегося) соединения затёрла бы свежий реконнект, занявший тот же слот,
// и broadcast перестал бы доходить до вернувшегося игрока.
func (g *Game) Detach(c domain.Color, conn Conn) (bothDisconnected bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conns[c] == conn {
		g.conns[c] = nil
	}
	return g.conns[domain.White] == nil && g.conns[domain.Black] == nil
}

// HandleMove обрабатывает MOVE от игрока цвета c. Вычисляет pip по From/To
// через NextPoint, выполняет Apply, рассылает STATE обоим и LEGAL_MOVES
// активному игроку.
//
// Возвращает ошибку, если ход не текущего игрока, неверный from/to,
// или Apply вернул ошибку. Сейчас ошибки молча проглатываются — позже
// будут конвертироваться в ERROR сообщения (#34e+).
//
// TDD plan #34 (часть 4).
func (g *Game) HandleMove(c domain.Color, from, to uint8) error {
	g.mu.Lock()
	defer g.maybePersist()
	defer g.mu.Unlock()

	if g.State.Turn != c {
		return ErrNotYourTurn
	}
	pip, ok := findPipFor(g.State, domain.Point(from), domain.Point(to))
	if !ok {
		return errors.New("no matching pip")
	}
	newState, err := domain.Apply(g.State, domain.Move{
		From: domain.Point(from),
		To:   domain.Point(to),
		Pip:  pip,
	})
	if err != nil {
		return err
	}
	g.State = newState

	if winner, kind, finished := domain.Winner(g.State.Board, g.State.BorneOff); finished {
		g.State.Status = domain.StatusFinished
		g.broadcastStateLocked()
		g.broadcastGameOverLocked(winner, kind)
		return nil
	}

	g.broadcastStateLocked()
	g.sendLegalMovesOrAutoEndTurnLocked(g.State.Turn)
	return nil
}

// broadcastGameOverLocked рассылает GAME_OVER обоим подключённым клиентам.
// Вызывается с захваченным mu.
func (g *Game) broadcastGameOverLocked(winner domain.Color, kind domain.WinKind) {
	msg := protocol.ServerMessage{
		Type:   "GAME_OVER",
		Winner: colorString(winner),
		Kind:   winKindString(kind),
	}
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

func winKindString(k domain.WinKind) string {
	switch k {
	case domain.Mars:
		return "mars"
	case domain.Koks:
		return "koks"
	default:
		return "oin"
	}
}

// findPipFor подбирает значение пипса из Remaining такое, что ход (from→to)
// валиден. Для выкида (to==0) предпочитает точный пипс над переборным.
func findPipFor(s domain.GameState, from, to domain.Point) (uint8, bool) {
	if to == 0 {
		// Выкид: пробуем сначала точный пипс, потом переборные.
		// Точный пипс для белого = from; для чёрного = from - 12
		// (пункт 13 → пип 1, пункт 18 → пип 6). Согласовано с NextPoint.
		exact := uint8(from)
		if s.Turn == domain.Black {
			exact = uint8(from) - 12
		}
		for _, p := range s.Dice.Remaining {
			if p == exact && domain.IsLegalBearOff(s.Board, s.Turn, from, p) {
				return p, true
			}
		}
		for _, p := range s.Dice.Remaining {
			if domain.IsLegalBearOff(s.Board, s.Turn, from, p) {
				return p, true
			}
		}
		return 0, false
	}
	for _, p := range s.Dice.Remaining {
		if domain.NextPoint(s.Turn, from, p) != to {
			continue
		}
		if domain.IsLegalStep(s.Board, s.Turn, domain.Move{From: from, To: to, Pip: p}) {
			return p, true
		}
	}
	return 0, false
}

// Roll бросает два кубика для игрока цвета c через rng и устанавливает
// Status в WaitingForMove. Допустим только если c == Turn и Status ==
// WaitingForRoll.
//
// После броска рассылает STATE обоим и LEGAL_MOVES активному.
//
// TDD plan #34 (часть 7 — ROLL).
func (g *Game) Roll(c domain.Color) error {
	g.mu.Lock()
	defer g.maybePersist()
	defer g.mu.Unlock()

	if g.State.Turn != c {
		return ErrNotYourTurn
	}
	if g.State.Status != domain.StatusWaitingForRoll {
		return ErrInvalidState
	}
	dice, err := domain.RollDice(g.rng)
	if err != nil {
		return err
	}
	g.State.Dice = dice
	g.State.Status = domain.StatusWaitingForMove
	g.broadcastStateLocked()
	g.sendLegalMovesOrAutoEndTurnLocked(c)
	return nil
}

// EndTurn передаёт ход сопернику.
//
// Если у текущего игрока ещё остались пипсы, которые можно использовать
// (см. domain.IsTurnComplete), возвращает ErrMustUsePip.
//
// При успехе обнуляет HeadConsumed, сбрасывает Dice, ставит Status в
// StatusWaitingForRoll, переключает Turn и помечает IsFirstMove[c] = false.
// Рассылает STATE обоим клиентам.
//
// Правило шести (SixBlockAllowed на финальной позиции) на этом этапе НЕ
// проверяется — будет добавлено следующим циклом (закрытие #20).
//
// TDD plan #34 (часть 5).
func (g *Game) EndTurn(c domain.Color) error {
	g.mu.Lock()
	defer g.maybePersist()
	defer g.mu.Unlock()

	if g.State.Turn != c {
		return ErrNotYourTurn
	}
	if !domain.IsTurnComplete(g.State) {
		return ErrMustUsePip
	}
	if !domain.SixBlockAllowed(g.State.Board, c) {
		return ErrRuleOfSix
	}

	next := domain.Black
	if c == domain.Black {
		next = domain.White
	}

	isFirstMove := g.State.IsFirstMove
	isFirstMove[c] = false

	g.State = domain.GameState{
		Board:        g.State.Board,
		Turn:         next,
		Dice:         domain.Dice{},
		BorneOff:     g.State.BorneOff,
		Status:       domain.StatusWaitingForRoll,
		HeadConsumed: [2]uint8{},
		IsFirstMove:  isFirstMove,
	}
	g.broadcastStateLocked()
	return nil
}

// RollForFirst обрабатывает сигнал «готов» от игрока c в фазе определения
// первого хода.
//
// Семантика: команда не означает «бросок» сама по себе — это сигнал готовности.
// Когда оба игрока сигналили, сервер бросает по одному кубику обоим
// (White первым, Black вторым по rng), при равенстве — переброс до победителя.
// Розыгрыш определяет ТОЛЬКО очередность: State.Turn = победитель, Dice пуст,
// Status = WaitingForRoll. Значения розыгрыша в ход не переносятся — победитель
// затем делает обычный ROLL. STATE и FIRST_ROLL рассылаются обоим; LEGAL_MOVES
// не шлётся (придёт после ROLL победителя).
//
// Если c уже сигналил ранее — игнорирует.
//
// TDD plan #34 (часть 2).
func (g *Game) RollForFirst(c domain.Color) error {
	g.mu.Lock()
	defer g.maybePersist()
	defer g.mu.Unlock()

	if g.rolledForFirst[c] {
		return nil
	}
	g.rolledForFirst[c] = true

	if !(g.rolledForFirst[domain.White] && g.rolledForFirst[domain.Black]) {
		return nil
	}

	var firstWhite, firstBlack uint8
	for {
		whiteVal, err := domain.RollOne(g.rng)
		if err != nil {
			return err
		}
		blackVal, err := domain.RollOne(g.rng)
		if err != nil {
			return err
		}
		if whiteVal == blackVal {
			continue
		}
		firstWhite, firstBlack = whiteVal, blackVal
		winner := domain.White
		if blackVal > whiteVal {
			winner = domain.Black
		}
		g.State.Turn = winner
		g.State.Dice = domain.Dice{}
		g.State.Status = domain.StatusWaitingForRoll
		break
	}

	g.broadcastStateLocked()
	g.broadcastFirstRollLocked(firstWhite, firstBlack)
	return nil
}

// broadcastFirstRollLocked рассылает индивидуальные броски первого хода обоим
// клиентам (FIRST_ROLL) — чтобы показать «кто сколько бросил». С захваченным mu.
func (g *Game) broadcastFirstRollLocked(white, black uint8) {
	msg := protocol.ServerMessage{
		Type:      "FIRST_ROLL",
		FirstRoll: &protocol.FirstRollPayload{White: int(white), Black: int(black)},
	}
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

// broadcastStateLocked рассылает текущее состояние всем подключённым
// соединениям. Вызывается с захваченным mu.
func (g *Game) broadcastStateLocked() {
	msg := StateMessage(g.State)
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

// broadcastTurnSkippedLocked сообщает обоим, что ход игрока c пропущен из-за
// отсутствия легальных ходов (с выпавшими кубиками d). Вызывается под mu, до
// обнуления Dice в autoEndTurnLocked.
func (g *Game) broadcastTurnSkippedLocked(c domain.Color, d domain.Dice) {
	msg := protocol.ServerMessage{
		Type:  "TURN_SKIPPED",
		Color: colorString(c),
		Dice:  dicePayload(d),
	}
	for _, conn := range g.conns {
		if conn != nil {
			_ = conn.Send(msg)
		}
	}
}

// sendLegalMovesOrAutoEndTurnLocked отправляет LEGAL_MOVES игроку цвета c
// если есть хотя бы один легальный ход. Если ходов нет — автоматически
// передаёт ход сопернику через autoEndTurnLocked. Вызывается с захваченным mu.
//
// SPEC: «Если массив пуст — у игрока нет легальных продолжений, сервер
// автоматически передаст ход (через внутренний END_TURN)».
func (g *Game) sendLegalMovesOrAutoEndTurnLocked(c domain.Color) {
	moves := domain.LegalMoves(g.State)
	if len(moves) > 0 {
		if g.conns[c] == nil {
			return
		}
		payload := make([]protocol.MovePayload, len(moves))
		for i, m := range moves {
			payload[i] = protocol.MovePayload{From: uint8(m.From), To: uint8(m.To), Pip: m.Pip}
		}
		_ = g.conns[c].Send(protocol.ServerMessage{Type: "LEGAL_MOVES", Moves: payload})
		return
	}
	g.autoEndTurnLocked(c)
}

// autoEndTurnLocked передаёт ход сопернику без проверки IsTurnComplete
// (она избыточна — мы попали сюда потому что ходов нет). Проверка
// SixBlockAllowed остаётся: при нарушении шлём ERROR обоим клиентам,
// ход не передаём — игра застряет.
func (g *Game) autoEndTurnLocked(c domain.Color) {
	if !domain.SixBlockAllowed(g.State.Board, c) {
		msg := protocol.ServerMessage{
			Type: "ERROR", Code: "RULE_OF_SIX",
			Message: "final position violates six rule",
		}
		for _, conn := range g.conns {
			if conn != nil {
				_ = conn.Send(msg)
			}
		}
		return
	}
	// «Бросил, а ходить нечем»: если игрок не использовал НИ ОДНОГО пипса
	// (полный набор остался) — шлём обоим TURN_SKIPPED с выпавшими кубиками,
	// чтобы было видно, почему очередь сразу перешла. Если часть пипсов уже
	// использована (auto-END в середине хода) — это не «нечем ходить», молчим.
	fullPips := 2
	if g.State.Dice.IsDouble {
		fullPips = 4
	}
	if len(g.State.Dice.Remaining) == fullPips {
		g.broadcastTurnSkippedLocked(c, g.State.Dice)
	}
	next := domain.Black
	if c == domain.Black {
		next = domain.White
	}
	isFirstMove := g.State.IsFirstMove
	isFirstMove[c] = false
	g.State = domain.GameState{
		Board:        g.State.Board,
		Turn:         next,
		Dice:         domain.Dice{},
		BorneOff:     g.State.BorneOff,
		Status:       domain.StatusWaitingForRoll,
		HeadConsumed: [2]uint8{},
		IsFirstMove:  isFirstMove,
	}
	g.broadcastStateLocked()
}

// JoinedMessage собирает JOINED-сообщение — ответ присоединившемуся
// клиенту с его цветом. Шлётся до STATE: из STATE цвет не узнать,
// он одинаков для обоих игроков.
func JoinedMessage(c domain.Color) protocol.ServerMessage {
	return protocol.ServerMessage{Type: "JOINED", Color: colorString(c)}
}

// StateMessage конвертирует доменное состояние в STATE-сообщение протокола.
func StateMessage(s domain.GameState) protocol.ServerMessage {
	board := make([]int8, len(s.Board))
	for i, v := range s.Board {
		board[i] = v
	}
	msg := protocol.ServerMessage{
		Type:   "STATE",
		Board:  board,
		Turn:   colorString(s.Turn),
		Status: statusString(s.Status),
		BorneOff: &protocol.BorneOffPayload{
			White: s.BorneOff[domain.White],
			Black: s.BorneOff[domain.Black],
		},
		IsFirstMove: &protocol.IsFirstMovePayload{
			White: s.IsFirstMove[domain.White],
			Black: s.IsFirstMove[domain.Black],
		},
	}
	if len(s.Dice.Remaining) > 0 || s.Dice.A != 0 || s.Dice.B != 0 {
		msg.Dice = dicePayload(s.Dice)
	}
	return msg
}

// dicePayload конвертирует доменный Dice в JSON-payload. Remaining — []int,
// а не []uint8 (см. DicePayload).
func dicePayload(d domain.Dice) *protocol.DicePayload {
	remaining := make([]int, len(d.Remaining))
	for i, p := range d.Remaining {
		remaining[i] = int(p)
	}
	return &protocol.DicePayload{A: d.A, B: d.B, IsDouble: d.IsDouble, Remaining: remaining}
}

// LegalMovesMessageFor возвращает LEGAL_MOVES-сообщение для игрока color, если
// сейчас его ход в фазе waitingForMove — для досылки при (ре)коннекте, чтобы
// клиент сразу видел доступные ходы (инвариант протокола; критично после
// рестарта сервера). Иначе nil. Сообщение отправляет вызывающий (handler)
// через своё соединение, поэтому метод не трогает g.conns.
func (g *Game) LegalMovesMessageFor(color domain.Color) *protocol.ServerMessage {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.State.Status != domain.StatusWaitingForMove || g.State.Turn != color {
		return nil
	}
	moves := domain.LegalMoves(g.State)
	payload := make([]protocol.MovePayload, len(moves))
	for i, mv := range moves {
		payload[i] = protocol.MovePayload{From: uint8(mv.From), To: uint8(mv.To), Pip: mv.Pip}
	}
	return &protocol.ServerMessage{Type: "LEGAL_MOVES", Moves: payload}
}

func colorString(c domain.Color) string {
	if c == domain.Black {
		return "black"
	}
	return "white"
}

func statusString(s domain.GameStatus) string {
	switch s {
	case domain.StatusWaitingForMove:
		return "waitingForMove"
	case domain.StatusFinished:
		return "finished"
	default:
		return "waitingForRoll"
	}
}

// Manager хранит активные игры через Storage и генерирует броски кубиков
// через rng. Безопасен для одновременного доступа из горутин WS-handler'а.
//
// active — in-memory реестр «живых» партий: один объект Game на активную
// игру, держащий WS-соединения (conns). Storage переживает рестарт процесса,
// но LoadGame отдаёт новый объект на каждый вызов — с PostgresStorage два
// игрока иначе попали бы в разные объекты, и broadcast не дошёл бы до
// соперника. Реестр гарантирует общий объект, пока партия активна.
type Manager struct {
	storage Storage
	rng     io.Reader
	ids     io.Reader // источник для gameID/token (crypto/rand; фикс-reader в тестах)

	mu     sync.Mutex
	active map[string]*Game
}

// NewManager создаёт менеджер с in-memory Storage и crypto/rand в качестве
// источника случайности.
func NewManager() *Manager {
	return &Manager{storage: NewMemoryStorage(), rng: crand.Reader, ids: crand.Reader, active: map[string]*Game{}}
}

// NewManagerWithRand — конструктор для тестов с фиксированным rng
// (bytes.Reader с заранее подготовленными байтами). Использует in-memory
// Storage.
func NewManagerWithRand(rng io.Reader) *Manager {
	return &Manager{storage: NewMemoryStorage(), rng: rng, ids: crand.Reader, active: map[string]*Game{}}
}

// NewManagerWithStorage позволяет подключить произвольный Storage —
// например, Postgres (#36). Источник случайности задаётся отдельно.
func NewManagerWithStorage(storage Storage, rng io.Reader) *Manager {
	return &Manager{storage: storage, rng: rng, ids: crand.Reader, active: map[string]*Game{}}
}

// CreateGame генерит новую игру: случайные gameID (8 байт) и creator-token
// (16 байт) через ids, начальную доску, creator-токен в слоте White (Black
// свободен — туда войдёт второй игрок через JoinByID). Сохраняет и возвращает
// gameID и token для передачи клиенту.
func (m *Manager) CreateGame() (string, string, error) {
	gameID, err := generateID(m.ids, 8)
	if err != nil {
		return "", "", err
	}
	token, err := generateID(m.ids, 16)
	if err != nil {
		return "", "", err
	}
	g := &Game{
		ID: gameID,
		State: domain.GameState{
			Board:       domain.InitialBoard(),
			Turn:        domain.White,
			Status:      domain.StatusWaitingForRoll,
			IsFirstMove: [2]bool{true, true},
		},
	}
	g.rng = m.rng
	g.storage = m.storage
	g.tokens[domain.White] = token
	if err := m.storage.SaveGame(g); err != nil {
		return "", "", err
	}
	return gameID, token, nil
}

// JoinByID резервирует свободный слот в существующей игре под нового игрока:
// генерит token (16 байт), кладёт в первый пустой слот tokens, сохраняет.
// Возвращает token для клиента. ErrGameNotFound если игры нет, ErrRoomFull
// если оба слота заняты. WS-JOIN с этим token подключит игрока к слоту
// (attachLocked сопоставляет по совпадению токена).
func (m *Manager) JoinByID(gameID string) (string, error) {
	g, ok := m.storage.LoadGame(gameID)
	if !ok {
		return "", ErrGameNotFound
	}
	g.rng = m.rng
	g.storage = m.storage

	token, err := generateID(m.ids, 16)
	if err != nil {
		return "", err
	}

	g.mu.Lock()
	reserved := false
	for c := 0; c < 2; c++ {
		if g.tokens[c] == "" {
			g.tokens[c] = token
			reserved = true
			break
		}
	}
	g.mu.Unlock()

	if !reserved {
		return "", ErrRoomFull
	}
	if err := m.storage.SaveGame(g); err != nil {
		return "", err
	}
	return token, nil
}

// JoinGame регистрирует соединение conn в игре с id.
//
// Логика:
//   - Если игры с id ещё нет — создаётся с начальной доской и Turn=White.
//   - Реконнект: если token != "" и совпадает с сохранённым в одном из
//     слотов — соединение заменяется на conn в том же слоте, возвращается
//     прежний цвет.
//   - Новый игрок: ищется слот, где token == "" и conn == nil (полностью
//     свободный). Если такой найден — занимается, token сохраняется (даже
//     пустой).
//   - Если ни реконнект, ни свободного слота нет — ErrRoomFull.
//
// При token == "" реконнект невозможен (на сервере нет способа отличить
// одного клиента от другого).
func (m *Manager) JoinGame(id, token string, conn Conn) (domain.Color, *Game, error) {
	// Берём live-объект из реестра, если партия уже активна (общие conns).
	// Иначе поднимаем из Storage (восстановление после рестарта) или создаём
	// новую и регистрируем. Создание/поиск — под m.mu, чтобы два одновременных
	// JOIN новой игры не создали два объекта.
	m.mu.Lock()
	g, ok := m.active[id]
	if !ok {
		loaded, found := m.storage.LoadGame(id)
		if found {
			g = loaded
		} else {
			g = &Game{
				ID: id,
				State: domain.GameState{
					Board:       domain.InitialBoard(),
					Turn:        domain.White,
					Status:      domain.StatusWaitingForRoll,
					IsFirstMove: [2]bool{true, true},
				},
			}
		}
		g.rng = m.rng
		g.storage = m.storage
		m.active[id] = g
	}
	m.mu.Unlock()

	g.mu.Lock()
	color, err := g.attachLocked(token, conn)
	g.mu.Unlock()
	if err != nil {
		return 0, nil, err
	}

	_ = m.storage.SaveGame(g)
	return color, g, nil
}

// Leave отсоединяет игрока цвета color от партии id и, если отключились оба,
// выгружает её из реестра активных игр — память не растёт по числу когда-либо
// открытых партий. Состояние уже персистится в Storage после каждого
// действия, поэтому повторный JOIN поднимет игру заново. Идемпотентно.
func (m *Manager) Leave(id string, color domain.Color, conn Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.active[id]
	if !ok {
		return
	}
	if g.Detach(color, conn) {
		delete(m.active, id)
	}
}

// ActiveCount возвращает число партий в in-memory реестре. Для инспекции
// (тесты жизненного цикла реестра).
func (m *Manager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.active)
}

// attachLocked регистрирует conn в свободном слоте или реконнектит в слот с
// тем же token. Должен вызываться под g.mu.
func (g *Game) attachLocked(token string, conn Conn) (domain.Color, error) {
	if token != "" {
		for c := 0; c < 2; c++ {
			if g.tokens[c] == token {
				g.conns[c] = conn
				return domain.Color(c), nil
			}
		}
	}
	for c := 0; c < 2; c++ {
		if g.tokens[c] == "" && g.conns[c] == nil {
			g.tokens[c] = token
			g.conns[c] = conn
			return domain.Color(c), nil
		}
	}
	return 0, ErrRoomFull
}
