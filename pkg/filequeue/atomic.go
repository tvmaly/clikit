package filequeue

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrAlreadyClaimed is returned by AtomicMove when the source file no longer
// exists — meaning another worker already claimed it.
var ErrAlreadyClaimed = errors.New("already claimed")

// AtomicWrite writes data to <dir>/<filename> atomically by writing to a temp
// file in dir and then renaming into place. The caller must ensure dir exists.
func AtomicWrite(dir, filename string, data []byte) error {
	// Create temp file in the same directory so os.Rename is on the same filesystem.
	tmp, err := os.CreateTemp(dir, filename+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on any error.
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	dest := filepath.Join(dir, filename)
	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("renaming to %q: %w", dest, err)
	}
	ok = true
	return nil
}

// AtomicMove renames <srcDir>/<filename> to <destDir>/<filename> atomically.
// If the source file no longer exists (another worker claimed it), it returns
// ErrAlreadyClaimed.
func AtomicMove(srcDir, destDir, filename string) error {
	src := filepath.Join(srcDir, filename)
	dst := filepath.Join(destDir, filename)
	if err := os.Rename(src, dst); err != nil {
		if os.IsNotExist(err) {
			return ErrAlreadyClaimed
		}
		return fmt.Errorf("moving %q to %q: %w", src, dst, err)
	}
	return nil
}

// SafeRequestID validates that id is exactly 16 lowercase hex characters.
// This prevents directory traversal via crafted request IDs.
func SafeRequestID(id string) bool {
	if len(id) != 16 {
		return false
	}
	for i := 0; i < 16; i++ {
		c := id[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// GenerateRequestID generates a cryptographically random 16-char lowercase hex ID.
func GenerateRequestID() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generating request ID: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
