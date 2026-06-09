package domain

import "errors"

// GameState — минимальный снимок партии для оркестрации хода.
//
// Полная модель из SPEC.md содержит также IsFirstMove, Status, Winner, WinKind —
// эти поля будут добавлены на этапе 11 (транспорт/сессии) по мере надобности.
//
// TDD plan #31, #32.
type GameState struct {
	Board    Board
	Turn     Color
	Dice     Dice
	BorneOff [2]uint8
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

	return GameState{
		Board:    newBoard,
		Turn:     s.Turn,
		Dice:     s.Dice.Use(m.Pip),
		BorneOff: newBorneOff,
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
