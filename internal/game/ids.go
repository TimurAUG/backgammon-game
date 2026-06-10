package game

import (
	"encoding/hex"
	"io"
)

// generateID читает nBytes из r и возвращает их hex-представление (2*nBytes
// символов). r — источник криптослучайности: crypto/rand в проде, фикс-reader
// в тестах. Используется для серверной генерации gameId и token в REST-флоу.
func generateID(r io.Reader, nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
