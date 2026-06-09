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
