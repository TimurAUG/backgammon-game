package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSixBlockAllowed_BasicViolationAndRelease проверяет базу правила шести:
// финальная позиция нелегальна, если у блокирующего есть 6+ подряд занятых
// пунктов И у соперника нет ни одной шашки в его собственном доме. Как только
// у соперника появляется хотя бы одна шашка в его доме — блок легален.
//
// Для чёрного блок может пересекать границу 24↔1 (направление чёрного —
// 12→1→24→13).
//
// TDD plan #18, #19.
func TestSixBlockAllowed_BasicViolationAndRelease(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(b *Board)
		blocker Color
		want    bool
	}{
		{
			name: "белый блок 7..12, соперник 0 в своём доме (13..18) → нелегально",
			setup: func(b *Board) {
				// Белые подряд на пунктах 7..12.
				b[6], b[7], b[8], b[9], b[10], b[11] = 1, 1, 1, 1, 1, 1
				// Чёрная шашка вне дома чёрных (13..18).
				b[0] = -1
			},
			blocker: White,
			want:    false,
		},
		{
			name: "чёрный блок 22..24 + 1..3 (через границу), соперник 0 в своём доме (1..6) → нелегально",
			setup: func(b *Board) {
				// Чёрные подряд через границу: 22, 23, 24, 1, 2, 3.
				b[21], b[22], b[23] = -1, -1, -1
				b[0], b[1], b[2] = -1, -1, -1
				// Белая шашка вне дома белых (1..6).
				b[18] = 1
			},
			blocker: Black,
			want:    false,
		},
		{
			name: "белый блок 7..12, соперник имеет шашку в своём доме (13..18) → легально",
			setup: func(b *Board) {
				b[6], b[7], b[8], b[9], b[10], b[11] = 1, 1, 1, 1, 1, 1
				// Чёрная на пункте 15 — внутри дома чёрных, запрет снят.
				b[14] = -1
			},
			blocker: White,
			want:    true,
		},
		{
			name: "чёрный блок 22..24 + 1..3, соперник имеет шашку в своём доме (1..6) → легально",
			setup: func(b *Board) {
				b[21], b[22], b[23] = -1, -1, -1
				b[0], b[1], b[2] = -1, -1, -1
				// Белая на пункте 5 — внутри дома белых, запрет снят.
				b[4] = 1
			},
			blocker: Black,
			want:    true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			require.Equal(t, tc.want, SixBlockAllowed(b, tc.blocker))
		})
	}
}
