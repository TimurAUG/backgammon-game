package domain

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewDice_PipCount проверяет: дубль даёт 4 пипса, обычный бросок — 2.
// Поля A, B и IsDouble тоже валидируются.
//
// TDD plan #11.
func TestNewDice_PipCount(t *testing.T) {
	cases := []struct {
		name    string
		a, b    uint8
		wantLen int
		wantDbl bool
	}{
		{"дубль 6:6 → 4 пипса", 6, 6, 4, true},
		{"дубль 3:3 → 4 пипса", 3, 3, 4, true},
		{"обычный 3:5 → 2 пипса", 3, 5, 2, false},
		{"обычный 1:2 → 2 пипса", 1, 2, 2, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDice(tc.a, tc.b)
			require.Equal(t, tc.a, d.A, "поле A")
			require.Equal(t, tc.b, d.B, "поле B")
			require.Equal(t, tc.wantDbl, d.IsDouble, "поле IsDouble")
			require.Equal(t, tc.wantLen, len(d.Remaining), "len(Remaining)")
			for i, p := range d.Remaining {
				// Для дубля все пипсы равны A.
				// Для обычного броска — [A, B] в этом порядке.
				if tc.wantDbl {
					require.Equalf(t, tc.a, p, "Remaining[%d] на дубле должен быть %d", i, tc.a)
				}
			}
		})
	}
}

// TestDice_Use_ConsumesAllPipsOfDouble проверяет, что последовательные вызовы
// Use на дубле 6:6 уменьшают Remaining с 4 до 0 за 4 шага.
//
// TDD plan #12.
func TestDice_Use_ConsumesAllPipsOfDouble(t *testing.T) {
	d := NewDice(6, 6)
	require.Equal(t, 4, len(d.Remaining), "стартовый дубль 6:6 — 4 пипса")

	for want := 3; want >= 0; want-- {
		d = d.Use(6)
		require.Equalf(t, want, len(d.Remaining),
			"после очередного Use(6) ожидалось %d пипсов, получено %d", want, len(d.Remaining))
	}
}

// TestRollDice покрывает RollDice(r io.Reader): обычный бросок, дубль,
// rejection sampling для устранения modulo-bias (байты ≥252 отбрасываются,
// поскольку 256 не делится на 6 нацело).
//
// Подготовка к #34b (ROLL_FOR_FIRST/ROLL на уровне транспорта).
func TestRollDice(t *testing.T) {
	cases := []struct {
		name          string
		input         []byte
		wantA, wantB  uint8
		wantDouble    bool
		wantRemaining []uint8
	}{
		{
			name:          "байты 0,1 → бросок 1:2",
			input:         []byte{0, 1},
			wantA:         1,
			wantB:         2,
			wantDouble:    false,
			wantRemaining: []uint8{1, 2},
		},
		{
			name:          "байты 5,11 → дубль 6:6, 4 пипса",
			input:         []byte{5, 11},
			wantA:         6,
			wantB:         6,
			wantDouble:    true,
			wantRemaining: []uint8{6, 6, 6, 6},
		},
		{
			name:          "rejection: байты 252,253 отбрасываются, затем 0,1 → 1:2",
			input:         []byte{252, 253, 0, 1},
			wantA:         1,
			wantB:         2,
			wantDouble:    false,
			wantRemaining: []uint8{1, 2},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := RollDice(bytes.NewReader(tc.input))
			require.NoError(t, err)
			require.Equal(t, tc.wantA, d.A)
			require.Equal(t, tc.wantB, d.B)
			require.Equal(t, tc.wantDouble, d.IsDouble)
			require.Equal(t, tc.wantRemaining, d.Remaining)
		})
	}
}
