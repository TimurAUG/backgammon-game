package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsLegalBearOff_GateAndExactPip покрывает базовые ворота правила сброса:
//   - выкид невозможен, пока хоть одна шашка игрока стоит вне его дома;
//   - выкид легален точным пипсом с пункта, равного пипсу
//     (для белого — пункт == пипс; для чёрного — пункт == 12 + пипс).
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
			name: "чёрные: все в доме, точный пипс 3 с пункта 15 → разрешён",
			setup: func(b *Board) {
				b[14] = -1  // 1 чёрная на пункте 15 (15−12 = пипс 3)
				b[12] = -14 // 14 чёрных на пункте 13 (все в доме)
			},
			color: Black, from: 15, pip: 3,
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

// TestIsLegalBearOff_OverpipFromFarthest проверяет переборный пипс: если
// в точном пункте нет шашек, выкид возможен с самого дальнего от выкида
// занятого пункта меньшего пипса (по направлению цвета).
//
// «Самый дальний» — это пункт, с которого до выкида ещё дальше всех. И белые,
// и чёрные выкидывают в сторону МЕНЬШИХ номеров, поэтому дальше — БО́ЛЬШИЙ
// номер пункта дома: у белых 1..6 (пункт 5 дальше 1), у чёрных 13..18
// (пункт 17 дальше пункта 14).
//
// TDD plan #23.
func TestIsLegalBearOff_OverpipFromFarthest(t *testing.T) {
	cases := []struct {
		name  string
		setup func(b *Board)
		color Color
		from  Point
		pip   uint8
		want  bool
	}{
		{
			name: "белые: шашки на 1,3,5, пипс 6 (на 6 пусто) → выкид с 5 легален",
			setup: func(b *Board) {
				b[0], b[2], b[4] = 1, 1, 1
			},
			color: White, from: 5, pip: 6,
			want: true,
		},
		{
			name: "белые: шашки на 1,3,5, пипс 6 → выкид с 3 нелегален (на 5 есть дальняя)",
			setup: func(b *Board) {
				b[0], b[2], b[4] = 1, 1, 1
			},
			color: White, from: 3, pip: 6,
			want: false,
		},
		{
			name: "чёрные: шашки на 14,17, пипс 6 (на 18 пусто) → выкид с 17 легален (дальний)",
			setup: func(b *Board) {
				b[13], b[16] = -1, -1
			},
			color: Black, from: 17, pip: 6,
			want: true,
		},
		{
			name: "чёрные: шашки на 14,17, пипс 6 → выкид с 14 нелегален (на 17 есть дальняя)",
			setup: func(b *Board) {
				b[13], b[16] = -1, -1
			},
			color: Black, from: 14, pip: 6,
			want: false,
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

// TestIsLegalStep_InHomeMovement фиксирует, что движение внутри дома по
// обычному пипсу — это легальный шаг. Специальной логики не требуется:
// IsLegalStep уже корректно обрабатывает такие шаги (направление + цвет
// целевой клетки). Тест документирует это явно.
//
// TDD plan #24.
func TestIsLegalStep_InHomeMovement(t *testing.T) {
	cases := []struct {
		name  string
		setup func(b *Board)
		color Color
		move  Move
	}{
		{
			name: "белые: с 6 на 4 пипсом 2, целевой пункт пуст → легально",
			setup: func(b *Board) {
				b[5] = 1
			},
			color: White,
			move:  Move{From: 6, To: 4, Pip: 2},
		},
		{
			name: "чёрные: с 17 на 15 пипсом 2, целевой пункт пуст → легально",
			setup: func(b *Board) {
				b[16] = -1
			},
			color: Black,
			move:  Move{From: 17, To: 15, Pip: 2},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			require.True(t, IsLegalStep(b, tc.color, tc.move))
		})
	}
}
