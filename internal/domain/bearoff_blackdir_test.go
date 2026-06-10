package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Проверка направления сброса чёрных. Все чёрные дома (13-18): 1 на пункте 13,
// 14 на пункте 18. Пип 6 ТОЧНО выкидывает с пункта 18 (18 нужен ровно 6:
// 18→…→13→выкид), а с пункта 13 пип 6 переборный, но на 18 (дальше от выкида)
// стоит шашка → выкид с 13 запрещён. Ожидаем выкид ТОЛЬКО с 18.
func TestLegalMoves_BlackBearOff_Pip6_BearsOffFarthest(t *testing.T) {
	var b Board
	b[12] = -1  // пункт 13: 1 чёрная
	b[17] = -14 // пункт 18: 14 чёрных
	b[0] = 15   // пункт 1: белые, вне дела
	s := GameState{
		Board:       b,
		Turn:        Black,
		Status:      StatusWaitingForMove,
		Dice:        Dice{A: 6, B: 6, Remaining: []uint8{6}},
		IsFirstMove: [2]bool{false, false},
	}

	moves := LegalMoves(s)

	var bearOffs []Point
	for _, m := range moves {
		if m.To == 0 {
			bearOffs = append(bearOffs, m.From)
		}
	}
	require.Equal(t, []Point{18}, bearOffs,
		"пип 6 у чёрных должен сбрасывать с пункта 18 (точный), не с 13")
}
