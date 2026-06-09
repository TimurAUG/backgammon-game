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

// TestHeadMoveAllowed_FirstMoveDoubleException проверяет исключение
// правила головы: на первом ходу партии при дубле 6:6, 4:4 или 3:3
// разрешено снять с головы вторую шашку.
//
// TDD plan #14, #15.
func TestHeadMoveAllowed_FirstMoveDoubleException(t *testing.T) {
	cases := []struct {
		name     string
		consumed uint8
		dice     Dice
		first    bool
		want     bool
	}{
		{"первый ход, дубль 6:6, 1 снято → можно", 1, NewDice(6, 6), true, true}, // #14
		{"первый ход, дубль 4:4, 1 снято → можно", 1, NewDice(4, 4), true, true}, // #15
		{"первый ход, дубль 3:3, 1 снято → можно", 1, NewDice(3, 3), true, true}, // #15
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, HeadMoveAllowed(tc.consumed, tc.dice, tc.first))
		})
	}
}
