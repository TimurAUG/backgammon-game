package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// reachKey описывает цель «человекочитаемо» для сравнения, не привязываясь к
// конкретному порядку пипсов в составной цепочке: куда пришли (to), сколько
// кубиков потратили (dice) и суммарная дальность (dist == сумма пипсов).
type reachKey struct {
	to   Point
	dice int
	dist int
}

func summarizeReach(t *testing.T, rs []Reach) []reachKey {
	t.Helper()
	out := make([]reachKey, 0, len(rs))
	for _, r := range rs {
		require.NotEmpty(t, r.Path, "Path не должен быть пустым")
		require.Len(t, r.Pips, len(r.Path), "длины Path и Pips должны совпадать")
		require.NotZero(t, r.From, "From должен быть проставлен")
		sum := 0
		for _, p := range r.Pips {
			sum += int(p)
		}
		out = append(out, reachKey{to: r.Path[len(r.Path)-1], dice: len(r.Pips), dist: sum})
	}
	return out
}

// TestReachableTargets_NonDouble_SingleCheckerChain: для не-дубля одна шашка с
// открытой дорожкой достижима одним кубиком (две одиночные цели) и двумя
// кубиками (составная цель). Белая на 13, кубики 2 и 4, пункты 11/9/7 пусты:
// 13→11 (пипс 2), 13→9 (пипс 4) и 13→7 двумя кубиками.
//
// FRONTEND_SPEC бэкенд-подложка, SPEC #50.
func TestReachableTargets_NonDouble_SingleCheckerChain(t *testing.T) {
	var b Board
	b[12] = 1 // белая на 13
	s := GameState{Board: b, Turn: White, Dice: NewDice(2, 4)}

	got := summarizeReach(t, ReachableTargets(s))

	require.ElementsMatch(t, []reachKey{
		{to: 11, dice: 1, dist: 2},
		{to: 9, dice: 1, dist: 4},
		{to: 7, dice: 2, dist: 6},
	}, got)
}

// TestReachableTargets_Double_Ladder: дубль даёт «лесенку» одной шашкой на все
// четыре кубика. Белая на 13, кубики 3:3, дорожка 10/7/4/1 пуста →
// 13→10 (1 куб), →7 (2), →4 (3), →1 (4). Выкид (To==0) в reach не попадает.
//
// SPEC #50.
func TestReachableTargets_Double_Ladder(t *testing.T) {
	var b Board
	b[12] = 1 // белая на 13
	s := GameState{Board: b, Turn: White, Dice: NewDice(3, 3)}

	got := summarizeReach(t, ReachableTargets(s))

	require.ElementsMatch(t, []reachKey{
		{to: 10, dice: 1, dist: 3},
		{to: 7, dice: 2, dist: 6},
		{to: 4, dice: 3, dist: 9},
		{to: 1, dice: 4, dist: 12},
	}, got)
}

// TestReachableTargets_BlockedIntermediate_OtherOrderReaches: если один порядок
// кубиков упирается в блок соперника, составная цель всё равно достижима другим
// порядком, а заблокированная одиночная цель не показывается. Белая на 13,
// кубики 2 и 4, чёрный блок на 11: пипс 2 (13→11) нелегален, но 13→9 (пипс 4)
// и 9→7 (пипс 2) дают составную цель 7. Цель 11 в reach отсутствует.
//
// SPEC #50.
func TestReachableTargets_BlockedIntermediate_OtherOrderReaches(t *testing.T) {
	var b Board
	b[12] = 1  // белая на 13
	b[10] = -2 // чёрный блок на 11
	s := GameState{Board: b, Turn: White, Dice: NewDice(2, 4)}

	got := summarizeReach(t, ReachableTargets(s))

	require.ElementsMatch(t, []reachKey{
		{to: 9, dice: 1, dist: 4},
		{to: 7, dice: 2, dist: 6},
	}, got)
}
