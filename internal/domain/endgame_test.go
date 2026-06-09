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

// TestWinner_MarsAndKoks покрывает усиленные победы при проигравшем с
// borneOff == 0:
//
//	#27: марс — нет шашек проигравшего ни в доме победителя, ни на его голове.
//	#28: кокс — есть шашка проигравшего в доме победителя ИЛИ на его голове.
//
// Дом победителя: White → 1..6, Black → 13..18.
// Голова победителя: White → пункт 24, Black → пункт 12.
//
// TDD plan #27, #28.
func TestWinner_MarsAndKoks(t *testing.T) {
	cases := []struct {
		name       string
		setup      func(b *Board)
		borneOff   [2]uint8
		wantWinner Color
		wantKind   WinKind
	}{
		// #27 Mars
		{
			name: "белые победили, 15 чёрных на голове чёрных (вне дома/головы белых) → марс",
			setup: func(b *Board) {
				b[11] = -15
			},
			borneOff:   [2]uint8{15, 0},
			wantWinner: White,
			wantKind:   Mars,
		},
		{
			name: "чёрные победили, 15 белых на голове белых (вне дома/головы чёрных) → марс",
			setup: func(b *Board) {
				b[23] = 15
			},
			borneOff:   [2]uint8{0, 15},
			wantWinner: Black,
			wantKind:   Mars,
		},
		// #28 Koks: в доме победителя
		{
			name: "белые победили, чёрная в доме белых (пункт 1) → кокс",
			setup: func(b *Board) {
				b[0] = -1
				b[11] = -14
			},
			borneOff:   [2]uint8{15, 0},
			wantWinner: White,
			wantKind:   Koks,
		},
		{
			name: "чёрные победили, белая в доме чёрных (пункт 13) → кокс",
			setup: func(b *Board) {
				b[12] = 1
				b[23] = 14
			},
			borneOff:   [2]uint8{0, 15},
			wantWinner: Black,
			wantKind:   Koks,
		},
		// #28 Koks: на голове победителя
		{
			name: "белые победили, чёрная на голове белых (пункт 24) → кокс",
			setup: func(b *Board) {
				b[23] = -1
				b[11] = -14
			},
			borneOff:   [2]uint8{15, 0},
			wantWinner: White,
			wantKind:   Koks,
		},
		{
			name: "чёрные победили, белая на голове чёрных (пункт 12) → кокс",
			setup: func(b *Board) {
				b[11] = 1
				b[23] = 14
			},
			borneOff:   [2]uint8{0, 15},
			wantWinner: Black,
			wantKind:   Koks,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			color, kind, finished := Winner(b, tc.borneOff)
			require.True(t, finished)
			require.Equal(t, tc.wantWinner, color)
			require.Equal(t, tc.wantKind, kind)
		})
	}
}
