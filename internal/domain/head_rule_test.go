package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHeadMoveAllowed_OnePerRegularTurn проверяет базовое ограничение
// правила головы: за обычный ход (без исключений) с головы можно снять
// только одну шашку.
//
// TDD plan #13.
func TestHeadMoveAllowed_OnePerRegularTurn(t *testing.T) {
	cases := []struct {
		name     string
		consumed uint8
		dice     Dice
		first    bool
		want     bool
	}{
		{"0 уже снято, обычный 5:3 — можно", 0, NewDice(5, 3), true, true},
		{"1 уже снято, обычный 5:3 — нельзя", 1, NewDice(5, 3), true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, HeadMoveAllowed(tc.consumed, tc.dice, tc.first))
		})
	}
}
