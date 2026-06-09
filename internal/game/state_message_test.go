package game_test

import (
	"encoding/json"
	"testing"

	"github.com/TimurAUG/backgammon-game/internal/domain"
	"github.com/TimurAUG/backgammon-game/internal/game"
	"github.com/stretchr/testify/require"
)

// TestStateMessage_IncludesBorneOffAndIsFirstMove фиксирует требование
// из nardy-protocol § «Сервер → клиент» / STATE и SPEC.md § 4: STATE
// обязан содержать поля borneOff и isFirstMove. Без них клиент не
// может показать счёт выкинутых шашек и не знает, действует ли
// исключение для дублей 6:6/4:4/3:3 на первом ходу.
//
// Закрывает открытый вопрос #1 из FRONTEND_SPEC.md § 9.
func TestStateMessage_IncludesBorneOffAndIsFirstMove(t *testing.T) {
	s := domain.GameState{
		Board:       domain.InitialBoard(),
		Turn:        domain.White,
		Status:      domain.StatusWaitingForRoll,
		BorneOff:    [2]uint8{3, 7},
		IsFirstMove: [2]bool{false, true},
	}

	msg := game.StateMessage(s)
	raw, err := json.Marshal(msg)
	require.NoError(t, err)

	var parsed struct {
		Type     string `json:"type"`
		BorneOff *struct {
			White uint8 `json:"white"`
			Black uint8 `json:"black"`
		} `json:"borneOff"`
		IsFirstMove *struct {
			White bool `json:"white"`
			Black bool `json:"black"`
		} `json:"isFirstMove"`
	}
	require.NoError(t, json.Unmarshal(raw, &parsed))

	require.Equal(t, "STATE", parsed.Type)

	require.NotNil(t, parsed.BorneOff, "STATE должен содержать поле borneOff")
	require.Equal(t, uint8(3), parsed.BorneOff.White)
	require.Equal(t, uint8(7), parsed.BorneOff.Black)

	require.NotNil(t, parsed.IsFirstMove, "STATE должен содержать поле isFirstMove")
	require.False(t, parsed.IsFirstMove.White, "white уже сделал первый ход")
	require.True(t, parsed.IsFirstMove.Black, "black ещё не ходил")
}

// TestStateMessage_RemainingIsJSONNumberArray фиксирует контракт nardy-protocol:
// dice.remaining — массив чисел ([int]), а не base64-строка. Регрессия: поле
// protocol.DicePayload.Remaining было []uint8, и encoding/json кодирует такие
// срезы в base64 ("BQM="), из-за чего TS-клиент (ждёт number[]) ломался на
// отображении кубиков. Go-тесты баг не ловили: base64↔[]uint8 round-trip
// внутри Go симметричен.
func TestStateMessage_RemainingIsJSONNumberArray(t *testing.T) {
	s := domain.GameState{
		Board:  domain.InitialBoard(),
		Turn:   domain.White,
		Status: domain.StatusWaitingForMove,
		Dice:   domain.NewDice(5, 3),
	}

	raw, err := json.Marshal(game.StateMessage(s))
	require.NoError(t, err)

	require.Contains(t, string(raw), `"remaining":[5,3]`,
		"dice.remaining должен сериализоваться как JSON-массив чисел, а не base64-строка")
}
