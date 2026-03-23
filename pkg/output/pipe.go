package output

import (
	"io"
	"os"
)

// IsPipedWriter reports whether w is piped (i.e. not an interactive terminal).
// If w is an *os.File, it checks whether it is a character device (terminal).
// Any other writer type is treated as piped.
func IsPipedWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return true
	}
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	// A character device is a terminal; everything else is piped.
	return fi.Mode()&os.ModeCharDevice == 0
}

// IsStdoutPiped reports whether os.Stdout is piped.
func IsStdoutPiped() bool {
	return IsPipedWriter(os.Stdout)
}

// CopyRaw copies all bytes from r to w without any transformation.
func CopyRaw(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	return err
}
