package domain

// SixBlockAllowed возвращает true, если финальная позиция b легальна по
// правилу шести для блокирующего цвета blocker.
//
// Правило шести запрещает финальную позицию, если выполнены ОБА условия:
//  1. У соперника нет ни одной шашки в его собственном доме (дом чёрных —
//     пункты 13..18, дом белых — пункты 1..6).
//  2. У блокирующего получился блок из 6 или более подряд занятых им пунктов.
//
// «Подряд» учитывает обёртку границы 24↔1: доска геометрически кольцевая,
// блок 22..24+1..3 — это 6 подряд занятых пунктов.
//
// ВАЖНО: проверка применима ТОЛЬКО к финальной позиции после END_TURN.
// Промежуточные позиции внутри хода не проверяются (см. SPEC #20, покрывается
// на этапе 10 в Apply/IsTurnComplete).
//
// TDD plan #18, #19.
func SixBlockAllowed(b Board, blocker Color) bool {
	if opponentHasInOwnHome(b, blocker) {
		return true
	}
	return !hasSixInARow(b, blocker)
}

// opponentHasInOwnHome — есть ли у соперника blocker'а хотя бы одна шашка
// в его собственном доме.
func opponentHasInOwnHome(b Board, blocker Color) bool {
	switch blocker {
	case White:
		// Соперник — чёрный, его дом: пункты 13..18 (индексы 12..17).
		for i := 12; i <= 17; i++ {
			if b[i] < 0 {
				return true
			}
		}
	case Black:
		// Соперник — белый, его дом: пункты 1..6 (индексы 0..5).
		for i := 0; i <= 5; i++ {
			if b[i] > 0 {
				return true
			}
		}
	}
	return false
}

// hasSixInARow — true, если у blocker'а есть блок из 6+ подряд занятых
// пунктов. Учёт обёртки 24↔1 реализован двойным проходом по индексам.
func hasSixInARow(b Board, blocker Color) bool {
	occupied := func(i int) bool {
		switch blocker {
		case White:
			return b[i] > 0
		case Black:
			return b[i] < 0
		}
		return false
	}
	run := 0
	for k := 0; k < 48; k++ {
		if occupied(k % 24) {
			run++
			if run >= 6 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}
