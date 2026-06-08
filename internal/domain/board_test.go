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

// TestInitialBoard_CheckerCount проверяет инвариант: на стартовой доске
// ровно 15 белых и 15 чёрных шашек.
//
// TDD plan #2.
func TestInitialBoard_CheckerCount(t *testing.T) {
	b := InitialBoard()
	white, black := b.CountByColor()
	require.Equal(t, uint8(15), white, "должно быть 15 белых шашек, получено %d", white)
	require.Equal(t, uint8(15), black, "должно быть 15 чёрных шашек, получено %d", black)
}

// TestNextPoint_SimpleForward проверяет простое движение вперёд для обоих
// цветов с головы — без пересечения границы 1↔24 и без выкида.
//
// TDD plan #3, #4.
func TestNextPoint_SimpleForward(t *testing.T) {
	cases := []struct {
		name  string
		color Color
		from  Point
		pip   uint8
		want  Point
	}{
		{"белый: с головы 24 пипсом 6 → 18", White, 24, 6, 18}, // #3
		{"чёрный: с головы 12 пипсом 6 → 6", Black, 12, 6, 6},  // #4
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, NextPoint(tc.color, tc.from, tc.pip))
		})
	}
}

// TestNextPoint_WrapAndBearOff покрывает два краевых случая:
//   - пересечение границы 1↔24 при движении чёрного из нижней половины;
//   - выкид (возврат 0), когда шаг уводит шашку за дом.
//
// TDD plan #5, #6.
func TestNextPoint_WrapAndBearOff(t *testing.T) {
	cases := []struct {
		name  string
		color Color
		from  Point
		pip   uint8
		want  Point
	}{
		{"чёрный: 5 пипсом 6 пересекает 1→24 → 23", Black, 5, 6, 23}, // #5
		{"белый: 3 пипсом 6 — выкид (переборный)", White, 3, 6, 0},   // #6
		{"белый: 6 пипсом 6 — выкид (точный)", White, 6, 6, 0},       // #6
		{"чёрный: 13 пипсом 1 — выкид из дома", Black, 13, 1, 0},     // #6
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, NextPoint(tc.color, tc.from, tc.pip))
		})
	}
}
