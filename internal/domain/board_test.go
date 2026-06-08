package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInitialBoard_StartingPosition проверяет стартовую расстановку
// длинных нард: 15 белых на пункте 24, 15 чёрных на пункте 12, остальные пусты.
//
// TDD plan #1.
func TestInitialBoard_StartingPosition(t *testing.T) {
	b := InitialBoard()

	// b[23] — пункт 24, голова белых.
	require.Equal(t, int8(15), b[23], "ожидалось 15 белых шашек на пункте 24, получено %d", b[23])

	// b[11] — пункт 12, голова чёрных. Знак минус = чёрные.
	require.Equal(t, int8(-15), b[11], "ожидалось 15 чёрных шашек на пункте 12, получено %d", b[11])

	// Остальные пункты — пусты.
	for i := 0; i < 24; i++ {
		if i == 23 || i == 11 {
			continue
		}
		require.Equalf(t, int8(0), b[i], "пункт %d должен быть пуст, получено %d", i+1, b[i])
	}
}
