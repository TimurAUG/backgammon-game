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
//   - либо пипс точный (from == bearOffPointForPip(c, pip)),
//   - либо пипс переборный: from «ближе к выкиду» чем точный пункт пипса,
//     и между ними нет занятых пунктов цвета c (т.е. from — самый дальний
//     от выкида занятый пункт с меньшим пипсом).
//
// TDD plan #21, #22, #23.
func IsLegalBearOff(b Board, c Color, from Point, pip uint8) bool {
	if !AllInHome(b, c) {
		return false
	}
	if from < 1 || from > 24 {
		return false
	}
	switch c {
	case White:
		if b[from-1] <= 0 {
			return false
		}
	case Black:
		if b[from-1] >= 0 {
			return false
		}
	default:
		return false
	}
	exact := bearOffPointForPip(c, pip)
	if from == exact {
		return true
	}
	// Переборный: на пунктах от from (исключительно) до exact (включительно)
	// по направлению цвета не должно быть шашек этого цвета. Если такая шашка
	// есть — она «дальше» from по пути, и пипс должен использоваться её.
	switch c {
	case White:
		// from < exact, путь дальше от выкида = больший номер.
		if from >= exact {
			return false
		}
		for p := from + 1; p <= exact; p++ {
			if b[p-1] > 0 {
				return false
			}
		}
		return true
	case Black:
		// Чёрные тоже выкидывают в сторону МЕНЬШИХ номеров (18→13→выкид),
		// поэтому логика зеркальна белым: from ближе к выкиду (< exact),
		// и между from и exact не должно быть шашек цвета (они дальше от
		// выкида → ими и надо ходить переборным пипсом).
		if from >= exact {
			return false
		}
		for p := from + 1; p <= exact; p++ {
			if b[p-1] < 0 {
				return false
			}
		}
		return true
	}
	return false
}

// bearOffPointForPip возвращает пункт, с которого данный пипс выкидывает
// шашку точным образом.
//
// Белый: путь 6→1→выкид, поэтому пипс N точно выкидывает с пункта N.
// Чёрный: путь 18→13→выкид (выкид в сторону меньших номеров), пункт N
// требует ровно пипс N−12, поэтому пипс N точно выкидывает с пункта 12+N
// (пипс 1 → пункт 13, пипс 6 → пункт 18). Согласовано с NextPoint.
func bearOffPointForPip(c Color, pip uint8) Point {
	switch c {
	case White:
		return Point(pip)
	case Black:
		return Point(12 + int(pip))
	}
	return 0
}
