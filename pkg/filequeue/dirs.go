package filequeue

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths holds the resolved absolute paths for all queue subdirectories.
type Paths struct {
	Pending     string
	Claimed     string
	Done        string
	Dead        string
	PayloadsReq  string
	PayloadsResp string
	Tmp         string
}

// QueuePaths returns the Paths struct for the given queue root.
func QueuePaths(root string) *Paths {
	return &Paths{
		Pending:     filepath.Join(root, "pending"),
		Claimed:     filepath.Join(root, "claimed"),
		Done:        filepath.Join(root, "done"),
		Dead:        filepath.Join(root, "dead"),
		PayloadsReq:  filepath.Join(root, "payloads", "req"),
		PayloadsResp: filepath.Join(root, "payloads", "resp"),
		Tmp:         filepath.Join(root, "tmp"),
	}
}

// EnsureQueueDirs creates all required queue subdirectories under root.
// It is idempotent — calling it multiple times is safe.
func EnsureQueueDirs(root string) error {
	p := QueuePaths(root)
	dirs := []string{
		p.Pending, p.Claimed, p.Done, p.Dead,
		p.PayloadsReq, p.PayloadsResp, p.Tmp,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return fmt.Errorf("creating queue dir %q: %w", d, err)
		}
	}
	return nil
}
