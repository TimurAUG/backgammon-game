package domain

// CanUsePip возвращает true, если у игрока цвета c есть хотя бы один легальный
// шаг или выкид пипсом pip с текущей доски b.
//
// Учитываются только базовые правила движения (IsLegalStep) и сброса
// (IsLegalBearOff). Правило головы и правило шести не проверяются — это
// уровни выше: они сужают набор легальных ходов в контексте конкретного хода,
// но не отменяют сам факт «пипс используется».
//
// Используется для:
//   - #29: если бо́льший пипс используется, а меньший — нет, игрок обязан
//     использовать бо́льший.
//   - #30: если ни один пипс из пары не используется — ход переходит сопернику.
//
// TDD plan #29, #30.
func CanUsePip(b Board, c Color, pip uint8) bool {
	if pip < 1 || pip > 6 {
		return false
	}
	for p := Point(1); p <= 24; p++ {
		if !hasCheckerOf(b, c, p) {
			continue
		}
		next := NextPoint(c, p, pip)
		if next == 0 {
			if IsLegalBearOff(b, c, p, pip) {
				return true
			}
			continue
		}
		if IsLegalStep(b, c, Move{From: p, To: next, Pip: pip}) {
			return true
		}
	}
	return false
}

// hasCheckerOf возвращает true, если на пункте p стоит хотя бы одна шашка
// цвета c.
func hasCheckerOf(b Board, c Color, p Point) bool {
	if p < 1 || p > 24 {
		return false
	}
	switch c {
	case White:
		return b[p-1] > 0
	case Black:
		return b[p-1] < 0
	}
	return false
}

// LegalMoves возвращает список легальных одиночных шагов игрока s.Turn при
// текущей доске и оставшихся пипсах s.Dice.Remaining.
//
// Каждый Move — это либо обычный шаг (To != 0) с легальным IsLegalStep,
// либо выкид (To == 0) с легальным IsLegalBearOff. Дубли по тройке
// (From, To, Pip) удаляются — при дубле в Remaining повторяющиеся пипсы
// порождали бы одинаковые ходы.
//
// Учитывает правило головы через HeadMoveAllowed(HeadConsumed, Dice,
// IsFirstMove). Правило шести (#20) на этом уровне не накладывается —
// это проверка только на END_TURN.
//
// Подготовка к #34 (LEGAL_MOVES в WS-протоколе).
func LegalMoves(s GameState) []Move {
	head := HeadPoint(s.Turn)
	headAllowed := HeadMoveAllowed(s.HeadConsumed[s.Turn], s.Dice, s.IsFirstMove[s.Turn])
	seen := map[Move]bool{}
	moves := make([]Move, 0)
	for _, pip := range s.Dice.Remaining {
		for from := Point(1); from <= 24; from++ {
			if !hasCheckerOf(s.Board, s.Turn, from) {
				continue
			}
			if from == head && !headAllowed {
				continue
			}
			next := NextPoint(s.Turn, from, pip)
			var m Move
			if next == 0 {
				if !IsLegalBearOff(s.Board, s.Turn, from, pip) {
					continue
				}
				m = Move{From: from, To: 0, Pip: pip}
			} else {
				m = Move{From: from, To: next, Pip: pip}
				if !IsLegalStep(s.Board, s.Turn, m) {
					continue
				}
			}
			if seen[m] {
				continue
			}
			seen[m] = true
			moves = append(moves, m)
		}
	}
	return moves
}
