package output

import "unicode/utf8"

// nonPrintableThreshold is the fraction of non-printable bytes above which
// content is considered binary. Excludes common text control chars (tab, LF, CR).
const nonPrintableThreshold = 0.30

// IsBinary reports whether data appears to be binary content.
// It returns true if data contains a null byte, invalid UTF-8 sequences,
// or a high ratio of non-printable bytes.
func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	nonPrintable := 0
	i := 0
	for i < len(data) {
		b := data[i]

		// Null byte is a definitive binary marker.
		if b == 0x00 {
			return true
		}

		// Allow common text control characters.
		if b == '\t' || b == '\n' || b == '\r' {
			i++
			continue
		}

		// Check for valid UTF-8 rune.
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte sequence — binary.
			return true
		}

		// Count non-printable ASCII bytes (below space, excluding allowed ones above).
		if b < 0x20 || b == 0x7f {
			nonPrintable++
		}

		i += size
	}

	ratio := float64(nonPrintable) / float64(len(data))
	return ratio >= nonPrintableThreshold
}
