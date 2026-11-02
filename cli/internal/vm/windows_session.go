package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const windowsOS = "windows"

// WindowsSessions holds the one Windows guest (ADR-055 decision 2): one QGA
// socket, one overlay, one set of ports. QEMU's QMP socket is the source of
// truth for the guest, the check `Run` refuses a second run on. Sessions() keeps
// one empty directory, a name and nothing else, so a guest that is gone can be
// told from one that was never made, and a session from a kept run.
type WindowsSessions struct {
	H       Host
	Machine Machine
	Agent   time.Duration // until the guest agent answers and the test user is logged on
	Poll    time.Duration
	Grace   time.Duration // how long a timed-out command gets to be killed in the guest
	Now     func() time.Time
}

// holder names the session that holds the guest, "" when none does.
func (s WindowsSessions) holder() (string, error) {
	entries, err := os.ReadDir(s.Machine.Layout.Sessions())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	sort.Strings(names)
	return names[0], nil
}

func (s WindowsSessions) rec(name string, st Status, start time.Time, msg string) Record {
	r := NewRecord(name, windowsOS, st)
	r.Stderr = msg
	r.DurationMS = time.Since(start).Milliseconds()
	return r
}

// refusal answers why the guest cannot be booted now, if it cannot.
func (s WindowsSessions) refusal(name string, start time.Time) (Record, bool) {
	l := s.Machine.Layout
	holder, err := s.holder()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error()), true
	}
	live := running(s.H, l)
	down := fmt.Sprintf("dotf vm windows down --session %s", holder)
	switch {
	case holder != "" && live:
		return s.rec(name, StatusBusy, start, fmt.Sprintf("session %s holds the Windows guest: `%s` first", holder, down)), true
	case holder != "":
		return s.rec(name, StatusStale, start, fmt.Sprintf("session %s has no running guest: `%s` clears it", holder, down)), true
	case live:
		return s.rec(name, StatusBusy, start, "a Windows guest is running that no session holds: `dotf vm windows stop` stops it"), true
	}
	done, err := baseBuilt(l, false)
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error()), true
	}
	if !done {
		return s.rec(name, StatusFailed, start, fmt.Sprintf("no clean-base at %s: run `dotf vm windows build` first", l.BaseDisk())), true
	}
	return Record{}, false
}

// Up boots an overlay of clean-base and leaves it running until Down. A boot
// that fails halfway leaves no guest, no overlay and no record. The guest
// outlives this process, so nothing here can watch it after Up returns.
func (s WindowsSessions) Up(_ context.Context, name string, progress, _ io.Writer) Record {
	start := time.Now()
	if name == "" {
		name = fmt.Sprintf("%s-%d", windowsOS, s.Now().Unix())
	}
	if err := ValidSessionName(name); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if refused, ok := s.refusal(name, start); ok {
		return refused
	}
	l := s.Machine.Layout
	p, q, err := bootGuest(s.H, RunOptions{Machine: s.Machine, Agent: s.Agent, Poll: s.Poll, Progress: progress})
	if q != nil {
		_ = q.Close()
	}
	dir := filepath.Join(l.Sessions(), name)
	if err == nil {
		err = os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		if p != nil {
			kill(p)
		}
		err = errors.Join(err, removeIfPresent(l.Disk()), removeIfPresent(l.Vars()))
		return s.rec(name, StatusFailed, start, err.Error())
	}
	r := s.rec(name, StatusOK, start, "")
	r.StateDir = dir
	return r
}

// Down stops the guest a session holds and deletes its overlay and record. It
// acts only on the session it is named, so a kept run is never its to stop.
func (s WindowsSessions) Down(_ context.Context, name string, out io.Writer) Record {
	start := time.Now()
	holder, err := s.holder()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if ValidSessionName(name) != nil || holder != name {
		return s.rec(name, StatusNoSession, start, fmt.Sprintf("no session %s", name))
	}
	if err := Stop(s.H, s.Machine.Layout, out); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	dir := filepath.Join(s.Machine.Layout.Sessions(), name)
	if err := os.RemoveAll(dir); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	r := s.rec(name, StatusOK, start, "")
	r.StateDir = dir
	return r
}

// List answers whether the guest the session holds is up: ok when QEMU answers,
// stale when it does not.
func (s WindowsSessions) List(context.Context) ([]Record, error) {
	holder, err := s.holder()
	if err != nil {
		return nil, err
	}
	if holder == "" {
		return []Record{}, nil
	}
	st := StatusStale
	if running(s.H, s.Machine.Layout) {
		st = StatusOK
	}
	r := NewRecord(holder, windowsOS, st)
	r.StateDir = filepath.Join(s.Machine.Layout.Sessions(), holder)
	return []Record{r}, nil
}
