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
// (From, To, Pip) удаляются.
//
// Учитывает правило головы через HeadMoveAllowed(HeadConsumed, Dice,
// IsFirstMove).
//
// SixBlock lookahead: шаг включается только если существует
// последовательность использования оставшихся после него пипсов, ведущая
// к легальной (по SixBlockAllowed) финальной позиции. Иначе шаг
// отфильтровывается — иначе игрок мог бы зайти в позицию, где END_TURN
// невозможен.
//
// Подготовка к #34 (LEGAL_MOVES в WS-протоколе).
func LegalMoves(s GameState) []Move {
	moves := candidateMoves(s)
	result := make([]Move, 0, len(moves))
	for _, m := range moves {
		ns, err := Apply(s, m)
		if err != nil {
			continue
		}
		if canReachLegalFinal(ns, s.Turn) {
			result = append(result, m)
		}
	}
	return result
}

// candidateMoves генерирует одиночные ходы s.Turn без проверки правила шести
// в перспективе — только базовые правила (направление, цвет цели,
// AllInHome для выкида) и правило головы.
//
// Используется LegalMoves для построения кандидатов перед DFS-проверкой,
// и canReachLegalFinal для рекурсивного перебора в каждом узле.
func candidateMoves(s GameState) []Move {
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

// canReachLegalFinal — true, если из состояния s существует последовательность
// ходов цвета blocker (исходный Turn до старта рекурсии), приводящая к
// финальной позиции, проходящей SixBlockAllowed.
//
// Финал — когда либо Dice.Remaining пуст, либо ни один из оставшихся пипсов
// не используется (CanUsePip == false для всех). В этом случае возвращается
// SixBlockAllowed(Board, blocker).
//
// Иначе перебираются все candidateMoves и рекурсивно проверяется каждый;
// достаточно одной успешной ветки.
func canReachLegalFinal(s GameState, blocker Color) bool {
	if isTerminal(s) {
		return SixBlockAllowed(s.Board, blocker)
	}
	for _, m := range candidateMoves(s) {
		ns, err := Apply(s, m)
		if err != nil {
			continue
		}
		if canReachLegalFinal(ns, blocker) {
			return true
		}
	}
	return false
}

// isTerminal — true, если хода больше нет: Remaining пуст или ни один пипс
// не используется.
func isTerminal(s GameState) bool {
	if len(s.Dice.Remaining) == 0 {
		return true
	}
	for _, pip := range s.Dice.Remaining {
		if CanUsePip(s.Board, s.Turn, pip) {
			return false
		}
	}
	return true
}
