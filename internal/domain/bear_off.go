package domain

// AllInHome возвращает true, если все шашки игрока c, находящиеся на доске,
// стоят внутри его дома.
//
// Дом белых: пункты 1..6 (индексы 0..5).
// Дом чёрных: пункты 13..18 (индексы 12..17).
//
// Функция смотрит только на доску — выкинутые шашки игнорируются (они уже
// «дома» по определению).
//
// TDD plan #21.
func AllInHome(b Board, c Color) bool {
	switch c {
	case White:
		for i := 6; i < 24; i++ {
			if b[i] > 0 {
				return false
			}
		}
		return true
	case Black:
		for i := 0; i < 12; i++ {
			if b[i] < 0 {
				return false
			}
		}
		for i := 18; i < 24; i++ {
			if b[i] < 0 {
				return false
			}
		}
		return true
	}
	return false
}

// IsLegalBearOff проверяет легальность выкида шашки игроком c с пункта from
// при пипсе pip.
//
// Условия легального выкида:
//   - все 15 шашек игрока c находятся в его доме (AllInHome);
//   - на пункте from стоит шашка игрока c;
//   - пипс точный: from совпадает с bearOffPointForPip(c, pip).
//
// Переборный пипс пока не поддерживается — он будет добавлен следующим
// циклом TDD (#23).
//
// TDD plan #21, #22.
func IsLegalBearOff(b Board, c Color, from Point, pip uint8) bool {
	if !AllInHome(b, c) {
		return false
	}
	if from < 1 || from > 24 {
		return false
	}
	cell := b[from-1]
	switch c {
	case White:
		if cell <= 0 {
			return false
		}
	case Black:
		if cell >= 0 {
			return false
		}
	default:
		return false
	}
	return bearOffPointForPip(c, pip) == from
}

// bearOffPointForPip возвращает пункт, с которого данный пипс выкидывает
// шашку точным образом.
//
// Белый: путь 6→1→выкид, поэтому пипс N точно выкидывает с пункта N.
// Чёрный: путь 18→13→выкид, поэтому пипс N точно выкидывает с пункта 19−N
// (пипс 1 → пункт 18, пипс 6 → пункт 13).
func bearOffPointForPip(c Color, pip uint8) Point {
	switch c {
	case White:
		return Point(pip)
	case Black:
		return Point(19 - int(pip))
	}
	return 0
}
