package spec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// stallWatch decides whether a transcript has stopped changing. It compares the
// file's size and modification time rather than reading it, so a transcript of
// hundreds of megabytes costs one stat per tick, and it counts a change rather
// than trusting the clock of the file: only "different from last time" is
// progress.
type stallWatch struct {
	path     string
	timeout  time.Duration
	tick     <-chan time.Time // nil when the detector is off: a nil channel never fires
	ticker   *time.Ticker
	size     int64
	modTime  time.Time
	progress time.Time
}

func newStallWatch(l Limits) *stallWatch {
	w := &stallWatch{path: l.Transcript, timeout: l.StallTimeout, progress: time.Now()}
	if l.StallTimeout <= 0 || l.Transcript == "" {
		return w
	}
	// A transcript left by an earlier round is the baseline, not progress.
	w.size, w.modTime = statTranscript(l.Transcript)
	w.ticker = time.NewTicker(pollEvery(l.StallTimeout))
	w.tick = w.ticker.C
	return w
}

func (w *stallWatch) stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
}

// stalled records any change since the last look and reports whether the
// transcript has been still for the whole timeout. A missing file reads as
// size -1, so its creation is a change and its absence is silence.
func (w *stallWatch) stalled() bool {
	size, mod := statTranscript(w.path)
	if size != w.size || !mod.Equal(w.modTime) {
		w.size, w.modTime, w.progress = size, mod, time.Now()
	}
	return time.Since(w.progress) >= w.timeout
}

func statTranscript(path string) (int64, time.Time) {
	fi, err := os.Stat(path)
	if err != nil {
		return -1, time.Time{}
	}
	return fi.Size(), fi.ModTime()
}

// pollEvery is how often the transcript is looked at: often enough that the
// stop lands close to the timeout, rarely enough to cost nothing.
func pollEvery(timeout time.Duration) time.Duration {
	return min(max(timeout/4, 20*time.Millisecond), 5*time.Second)
}

// transcriptTail is how much of the end of a transcript LastToolCall reads. A
// settled tool event is a few kilobytes at most, and the call that wedged is
// near the end by definition.
const transcriptTail = 256 << 10

// lastCallWidth bounds the command in the stop message.
const lastCallWidth = 200

// LastToolCall names the last tool call a reviewer's transcript recorded, as
// `<tool>: <command or path>` on one bounded line, or "" when it records none.
//
// The shape is pi's (`toolName` and `args`, on tool_execution_* events), the
// runner whose transcript has been measured. Another runner's stream records no
// such event and gets no line: a guessed wedge is worse than none, the rule
// TurnCapHint follows.
func LastToolCall(transcript string) string {
	f, err := os.Open(transcript)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	tail, err := readTail(f, transcriptTail)
	if err != nil {
		return ""
	}
	lines := bytes.Split(tail, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		var ev struct {
			Type     string `json:"type"`
			ToolName string `json:"toolName"`
			Args     struct {
				Command string `json:"command"`
				Path    string `json:"path"`
			} `json:"args"`
		}
		// A line cut at the start of the tail, or one that is not JSON, is
		// skipped like any other that names no call.
		if json.Unmarshal(lines[i], &ev) != nil || !strings.HasPrefix(ev.Type, "tool_execution_") || ev.ToolName == "" {
			continue
		}
		target := ev.Args.Command
		if target == "" {
			target = ev.Args.Path
		}
		if target = boundedLine(target); target == "" {
			return ev.ToolName
		}
		return ev.ToolName + ": " + target
	}
	return ""
}

func readTail(f *os.File, n int64) ([]byte, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() > n {
		if _, err := f.Seek(fi.Size()-n, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return io.ReadAll(bufio.NewReader(f))
}

// boundedLine folds s onto one line (oneLine) no wider than lastCallWidth runes.
func boundedLine(s string) string {
	s = oneLine(s)
	if utf8.RuneCountInString(s) <= lastCallWidth {
		return s
	}
	return string([]rune(s)[:lastCallWidth]) + "…"
}

// StoppedRunRecord says what a run the launcher stopped left in its review file
// (CLI-094, #2448). The launcher moves the previous round away before it
// starts, so the file is absent, a provisional record the reviewer wrote first,
// or a verdict it signed before the stop. Without this line, a stopped run and
// a run that never began look the same on disk.
func StoppedRunRecord(reviewPath string) string {
	name := filepath.Base(reviewPath)
	raw, err := os.ReadFile(reviewPath)
	if err != nil {
		return name + " was not rewritten by this run: nothing is on disk to archive"
	}
	r, perr := ParseReview(string(raw))
	switch {
	case r.Verdict == VerdictPending:
		return "provisional " + name + " kept (verdict PENDING): the findings so far, unsigned; `dotf spec archive` refuses it"
	case perr == nil:
		return fmt.Sprintf("%s holds the verdict %s, written before the run was stopped", name, r.Verdict)
	default:
		return fmt.Sprintf("%s exists but cannot be read as a review (%v)", name, perr)
	}
}
