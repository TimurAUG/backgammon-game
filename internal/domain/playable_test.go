package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCanUsePip_OnlyLargerOfPair демонстрирует правило #29: если из пары
// пипсов используется только бо́льший — игрок обязан использовать его.
// Проверяем через явную пару значений CanUsePip:
// CanUsePip(bigger) == true, CanUsePip(smaller) == false.
//
// TDD plan #29.
func TestCanUsePip_OnlyLargerOfPair(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(b *Board)
		color   Color
		bigger  uint8
		smaller uint8
	}{
		{
			// Белая на 10; чёрная блокирует пункт 5 (цель пипса 5).
			// Пипс 6: 10→4 (пусто) — легален.
			// Пипс 5: 10→5 (блок чёрной) — нелегален.
			name:    "белые: шашка на 10, блок чёрного на 5 → пипс 5 нельзя, 6 можно",
			setup:   func(b *Board) { b[9] = 1; b[4] = -1 },
			color:   White,
			bigger:  6,
			smaller: 5,
		},
		{
			// Чёрная на 11; белая блокирует пункт 6 (цель пипса 5).
			// Пипс 6: 11→5 (пусто) — легален.
			// Пипс 5: 11→6 (блок белой) — нелегален.
			name:    "чёрные: шашка на 11, блок белого на 6 → пипс 5 нельзя, 6 можно",
			setup:   func(b *Board) { b[10] = -1; b[5] = 1 },
			color:   Black,
			bigger:  6,
			smaller: 5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			require.True(t, CanUsePip(b, tc.color, tc.bigger), "бо́льший пипс должен быть доступен")
			require.False(t, CanUsePip(b, tc.color, tc.smaller), "меньший пипс должен быть недоступен")
		})
	}
}

// TestCanUsePip_NoneOfPair_TurnPasses демонстрирует правило #30: если ни
// один пипс из пары не используется — ход переходит сопернику. Проверяем
// через CanUsePip(both) == false.
//
// TDD plan #30.
func TestCanUsePip_NoneOfPair_TurnPasses(t *testing.T) {
	cases := []struct {
		name  string
		setup func(b *Board)
		color Color
		pipA  uint8
		pipB  uint8
	}{
		{
			// Белая на 10; чёрные блокируют 4 и 5 — оба пипса в тупике.
			name:  "белые: шашка на 10, блоки чёрного на 4 и 5 → пипсы 5 и 6 недоступны",
			setup: func(b *Board) { b[9] = 1; b[3] = -1; b[4] = -1 },
			color: White,
			pipA:  6,
			pipB:  5,
		},
		{
			// Чёрная на 11; белые блокируют 5 и 6 — оба пипса в тупике.
			name:  "чёрные: шашка на 11, блоки белого на 5 и 6 → пипсы 5 и 6 недоступны",
			setup: func(b *Board) { b[10] = -1; b[4] = 1; b[5] = 1 },
			color: Black,
			pipA:  6,
			pipB:  5,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b Board
			tc.setup(&b)
			require.False(t, CanUsePip(b, tc.color, tc.pipA))
			require.False(t, CanUsePip(b, tc.color, tc.pipB))
		})
	}
}

