package converge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// A converge run keeps its own transcript beside its report (#1098):
//
//	<StateDir>/converge/runs/<stamp>.log   everything the run printed, subprocesses included
//	<StateDir>/converge/runs/<stamp>.json  the run's report (last.json is the latest full run)
//
// The transcript is redacted on write and owner-only from creation, because it
// holds what uv, npm, winget and brew printed, and any of them can print a
// credential. It stays on the machine: StateDir is outside the checkout and
// outside the deploy mirror, and nothing uploads it.

// runStampLayout names a run: UTC, sortable, and legal in a Windows file name.
const runStampLayout = "20060102T150405Z"

// RunLogKeep is how many runs PruneRuns leaves.
const RunLogKeep = 20

// minMaskLen is the shortest value masked literally: shorter ones ("1", "main")
// would shred every line. `dotf secrets run` uses the same floor.
const minMaskLen = 6

// RunLog is one run's transcript. Every Stream writes whole, redacted lines.
type RunLog struct {
	dir, stamp string
	mu         sync.Mutex
	f          *os.File
	masks      []mask
	streams    []*runStream
}

type mask struct{ value, replacement string }

// OpenRunLog creates the run's transcript under state/converge/runs, owner-only,
// and writes its header. secrets maps a variable name to its value; each value
// of at least minMaskLen bytes is replaced by [REDACTED:<name>] wherever it
// appears, before any token-shape rule runs.
func OpenRunLog(state string, start time.Time, secrets map[string]string, header string) (*RunLog, error) {
	dir := filepath.Join(state, "converge", "runs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	// MkdirAll leaves an existing directory's mode alone: tighten it, never relax.
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, err
	}
	base := start.UTC().Format(runStampLayout)
	var (
		f     *os.File
		stamp string
		err   error
	)
	for i := 1; i <= 100; i++ {
		stamp = base
		if i > 1 {
			stamp = fmt.Sprintf("%s-%d", base, i)
		}
		f, err = os.OpenFile(filepath.Join(dir, stamp+".log"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if !errors.Is(err, os.ErrExist) {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	rl := &RunLog{dir: dir, stamp: stamp, f: f}
	for name, value := range secrets {
		if len(value) >= minMaskLen {
			rl.masks = append(rl.masks, mask{value, "[REDACTED:" + name + "]"})
		}
	}
	// Longest first, so a value that is a prefix of another cannot break the
	// longer one's match and leak its suffix.
	sort.SliceStable(rl.masks, func(i, j int) bool { return len(rl.masks[i].value) > len(rl.masks[j].value) })
	if _, err := fmt.Fprintf(f, "# %s\n# started %s\n", rl.redact(header), start.UTC().Format(time.RFC3339)); err != nil {
		_ = f.Close()
		return nil, err
	}
	return rl, nil
}

// Path is the transcript's file.
func (rl *RunLog) Path() string { return filepath.Join(rl.dir, rl.stamp+".log") }

// Dir is the directory every run's files live in.
func (rl *RunLog) Dir() string { return rl.dir }

// Stream returns a writer with its own partial line, so two streams (stdout
// and stderr) interleave by whole lines in the file.
func (rl *RunLog) Stream() io.Writer {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	s := &runStream{rl: rl}
	rl.streams = append(rl.streams, s)
	return s
}

// Redacted returns a copy of a run's report and error with the transcript's
// redaction applied to every detail and to the error, for the files written
// beside it. The caller's report is not changed.
func (rl *RunLog) Redacted(rep Report, runErr error) (Report, error) {
	out := rep
	out.Entries = make([]Entry, len(rep.Entries))
	for i, e := range rep.Entries {
		e.Detail = rl.redact(e.Detail)
		out.Entries[i] = e
	}
	if runErr != nil {
		runErr = errors.New(rl.redact(runErr.Error()))
	}
	return out, runErr
}

// WriteReport persists the run's report beside its transcript. Pass it what
// Redacted returns.
func (rl *RunLog) WriteReport(rep Report, runErr error) error {
	return WriteReport(filepath.Join(rl.dir, rl.stamp+".json"), rep, runErr)
}

// Close flushes every stream's unterminated last line and closes the file.
func (rl *RunLog) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.f == nil {
		return nil
	}
	for _, s := range rl.streams {
		if len(s.partial) > 0 {
			rl.writeLine(append(s.partial, '\n'))
			s.partial = nil
		}
	}
	err := rl.f.Close()
	rl.f = nil
	return err
}

// writeLine redacts one whole line and appends it. Callers hold rl.mu. A
// write after Close is dropped: a child that outlived the run has no transcript
// left to write to.
func (rl *RunLog) writeLine(line []byte) {
	if rl.f == nil {
		return
	}
	_, _ = rl.f.Write([]byte(rl.redact(string(line))))
}

type runStream struct {
	rl      *RunLog
	partial []byte
}

// maxPartialLine bounds the line a stream holds while it waits for a newline.
// A child that never ends one (carriage-return progress bars, a binary on
// stdout) would otherwise grow the buffer for the whole run.
var maxPartialLine = 1 << 20

func (s *runStream) Write(p []byte) (int, error) {
	s.rl.mu.Lock()
	defer s.rl.mu.Unlock()
	scanned := len(s.partial)
	data := append(s.partial, p...)
	consumed := false
	for {
		i := bytes.IndexByte(data[scanned:], '\n')
		if i < 0 {
			break
		}
		s.rl.writeLine(data[:scanned+i+1])
		data, scanned, consumed = data[scanned+i+1:], 0, true
	}
	if keep := s.rl.tail(); len(data) > maxPartialLine+keep {
		cut := s.rl.safeCut(data, len(data)-keep)
		s.rl.writeLine(data[:cut])
		data, consumed = data[cut:], true
	}
	if consumed {
		// Move the remainder to the front of the same array: only what is
		// left after a line is copied, never the line as it grows.
		data = append(s.partial[:0], data...)
	}
	s.partial = data
	return len(p), nil
}

// tail is how much of an over-long line stays buffered when its head is
// written: enough to hold the start of any secret that has not fully arrived,
// a masked value or a token shape.
func (rl *RunLog) tail() int {
	n := 256
	if len(rl.masks) > 0 && len(rl.masks[0].value) > n { // masks are longest first
		n = len(rl.masks[0].value)
	}
	return n
}

// safeCut moves cut back to before any masked value, token shape or escape
// sequence that spans it, so the head and the rest are each redacted whole.
// A match that starts at the very beginning is longer than maxPartialLine; no
// credential is, so the cut stays where it was.
func (rl *RunLog) safeCut(data []byte, cut int) int {
	for moved := true; moved; {
		moved = false
		for _, m := range rl.masks {
			v := []byte(m.value)
			from := max(0, cut-len(v)+1)
			if i := bytes.Index(data[from:min(len(data), cut+len(v)-1)], v); i >= 0 && from+i < cut {
				cut, moved = from+i, true
			}
		}
		for _, re := range append([]*regexp.Regexp{ansiEscape}, tokenShapes...) {
			for _, loc := range re.FindAllIndex(data, -1) {
				if loc[0] < cut && cut < loc[1] {
					cut, moved = loc[0], true
				}
			}
		}
	}
	if cut <= 0 {
		return len(data) - rl.tail()
	}
	return cut
}

var (
	// ansiEscape matches CSI and OSC sequences: colour, cursor and title codes.
	ansiEscape = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)`)
	// tokenShapes are credentials recognisable by their form alone. Each has
	// two groups, the context before and after the secret, which survive; the
	// secret between them is masked.
	tokenShapes = []*regexp.Regexp{
		regexp.MustCompile(`()\bgh[pousr]_[A-Za-z0-9]{30,}()`),
		regexp.MustCompile(`()\bgithub_pat_[A-Za-z0-9_]{20,}()`),
		regexp.MustCompile(`()\bsk-[A-Za-z0-9_-]{20,}()`),
		regexp.MustCompile(`()\bAKIA[0-9A-Z]{16}\b()`),
		regexp.MustCompile(`()\bxox[abprs]-[A-Za-z0-9-]{10,}()`),
		regexp.MustCompile(`()\bAGE-SECRET-KEY-1[0-9A-Z]{20,}()`),
		regexp.MustCompile(`()\bnpm_[A-Za-z0-9]{36}\b()`),
		regexp.MustCompile(`()\bpypi-AgEI[A-Za-z0-9_-]{20,}()`),
		regexp.MustCompile(`(?i)(\b(?:proxy-)?authorization:\s*(?:bearer|basic|token)\s+)[^\s'"]+()`),
		regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._~+/=-]{16,}()`),
		// URL userinfo: scheme://user:password@host keeps the user, masks the password.
		regexp.MustCompile(`(\b[A-Za-z][A-Za-z0-9+.-]*://[^/\s:@]+:)[^@\s/]+(@)`),
	}
)

func (rl *RunLog) redact(line string) string {
	line = ansiEscape.ReplaceAllString(line, "")
	for _, m := range rl.masks {
		line = strings.ReplaceAll(line, m.value, m.replacement)
	}
	for _, re := range tokenShapes {
		line = re.ReplaceAllString(line, "${1}[REDACTED]${2}")
	}
	return line
}

// PruneRuns leaves the newest keep runs in dir. A run is every file sharing a
// stamp (its .log and its .json); an orphan transcript counts as a run, and a
// file that is not a run is never touched.
func PruneRuns(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	files := map[string][]string{}
	for _, e := range entries {
		name := e.Name()
		stamp := strings.TrimSuffix(strings.TrimSuffix(name, ".json"), ".log")
		if stamp == name || !isRunStamp(stamp) {
			continue
		}
		files[stamp] = append(files[stamp], name)
	}
	stamps := make([]string, 0, len(files))
	for s := range files {
		stamps = append(stamps, s)
	}
	sort.Strings(stamps)
	var errs []error
	for len(stamps) > keep {
		for _, name := range files[stamps[0]] {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				errs = append(errs, err)
			}
		}
		stamps = stamps[1:]
	}
	return errors.Join(errs...)
}

// isRunStamp reports whether s is a run stamp, with or without a same-second
// suffix ("-2").
func isRunStamp(s string) bool {
	base, _, _ := strings.Cut(s, "-")
	_, err := time.Parse(runStampLayout, base)
	return err == nil
}
