package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWinner_VictoryAndOin покрывает базу окончания партии:
//
//	#25: игра завершена, когда у победителя borneOff == 15.
//	#26: тип победы — оин, если проигравший успел выкинуть хотя бы одну
//	     шашку.
//
// Марс/кокс (проигравший выкинул 0) покрываются следующим циклом (#27-#28).
//
// Для незаконченной игры функция возвращает нулевые значения (White, 0, false).
//
// TDD plan #25, #26.
func TestWinner_VictoryAndOin(t *testing.T) {
	cases := []struct {
		name         string
		setup        func(b *Board)
		borneOff     [2]uint8
		wantFinished bool
		wantWinner   Color
		wantKind     WinKind
	}{
		{
			name: "никто не выкинул 15 → игра продолжается",
			setup: func(b *Board) {
				b[5] = 8    // 8 белых в доме
				b[11] = -15 // 15 чёрных на голове
			},
			borneOff:     [2]uint8{7, 0},
			wantFinished: false,
			wantWinner:   White, // нулевое значение Color
			wantKind:     0,
		},
		{
			name: "белые выкинули 15, чёрные выкинули 3 → оин для белых",
			setup: func(b *Board) {
				b[11] = -12 // 12 чёрных на голове (15 − 3 выкинутых)
			},
			borneOff:     [2]uint8{15, 3},
			wantFinished: true,
			wantWinner:   White,
			wantKind:     Oin,
		},
		{
			name: "чёрные выкинули 15, белые выкинули 7 → оин для чёрных",
			setup: func(b *Board) {
				b[23] = 8 // 8 белых на 24 (15 − 7 выкинутых)
			},
			borneOff:     [2]uint8{7, 15},
			wantFinished: true,
			wantWinner:   Black,
			wantKind:     Oin,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			color, kind, finished := Winner(b, tc.borneOff)
			require.Equal(t, tc.wantFinished, finished)
			require.Equal(t, tc.wantWinner, color)
			require.Equal(t, tc.wantKind, kind)
		})
	}
}
