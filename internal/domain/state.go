package domain

import "errors"

// GameStatus — фаза партии. Используется и доменом, и транспортом.
type GameStatus uint8

const (
	// StatusWaitingForRoll — ход уже за определённым игроком, но кубики
	// ещё не брошены. Также используется как начальный статус до
	// ROLL_FOR_FIRST: оба игрока должны прислать сигнал.
	StatusWaitingForRoll GameStatus = iota
	// StatusWaitingForMove — кубики брошены, текущий игрок ходит.
	StatusWaitingForMove
	// StatusFinished — партия окончена.
	StatusFinished
)

// GameState — снимок партии для оркестрации хода.
//
// HeadConsumed[c] — сколько шашек цвета c снято с головы в текущем ходу.
// Обнуляется на END_TURN. Используется в LegalMoves для правила головы.
//
// IsFirstMove[c] — это первый ход партии для цвета c (для исключения
// дублей 6:6/4:4/3:3 в правиле головы). Сбрасывается в false после первого
// END_TURN этого цвета.
//
// Поля Winner, WinKind будут добавлены позже по мере надобности транспорта.
//
// TDD plan #31, #32.
type GameState struct {
	Board        Board
	Turn         Color
	Dice         Dice
	BorneOff     [2]uint8
	Status       GameStatus
	HeadConsumed [2]uint8
	IsFirstMove  [2]bool
}

// HeadPoint возвращает пункт головы цвета c: 24 для белых, 12 для чёрных.
func HeadPoint(c Color) Point {
	if c == Black {
		return 12
	}
	return 24
}

// ErrIllegalMove возвращается из Apply, если ход нелегален: целевая клетка
// занята соперником, шаг не соответствует пипсу, выкид без AllInHome и т.п.
var ErrIllegalMove = errors.New("illegal move")

// ErrPipNotInRemaining возвращается из Apply, если пипс хода отсутствует
// в Dice.Remaining (игрок пытается потратить пипс, которого у него нет).
var ErrPipNotInRemaining = errors.New("pip not in remaining")

// Apply применяет ход m к состоянию s и возвращает новое состояние.
//
// Не учитывает правило головы и правило шести — это уровни выше
// (LegalMoves / END_TURN). На этом уровне проверяются:
//   - наличие пипса в Dice.Remaining;
//   - легальность шага через IsLegalStep или выкида через IsLegalBearOff.
//
// При ошибке возвращает исходное состояние без изменений.
//
// TDD plan #31.
func Apply(s GameState, m Move) (GameState, error) {
	if !pipInRemaining(s.Dice.Remaining, m.Pip) {
		return s, ErrPipNotInRemaining
	}
	if m.To == 0 {
		if !IsLegalBearOff(s.Board, s.Turn, m.From, m.Pip) {
			return s, ErrIllegalMove
		}
	} else {
		if !IsLegalStep(s.Board, s.Turn, m) {
			return s, ErrIllegalMove
		}
	}

	newBoard := s.Board
	switch s.Turn {
	case White:
		newBoard[m.From-1]--
	case Black:
		newBoard[m.From-1]++
	}

	newBorneOff := s.BorneOff
	if m.To == 0 {
		newBorneOff[s.Turn]++
	} else {
		switch s.Turn {
		case White:
			newBoard[m.To-1]++
		case Black:
			newBoard[m.To-1]--
		}
	}

	newHeadConsumed := s.HeadConsumed
	if m.From == HeadPoint(s.Turn) {
		newHeadConsumed[s.Turn]++
	}

	return GameState{
		Board:        newBoard,
		Turn:         s.Turn,
		Dice:         s.Dice.Use(m.Pip),
		BorneOff:     newBorneOff,
		Status:       s.Status,
		HeadConsumed: newHeadConsumed,
		IsFirstMove:  s.IsFirstMove,
	}, nil
}

// IsTurnComplete возвращает true, если у текущего игрока больше нет ходов
// в этом ходу: либо Dice.Remaining пуст, либо ни один из оставшихся пипсов
// невозможно использовать на текущей доске.
//
// TDD plan #32.
func IsTurnComplete(s GameState) bool {
	if len(s.Dice.Remaining) == 0 {
		return true
	}
	for _, p := range s.Dice.Remaining {
		if CanUsePip(s.Board, s.Turn, p) {
			return false
		}
	}
	return true
}

func pipInRemaining(remaining []uint8, pip uint8) bool {
	for _, p := range remaining {
		if p == pip {
			return true
		}
	}
	return false
}
