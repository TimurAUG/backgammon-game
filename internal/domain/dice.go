package domain

// Dice — состояние кубиков на текущий ход.
//
// A, B: значения двух кубиков (1..6).
// IsDouble: true, если A == B (дубль).
// Remaining: пипсы, которые ещё не использованы в этом ходу.
//   - При обычном броске: [A, B] (2 элемента).
//   - При дубле: [A, A, A, A] (4 элемента).
//
// TDD plan #11.
type Dice struct {
	A, B      uint8
	IsDouble  bool
	Remaining []uint8
}

// NewDice создаёт Dice из значений двух кубиков. При дубле (a == b) заводит
// 4 одинаковых пипса в Remaining, иначе — два пипса [a, b].
//
// TDD plan #11.
func NewDice(a, b uint8) Dice {
	isDouble := a == b
	var remaining []uint8
	if isDouble {
		remaining = []uint8{a, a, a, a}
	} else {
		remaining = []uint8{a, b}
	}
	return Dice{A: a, B: b, IsDouble: isDouble, Remaining: remaining}
}

// Use возвращает новый Dice, в котором один пипс заданного значения убран
// из Remaining. Если такого пипса в Remaining нет — возвращает d без
// изменений (вызывающий обязан сам проверить доступность пипса).
//
// Dice иммутабелен: исходный d не меняется.
//
// TDD plan #12.
func (d Dice) Use(pip uint8) Dice {
	for i, p := range d.Remaining {
		if p != pip {
			continue
		}
		newRemaining := make([]uint8, 0, len(d.Remaining)-1)
		newRemaining = append(newRemaining, d.Remaining[:i]...)
		newRemaining = append(newRemaining, d.Remaining[i+1:]...)
		return Dice{
			A:         d.A,
			B:         d.B,
			IsDouble:  d.IsDouble,
			Remaining: newRemaining,
		}
	}
	return d
}
