package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tvmaly/clikit/pkg/filequeue"
)

// RunCleanup removes expired files from done/, dead/, orphaned payloads, and tmp/.
// Returns the count of files deleted.
func RunCleanup(queueRoot string, doneTTL, deadTTL time.Duration) (int, error) {
	p := filequeue.QueuePaths(queueRoot)
	cleaned := 0

	// Clean done/.
	n, err := cleanMetaDir(p.Done, p.PayloadsResp, p.PayloadsReq, doneTTL)
	if err != nil {
		return cleaned, fmt.Errorf("cleaning done/: %w", err)
	}
	cleaned += n

	// Clean dead/.
	n, err = cleanMetaDir(p.Dead, p.PayloadsResp, p.PayloadsReq, deadTTL)
	if err != nil {
		return cleaned, fmt.Errorf("cleaning dead/: %w", err)
	}
	cleaned += n

	// Clean orphaned tmp/ files older than 1 hour.
	n, err = cleanTmp(p.Tmp, time.Hour)
	if err != nil {
		return cleaned, fmt.Errorf("cleaning tmp/: %w", err)
	}
	cleaned += n

	return cleaned, nil
}

// cleanMetaDir removes meta files older than ttl and their associated payloads.
func cleanMetaDir(metaDir, payloadsRespDir, payloadsReqDir string, ttl time.Duration) (int, error) {
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cleaned := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		metaPath := filepath.Join(metaDir, e.Name())
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta filequeue.ResponseMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		if meta.CompletedAt.IsZero() || time.Since(meta.CompletedAt) <= ttl {
			continue
		}

		// Delete meta file.
		if err := os.Remove(metaPath); err == nil {
			cleaned++
		}

		// Delete response payload.
		id := meta.RequestID
		if id != "" {
			respPayload := filepath.Join(payloadsRespDir, id+".json")
			if err := os.Remove(respPayload); err == nil {
				cleaned++
			}
			reqPayload := filepath.Join(payloadsReqDir, id+".json")
			if err := os.Remove(reqPayload); err == nil {
				cleaned++
			}
		}
	}
	return cleaned, nil
}

// cleanTmp removes files in tmpDir older than maxAge.
func cleanTmp(tmpDir string, maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	cleaned := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(fi.ModTime()) > maxAge {
			if err := os.Remove(filepath.Join(tmpDir, e.Name())); err == nil {
				cleaned++
			}
		}
	}
	return cleaned, nil
}
