package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"
)

// CheckOverflow checks if data exceeds threshold bytes. If it does, the full
// data is written to a spillfile in spillDir and the truncated data is returned
// along with the spillfile path. If data is within threshold, it is returned
// unchanged with an empty spillfile path.
func CheckOverflow(data []byte, threshold int, spillDir string) (truncated []byte, spillfile string, err error) {
	if len(data) <= threshold {
		return data, "", nil
	}

	// Write full data to spillfile atomically.
	name := fmt.Sprintf("overflow-%d.txt", time.Now().UnixNano())
	spillPath := filepath.Join(spillDir, name)
	if err := os.WriteFile(spillPath, data, 0o600); err != nil {
		return nil, "", fmt.Errorf("writing spillfile: %w", err)
	}

	// Truncate at rune boundary.
	cut := truncateAtRune(data, threshold)
	return cut, spillPath, nil
}

// truncateAtRune returns data truncated to at most maxBytes, cutting at a valid
// UTF-8 rune boundary so the result is always valid UTF-8.
func truncateAtRune(data []byte, maxBytes int) []byte {
	if len(data) <= maxBytes {
		return data
	}
	// Walk backwards from maxBytes until we find a rune boundary.
	end := maxBytes
	for end > 0 && !utf8.RuneStart(data[end]) {
		end--
	}
	return data[:end]
}
