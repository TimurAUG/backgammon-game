package domain

// HeadMoveAllowed возвращает true, если игроку разрешено снять ещё одну
// шашку с головы в текущем ходу.
//
// Параметры:
//   - consumedThisTurn: сколько шашек с головы уже снято в этом ходу;
//   - dice: текущие кубики (поле IsDouble и значение A используются
//     для исключения дублей);
//   - isFirstMove: первый ли это ход партии для этого игрока.
//
// Правило длинных нард:
//   - 0 уже снято → разрешено всегда;
//   - 1 уже снято → разрешено только при исключении: первый ход партии и
//     дубль из {6:6, 4:4, 3:3};
//   - 2+ уже снято → запрещено.
//
// TDD plan #13–#17.
func HeadMoveAllowed(consumedThisTurn uint8, dice Dice, isFirstMove bool) bool {
	if consumedThisTurn == 0 {
		return true
	}
	// Исключение для #14, #15: на первом ходу при любом дубле — можно вторую.
	// Сузим до {6, 4, 3} в #16.
	if consumedThisTurn == 1 && isFirstMove && dice.IsDouble {
		return true
	}
	return false
}