// TestLegalMoves покрывает базовую сборку списка легальных одиночных шагов:
// шаги и выкиды для каждого пипса из Remaining, дедупликация. Правило головы
// и правило шести на этом уровне не учитываются — это уровни выше.
//
// Подготовка к #34 (LEGAL_MOVES в WS-протоколе).
func TestLegalMoves(t *testing.T) {
	cases := []struct {
		name  string
		setup func() GameState
		want  []Move
	}{
		{
			name: "initial board, белые [5,3]: 24→19 пипсом 5 и 24→21 пипсом 3",
			setup: func() GameState {
				return GameState{
					Board: InitialBoard(),
					Turn:  White,
					Dice:  NewDice(5, 3),
				}
			},
			want: []Move{
				{From: 24, To: 19, Pip: 5},
				{From: 24, To: 21, Pip: 3},
			},
		},
		{
			name: "белая на 10 + блоки на 4 и 5, [5,6]: пустой список",
			setup: func() GameState {
				var b Board
				b[9] = 1
				b[3] = -1
				b[4] = -1
				return GameState{Board: b, Turn: White, Dice: NewDice(5, 6)}
			},
			want: []Move{},
		},
		{
			name: "одна белая на 3 (все в доме), [3,4]: выкид с 3 точным и переборным",
			setup: func() GameState {
				var b Board
				b[2] = 1
				return GameState{Board: b, Turn: White, Dice: NewDice(3, 4)}
			},
			want: []Move{
				{From: 3, To: 0, Pip: 3},
				{From: 3, To: 0, Pip: 4},
			},
		},
		{
			name: "правило головы: initial board, белые [5,3], HeadConsumed=1 → 0 ходов (только голова)",
			setup: func() GameState {
				return GameState{
					Board:        InitialBoard(),
					Turn:         White,
					Dice:         NewDice(5, 3),
					HeadConsumed: [2]uint8{1, 0},
					IsFirstMove:  [2]bool{true, true},
				}
			},
			want: []Move{},
		},
		{
			name: "lookahead: ход 13→12 неизбежно создаёт финальный блок 6 (соперник 0 в доме) — отфильтрован",
			setup: func() GameState {
				var b Board
				b[6], b[7], b[8], b[9], b[10] = 1, 1, 1, 1, 1 // белые 7..11
				b[12] = 1                                     // белая на 13
				return GameState{
					Board:       b,
					Turn:        White,
					Dice:        Dice{Remaining: []uint8{1}},
					IsFirstMove: [2]bool{false, false},
				}
			},
			// 13→12 после Apply: блок 7..12 = 6 подряд. Remaining пуст → final.
			// Чёрных в 13..18 нет → SixBlockAllowed=false → нелегально.
			// Остальные ходы рассеивают блок.
			want: []Move{
				{From: 7, To: 6, Pip: 1},
				{From: 8, To: 7, Pip: 1},
				{From: 9, To: 8, Pip: 1},
				{From: 10, To: 9, Pip: 1},
				{From: 11, To: 10, Pip: 1},
			},
		},
		{
			name: "исключение головы 6:6 на первом ходу: 14 белых на 24 + 1 на 18, HeadConsumed=1, [6,6,6,6] → 24→18 и 18→12",
			setup: func() GameState {
				var b Board
				b[23] = 14
				b[17] = 1
				return GameState{
					Board:        b,
					Turn:         White,
					Dice:         NewDice(6, 6),
					HeadConsumed: [2]uint8{1, 0},
					IsFirstMove:  [2]bool{true, true},
				}
			},
			want: []Move{
				{From: 24, To: 18, Pip: 6},
				{From: 18, To: 12, Pip: 6},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.setup()
			moves := LegalMoves(s)
			require.ElementsMatch(t, tc.want, moves)
		})
	}
}

// TestLegalMoves_FirstMoveDoubleSix_PlayableFromHead — регрессия бага: на самом
// первом ходу партии дубль 6:6 ошибочно пропускался как «нет ходов», хотя с
// головы можно снять две шашки (исключение головы). С чистого старта все 15
// шашек на голове, единственный кандидат — снять с головы (белые 24→18,
// чёрные 12→6). Дальше шестёрки упираются: 18→12 (соответственно 6→24) закрыты
// головой соперника, а голова после двух снятий — правилом головы. Это
// легальный тупик, ход НЕ должен отбраковываться целиком.
//
// Причина бага: canReachLegalFinal определял терминал через CanUsePip, который
// намеренно игнорирует правило головы → тупик «осталась только голова, но она
// закрыта правилом» считался «ещё есть ход» и SixBlockAllowed в нём не
// проверялся → вся ветка возвращала false → LegalMoves пуст → авто-пропуск.
func TestLegalMoves_FirstMoveDoubleSix_PlayableFromHead(t *testing.T) {
	cases := []struct {
		name string
		turn Color
		want []Move
	}{
		{"белые: 24→18", White, []Move{{From: 24, To: 18, Pip: 6}}},
		{"чёрные: 12→6", Black, []Move{{From: 12, To: 6, Pip: 6}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := GameState{
				Board:        InitialBoard(),
				Turn:         tc.turn,
				Dice:         NewDice(6, 6),
				HeadConsumed: [2]uint8{0, 0},
				IsFirstMove:  [2]bool{true, true},
			}
			moves := LegalMoves(s)
			require.ElementsMatch(t, tc.want, moves,
				"первый ход 6:6 должен быть играбелен с головы, а не пропущен как «нет ходов»")
		})
	}
}
