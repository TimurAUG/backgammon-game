package game

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateID_FixedReader_HexEncodesBytes — generateID читает nBytes из
// reader и возвращает их hex. Фикс-reader даёт детерминизм в тестах, в проде
// источник — crypto/rand. TDD plan #38.
func TestGenerateID_FixedReader_HexEncodesBytes(t *testing.T) {
	r := bytes.NewReader([]byte{0xde, 0xad, 0xbe, 0xef})

	id, err := generateID(r, 4)

	require.NoError(t, err)
	require.Equal(t, "deadbeef", id)
}

// TestGenerateID_ShortRead_ReturnsError — если в reader меньше байт, чем
// запрошено, generateID возвращает ошибку (io.ReadFull, не частичное чтение).
func TestGenerateID_ShortRead_ReturnsError(t *testing.T) {
	r := bytes.NewReader([]byte{0x01})

	_, err := generateID(r, 4)

	require.Error(t, err)
}
