package domain

// WinKind — тип победы по правилам длинных нард.
//
// Значение 0 зарезервировано как «не победа» (используется в незаконченной
// игре или ошибочном вызове).
type WinKind uint8

const (
	// Oin — победа в 1 очко: проигравший успел выкинуть ≥1 шашку.
	Oin WinKind = iota + 1
	// Mars — победа в 2 очка: проигравший выкинул 0 шашек, и у него нет
	// шашек ни в доме победителя, ни на голове победителя.
	Mars
	// Koks — победа в 3 очка: проигравший выкинул 0 шашек, и есть хотя бы
	// одна его шашка в доме победителя или на голове победителя.
	Koks
)

// Winner определяет, закончена ли партия, и если да — победителя и тип победы.
//
// Закончена: у одного из игроков выкинуто 15 шашек.
// Тип победы:
//   - Oin — проигравший выкинул ≥1.
//   - Mars — проигравший выкинул 0, нет его шашек в доме победителя и на его голове.
//   - Koks — проигравший выкинул 0, есть его шашка в доме победителя или на его голове.
//
// Дом победителя: White → 1..6, Black → 13..18.
// Голова победителя: White → пункт 24, Black → пункт 12.
//
// Для незаконченной игры возвращает нулевые значения (White, 0, false).
//
// TDD plan #25, #26, #27, #28.
func Winner(b Board, borneOff [2]uint8) (Color, WinKind, bool) {
	var winner, loser Color
	switch {
	case borneOff[White] >= 15:
		winner, loser = White, Black
	case borneOff[Black] >= 15:
		winner, loser = Black, White
	default:
		return White, 0, false
	}
	if borneOff[loser] >= 1 {
		return winner, Oin, true
	}
	if loserInWinnerHomeOrHead(b, winner) {
		return winner, Koks, true
	}
	return winner, Mars, true
}

// loserInWinnerHomeOrHead — есть ли хотя бы одна шашка проигравшего в доме
// победителя или на голове победителя.
func loserInWinnerHomeOrHead(b Board, winner Color) bool {
	switch winner {
	case White:
		for i := 0; i <= 5; i++ {
			if b[i] < 0 {
				return true
			}
		}
		return b[23] < 0
	case Black:
		for i := 12; i <= 17; i++ {
			if b[i] > 0 {
				return true
			}
		}
		return b[11] > 0
	}
	return false
}
