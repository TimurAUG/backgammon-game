package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApply_UpdatesStateOnLegalMove проверяет, что Apply на легальном шаге
// или выкиде корректно обновляет Board, BorneOff и Dice.Remaining (#31).
//
// TDD plan #31.
func TestApply_UpdatesStateOnLegalMove(t *testing.T) {
	cases := []struct {
		name   string
		setup  func() GameState
		move   Move
		assert func(t *testing.T, ns GameState)
	}{
		{
			name: "обычный шаг белой 10→5 пипсом 5: Board обновлена, Remaining уменьшен",
			setup: func() GameState {
				var b Board
				b[9] = 1
				b[11] = -15
				return GameState{
					Board:    b,
					Turn:     White,
					Dice:     NewDice(5, 3),
					BorneOff: [2]uint8{0, 0},
				}
			},
			move: Move{From: 10, To: 5, Pip: 5},
			assert: func(t *testing.T, ns GameState) {
				require.Equal(t, int8(0), ns.Board[9], "пункт 10 должен опустеть")
				require.Equal(t, int8(1), ns.Board[4], "пункт 5 должен получить белую")
				require.Equal(t, [2]uint8{0, 0}, ns.BorneOff)
				require.Equal(t, []uint8{3}, ns.Dice.Remaining)
			},
		},
		{
			name: "выкид белой с 3 пипсом 3: пункт пуст, BorneOff++, Remaining уменьшен",
			setup: func() GameState {
				var b Board
				b[2] = 1
				return GameState{
					Board:    b,
					Turn:     White,
					Dice:     NewDice(3, 4),
					BorneOff: [2]uint8{14, 0},
				}
			},
			move: Move{From: 3, To: 0, Pip: 3},
			assert: func(t *testing.T, ns GameState) {
				require.Equal(t, int8(0), ns.Board[2], "пункт 3 должен опустеть")
				require.Equal(t, uint8(15), ns.BorneOff[White])
				require.Equal(t, []uint8{4}, ns.Dice.Remaining)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.setup()
			ns, err := Apply(s, tc.move)
			require.NoError(t, err)
			tc.assert(t, ns)
		})
	}
}

// TestApply_IncrementsHeadConsumed — при ходе с головы HeadConsumed[Turn]
// увеличивается на 1. Голова белых — пункт 24, голова чёрных — пункт 12.
//
// TDD plan: подготовка к #34d (учёт правила головы в LEGAL_MOVES).
func TestApply_IncrementsHeadConsumed(t *testing.T) {
	t.Run("белый ходит с 24 → HeadConsumed[White]=1", func(t *testing.T) {
		var b Board
		b[23] = 15
		b[11] = -15
		s := GameState{
			Board:       b,
			Turn:        White,
			Dice:        NewDice(5, 3),
			IsFirstMove: [2]bool{true, true},
		}
		ns, err := Apply(s, Move{From: 24, To: 19, Pip: 5})
		require.NoError(t, err)
		require.Equal(t, uint8(1), ns.HeadConsumed[White])
		require.Equal(t, uint8(0), ns.HeadConsumed[Black])
	})
	t.Run("чёрный ходит с 12 → HeadConsumed[Black]=1", func(t *testing.T) {
		var b Board
		b[11] = -15
		b[23] = 15
		s := GameState{
			Board:       b,
			Turn:        Black,
			Dice:        NewDice(5, 3),
			IsFirstMove: [2]bool{true, true},
		}
		ns, err := Apply(s, Move{From: 12, To: 7, Pip: 5})
		require.NoError(t, err)
		require.Equal(t, uint8(0), ns.HeadConsumed[White])
		require.Equal(t, uint8(1), ns.HeadConsumed[Black])
	})
	t.Run("ход не с головы → HeadConsumed не меняется", func(t *testing.T) {
		var b Board
		b[9] = 1 // белая на 10 — не голова
		b[11] = -15
		s := GameState{
			Board:        b,
			Turn:         White,
			Dice:         NewDice(5, 3),
			HeadConsumed: [2]uint8{0, 0},
		}
		ns, err := Apply(s, Move{From: 10, To: 5, Pip: 5})
		require.NoError(t, err)
		require.Equal(t, uint8(0), ns.HeadConsumed[White])
	})
}

