package secrets

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BWServeState is where a dotf-managed bw serve daemon leaves its trace: the
// log its stdout+stderr append to, and the pid file naming the process that
// was started.
//
// It exists because of what #1315 was. The daemon is detached from every
// console by design (WIN-012 — that is the whole point of one unlock serving
// every later terminal), so nobody is ever watching its stderr; and Start()
// ran it with nil stdio and recorded no pid. On the Windows work box
// (2026-08-27) it died twice within minutes of `dotf secrets unlock`, and the
// only evidence was that nothing listened on 8087 any more. A process nobody
// can watch has to leave its evidence behind itself.
//
// Dir is the deploy dir's state area, <DOTFILES_DIR>/state — beside the
// DR-drill marker (doctor/checks_dr.go), and for the same reason: a daemon's
// death is a property of THIS box, not of the source tree. Callers derive it
// through NewBWServeState from their own deploy-dir resolution (env.DotfilesDir
// in the CLI, cfg.DotfilesDir in doctor) so the writer and the reader can never
// disagree on where the files are.
type BWServeState struct {
	// Dir is the state directory. "" writes and reads nothing — the pre-#1315
	// behaviour, kept so tests of the HTTP half need no filesystem. Production
	// always sets it.
	Dir string
}

const (
	bwServeStateSubdir = "state"
	bwServeLogName     = "bw-serve.log"
	bwServePIDName     = "bw-serve.pid"

	// bwServeLogCap bounds the log. A log over the cap is rotated to `.log.1`
	// (replacing the previous one) the next time a daemon is started, so the
	// two files together never exceed twice this. A few hundred KB holds many
	// Node stack traces, which is what bw serve emits when it dies.
	bwServeLogCap = 256 * 1024

	// bwServeTailBytes is how much of the log LastLogLines reads. The last
	// lines are the evidence; the whole file never travels into a report.
	bwServeTailBytes = 16 * 1024

	// bwServeLogLineWidth caps each reported line. Doctor prints these on one
	// report line, and a Node stack frame can run to hundreds of characters.
	bwServeLogLineWidth = 160
)

// NewBWServeState places the state area under dotfilesDir. An empty dotfilesDir
// yields a disabled state rather than a relative "state/" in the cwd.
func NewBWServeState(dotfilesDir string) BWServeState {
	if dotfilesDir == "" {
		return BWServeState{}
	}
	return BWServeState{Dir: filepath.Join(dotfilesDir, bwServeStateSubdir)}
}

func (s BWServeState) enabled() bool { return s.Dir != "" }

// LogPath is the daemon's log; "" when the state is disabled.
func (s BWServeState) LogPath() string {
	if !s.enabled() {
		return ""
	}
	return filepath.Join(s.Dir, bwServeLogName)
}

// PIDPath is the daemon's pid file; "" when the state is disabled.
func (s BWServeState) PIDPath() string {
	if !s.enabled() {
		return ""
	}
	return filepath.Join(s.Dir, bwServePIDName)
}

// openLog opens the log for appending — rotating it first when it is over the
// cap — and creates the state dir on first use. 0700/0600: nothing here is a
// secret (bw serve writes diagnostics to stdio, never vault material), but the
// directory sits beside the DR escrow and inherits its posture.
func (s BWServeState) openLog() (*os.File, error) {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("create bw serve state dir: %w", err)
	}
	if err := s.rotateIfOver(bwServeLogCap); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(s.LogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open bw serve log: %w", err)
	}
	return f, nil
}

// rotateIfOver moves the log aside once it exceeds capBytes. Rename over an
// existing `.log.1` replaces it, so exactly one generation is kept. A rename
// can fail while a still-alive daemon holds the file open (Windows, when the
// handle was not opened with FILE_SHARE_DELETE); truncating then keeps the cap
// honest at the cost of the older half, which beats a log that grows without
// bound.
func (s BWServeState) rotateIfOver(capBytes int64) error {
	fi, err := os.Stat(s.LogPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat bw serve log: %w", err)
	}
	if fi.Size() <= capBytes {
		return nil
	}
	if err := os.Rename(s.LogPath(), s.LogPath()+".1"); err == nil {
		return nil
	}
	if err := os.Truncate(s.LogPath(), 0); err != nil {
		return fmt.Errorf("rotate bw serve log: %w", err)
	}
	return nil
}

// WritePID records the started daemon's pid, replacing whatever a previous
// start left behind.
func (s BWServeState) WritePID(pid int) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return fmt.Errorf("create bw serve state dir: %w", err)
	}
	if err := os.WriteFile(s.PIDPath(), []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		return fmt.Errorf("write bw serve pid file: %w", err)
	}
	return nil
}

// ReadPID returns the recorded pid. A missing file — no daemon was ever
// started from this deploy dir, or the state is disabled — reports
// os.ErrNotExist (wrapped), which callers tell apart from an unparseable one.
func (s BWServeState) ReadPID() (int, error) {
	if !s.enabled() {
		return 0, fmt.Errorf("bw serve state disabled: %w", os.ErrNotExist)
	}
	raw, err := os.ReadFile(s.PIDPath())
	if err != nil {
		return 0, err
	}
	text := strings.TrimSpace(string(raw))
	pid, err := strconv.Atoi(text)
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("%s: not a pid: %q", s.PIDPath(), text)
	}
	return pid, nil
}

// LastLogLines returns at most n trailing non-empty lines of the log, each cut
// to bwServeLogLineWidth. It reads only the tail of the file, so a log at its
// cap costs the same as an empty one. A missing log yields no lines and no
// error: the caller is reporting a dead daemon, and "it left no log" is a
// finding, not a failure of the reporter.
func (s BWServeState) LastLogLines(n int) ([]string, error) {
	if !s.enabled() || n <= 0 {
		return nil, nil
	}
	f, err := os.Open(s.LogPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	start := fi.Size() - bwServeTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	if start > 0 && len(lines) > 0 {
		lines = lines[1:] // the seek landed mid-line; the fragment is not a line
	}
	return tailLines(lines, n), nil
}

// tailLines keeps the last n non-empty lines, trimmed and width-capped.
func tailLines(lines []string, n int) []string {
	kept := make([]string, 0, n)
	for i := len(lines) - 1; i >= 0 && len(kept) < n; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if len(line) > bwServeLogLineWidth {
			line = line[:bwServeLogLineWidth] + "…"
		}
		kept = append(kept, line)
	}
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}
	return kept
}

// markStart writes the one line dotf itself contributes to the log: which pid
// it started and when. It is what separates one daemon's output from the
// next's in an append-only file, and it is written by dotf rather than the
// child so it lands even if bw serve prints nothing before dying.
func markStart(w io.Writer, pid int, now time.Time) {
	_, _ = fmt.Fprintf(w, "==== dotf: started bw serve pid %d at %s ====\n", pid, now.UTC().Format(time.RFC3339))
}

// Trace names where the daemon left its trace, for `dotf secrets unlock` and
// `lock` to print beside their confirmation: an operator who reads "unlocked"
// also learns where to look when the daemon is gone by the next call. A daemon
// this dotf did not start has no pid file, and that is said rather than
// guessed at.
func (d *BWServeDaemon) Trace() string {
	logPath := d.State.LogPath()
	if logPath == "" {
		return "no state dir — pid and log not recorded"
	}
	pid, err := d.State.ReadPID()
	if err != nil {
		return fmt.Sprintf("pid unknown — not started by this dotf; log %s", logPath)
	}
	return fmt.Sprintf("pid %d, log %s", pid, logPath)
}
