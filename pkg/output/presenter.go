package output

import (
	"fmt"
	"io"
	"math"
	"time"
)

// Options configures the Presenter.
type Options struct {
	// OverflowThreshold is the max bytes before spillfile is written.
	// Zero means no overflow check.
	OverflowThreshold int
	// SpillDir is the directory used for spillfiles when overflow occurs.
	SpillDir string
}

// Presenter applies the LLM presentation layer to command output:
// binary guard → overflow check → write output → metadata footer → stderr attach.
type Presenter struct {
	w    io.Writer
	opts Options
}

// NewPresenter creates a Presenter that writes to w.
func NewPresenter(w io.Writer, opts Options) *Presenter {
	return &Presenter{w: w, opts: opts}
}

// Present writes data to the presenter's writer with the full presentation
// pipeline applied. exitCode is the command exit code, stderr is the captured
// stderr (attached only on non-zero exit), dur is the command duration.
func (p *Presenter) Present(data []byte, exitCode int, stderr []byte, dur time.Duration) error {
	// 1. Binary guard.
	if IsBinary(data) {
		return fmt.Errorf("binary output detected: cannot display binary content (use --raw or redirect to file)")
	}

	// 2. Overflow check.
	out := data
	if p.opts.OverflowThreshold > 0 && len(data) > p.opts.OverflowThreshold {
		truncated, spillfile, err := CheckOverflow(data, p.opts.OverflowThreshold, p.opts.SpillDir)
		if err != nil {
			return fmt.Errorf("overflow: %w", err)
		}
		out = truncated
		if _, err := fmt.Fprintf(p.w, "%s\n[output truncated — full data written to: %s]\n", out, spillfile); err != nil {
			return err
		}
	} else {
		// 3. Write output.
		if _, err := p.w.Write(out); err != nil {
			return err
		}
		// Ensure newline before footer.
		if len(out) > 0 && out[len(out)-1] != '\n' {
			if _, err := fmt.Fprintln(p.w); err != nil {
				return err
			}
		}
	}

	// 4. Metadata footer.
	footer := formatFooter(exitCode, dur)
	if _, err := fmt.Fprintln(p.w, footer); err != nil {
		return err
	}

	// 5. Stderr attach on non-zero exit.
	if exitCode != 0 && len(stderr) > 0 {
		if _, err := fmt.Fprintf(p.w, "[stderr]\n%s\n", stderr); err != nil {
			return err
		}
	}

	return nil
}

// formatFooter returns the metadata footer string.
// Format: [exit:N | Xms] for <1s, [exit:N | X.Ys] for >=1s.
func formatFooter(exitCode int, dur time.Duration) string {
	if dur < time.Second {
		ms := int(math.Round(float64(dur) / float64(time.Millisecond)))
		return fmt.Sprintf("[exit:%d | %dms]", exitCode, ms)
	}
	secs := float64(dur) / float64(time.Second)
	return fmt.Sprintf("[exit:%d | %.1fs]", exitCode, secs)
}