// TestApply_ReturnsErrorOnIllegalMove проверяет, что Apply возвращает ошибку
// при нелегальном ходе (целевой пункт занят соперником) и при попытке
// использовать пипс, которого нет в Remaining.
//
// TDD plan #31.
func TestApply_ReturnsErrorOnIllegalMove(t *testing.T) {
	cases := []struct {
		name  string
		setup func() GameState
		move  Move
	}{
		{
			name: "целевой пункт занят чёрной → ошибка",
			setup: func() GameState {
				var b Board
				b[9] = 1
				b[4] = -1
				return GameState{
					Board: b, Turn: White, Dice: NewDice(5, 3),
				}
			},
			move: Move{From: 10, To: 5, Pip: 5},
		},
		{
			name: "пипс отсутствует в Remaining → ошибка",
			setup: func() GameState {
				var b Board
				b[9] = 1
				return GameState{
					Board: b, Turn: White, Dice: NewDice(5, 3),
				}
			},
			move: Move{From: 10, To: 6, Pip: 4}, // пипс 4, в Remaining только [5,3]
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.setup()
			_, err := Apply(s, tc.move)
			require.Error(t, err)
		})
	}
}

// TestIsTurnComplete покрывает условия завершения хода (#32): пустой Remaining,
// тупик (никаких легальных ходов оставшимися пипсами), и обычная ситуация
// (есть оставшиеся пипсы и есть ходы).
//
// TDD plan #32.
func TestIsTurnComplete(t *testing.T) {
	cases := []struct {
		name  string
		setup func() GameState
		want  bool
	}{
		{
			name: "Remaining пуст → ход завершён",
			setup: func() GameState {
				return GameState{
					Board: Board{},
					Turn:  White,
					Dice:  Dice{Remaining: []uint8{}},
				}
			},
			want: true,
		},
		{
			name: "Remaining не пуст, но все ходы в тупике → ход завершён",
			setup: func() GameState {
				var b Board
				b[9] = 1
				b[3] = -1
				b[4] = -1
				return GameState{
					Board: b, Turn: White, Dice: NewDice(5, 6),
				}
			},
			want: true,
		},
		{
			name: "Remaining не пуст, есть легальные ходы → ход не завершён",
			setup: func() GameState {
				var b Board
				b[9] = 1
				return GameState{
					Board: b, Turn: White, Dice: NewDice(5, 6),
				}
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.setup()
			require.Equal(t, tc.want, IsTurnComplete(s))
		})
	}
}

// TestIsTurnComplete_HeadBlockedLeftoverPip_Complete — регрессия: если
// оставшиеся пипсы «используемы» лишь с головы, но правило головы это
// запрещает (а других ходов нет), ход завершён. Иначе игрок застревает: ходить
// нечем, а END_TURN отклоняется (ErrMustUsePip), т.к. IsTurnComplete=false.
//
// Сцена первого хода: белые дублем 6:6 сняли две шашки с головы на 18
// (исключение), но 18→12 закрыт головой чёрных, а голова закрыта правилом.
// Тот же корень, что у бага LegalMoves: CanUsePip игнорирует правило головы.
func TestIsTurnComplete_HeadBlockedLeftoverPip_Complete(t *testing.T) {
	var b Board
	b[23] = 13  // 13 белых на голове (24)
	b[17] = 2   // 2 белых на 18 (сняты с головы этим ходом)
	b[11] = -15 // 15 чёрных на голове (12) — закрывают 18→12
	s := GameState{
		Board:        b,
		Turn:         White,
		Dice:         Dice{A: 6, B: 6, IsDouble: true, Remaining: []uint8{6, 6}},
		HeadConsumed: [2]uint8{2, 0},
		IsFirstMove:  [2]bool{true, true},
	}
	require.True(t, IsTurnComplete(s),
		"ходить нечем (голова закрыта правилом, 18→12 закрыт соперником) → ход завершён")
}
