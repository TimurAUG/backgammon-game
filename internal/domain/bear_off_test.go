package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsLegalBearOff_GateAndExactPip покрывает базовые ворота правила сброса:
//   - выкид невозможен, пока хоть одна шашка игрока стоит вне его дома;
//   - выкид легален точным пипсом с пункта, равного пипсу
//     (для белого — пункт == пипс; для чёрного — пункт == 19 − пипс).
//
// TDD plan #21, #22.
func TestIsLegalBearOff_GateAndExactPip(t *testing.T) {
	cases := []struct {
		name  string
		setup func(b *Board)
		color Color
		from  Point
		pip   uint8
		want  bool
	}{
		// #21: гейт «все в доме»
		{
			name: "белые: одна шашка вне дома (на пункте 10) → выкид с 6 пипсом 6 запрещён",
			setup: func(b *Board) {
				b[5] = 14 // 14 белых на пункте 6
				b[9] = 1  // 1 белая на пункте 10 (вне дома)
			},
			color: White, from: 6, pip: 6,
			want: false,
		},
		{
			name: "чёрные: одна шашка вне дома (на пункте 20) → выкид с 13 пипсом 6 запрещён",
			setup: func(b *Board) {
				b[12] = -14 // 14 чёрных на пункте 13
				b[19] = -1  // 1 чёрная на пункте 20 (вне дома)
			},
			color: Black, from: 13, pip: 6,
			want: false,
		},
		// #22: точный пипс
		{
			name: "белые: все в доме, точный пипс 3 с пункта 3 → разрешён",
			setup: func(b *Board) {
				b[2] = 1  // 1 белая на пункте 3
				b[5] = 14 // 14 белых на пункте 6 (все в доме)
			},
			color: White, from: 3, pip: 3,
			want: true,
		},
		{
			name: "чёрные: все в доме, точный пипс 3 с пункта 16 → разрешён",
			setup: func(b *Board) {
				b[15] = -1  // 1 чёрная на пункте 16
				b[12] = -14 // 14 чёрных на пункте 13 (все в доме)
			},
			color: Black, from: 16, pip: 3,
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			require.Equal(t, tc.want, IsLegalBearOff(b, tc.color, tc.from, tc.pip))
		})
	}
}
