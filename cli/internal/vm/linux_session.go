package vm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

const linuxOS = "linux"

var sessionName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,30}$`)

// ValidSessionName refuses a name that would not be one path element or one
// Lima instance name.
func ValidSessionName(name string) error {
	if !sessionName.MatchString(name) {
		return fmt.Errorf("session name %q: use 1 to 31 lower-case letters, digits and hyphens, not starting with a hyphen", name)
	}
	return nil
}

// LinuxSessions holds named Lima guests (ADR-055). The hypervisor is the
// source of truth for a guest; Dir keeps one empty directory per session, a
// name and nothing else, so a guest that is gone can be told from one that was
// never made.
type LinuxSessions struct {
	Dir string // <state>/vm/linux/sessions
	L   Limactl
	Now func() time.Time
}

func instanceOf(session string) string { return "dotf-" + session }

// lima is one entry of `limactl list --json`, which prints a JSON object per
// instance and nothing when there is none (Lima 2.2.1).
type lima struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (s LinuxSessions) instances() (map[string]lima, error) {
	var out bytes.Buffer
	if err := s.L(nil, &out, io.Discard, "list", "--json"); err != nil {
		return nil, fmt.Errorf("limactl list: %w", err)
	}
	found := map[string]lima{}
	sc := bufio.NewScanner(&out)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var l lima
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("limactl list: %w", err)
		}
		found[l.Name] = l
	}
	return found, sc.Err()
}

func (s LinuxSessions) held(name string) bool {
	_, err := os.Stat(filepath.Join(s.Dir, name))
	return err == nil
}

func (s LinuxSessions) rec(name string, st Status, start time.Time, msg string) Record {
	r := NewRecord(name, linuxOS, st)
	r.Stderr = msg
	r.DurationMS = time.Since(start).Milliseconds()
	return r
}

// Up creates a session, named name or linux-<unix time>, and keeps it until
// Down. limactl's own output goes to out, never to the record.
func (s LinuxSessions) Up(name string, progress, out io.Writer) Record {
	start := time.Now()
	if name == "" {
		name = fmt.Sprintf("linux-%d", s.Now().Unix())
	}
	if err := ValidSessionName(name); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	down := fmt.Sprintf("dotf vm linux down --session %s", name)
	found, err := s.instances()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	inst, there := found[instanceOf(name)]
	switch {
	case s.held(name) && there && inst.Status == "Running":
		return s.rec(name, StatusBusy, start, fmt.Sprintf("session %s is up: use another name, or `%s` first", name, down))
	case s.held(name):
		return s.rec(name, StatusStale, start, fmt.Sprintf("session %s has no running guest: `%s` clears it", name, down))
	case there:
		return s.rec(name, StatusBusy, start, fmt.Sprintf("a Lima instance %s exists that dotf did not create: use another name", instanceOf(name)))
	}
	dir := filepath.Join(s.Dir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if err := s.start(instanceOf(name), progress, out); err != nil {
		// A start that fails partway can leave the instance and its disk behind.
		err = errors.Join(err, s.L(nil, out, out, "delete", "--force", instanceOf(name)), os.RemoveAll(dir))
		return s.rec(name, StatusFailed, start, err.Error())
	}
	r := s.rec(name, StatusOK, start, "")
	r.StateDir = dir
	return r
}

func (s LinuxSessions) start(instance string, progress, out io.Writer) error {
	tmpl, cleanup, err := writeTemplate()
	if err != nil {
		return err
	}
	defer cleanup()
	step(progress, "boot %s (Ubuntu 24.04, nothing shared with the host)", instance)
	if err := s.L(nil, out, out, "start", "--tty=false", "--name", instance, tmpl); err != nil {
		return fmt.Errorf("limactl start: %w", err)
	}
	return nil
}

// Down deletes the session's guest and its state, and clears a stale one. It
// acts only on a session dotf holds, never on a Lima instance it did not make.
func (s LinuxSessions) Down(name string, out io.Writer) Record {
	start := time.Now()
	if ValidSessionName(name) != nil || !s.held(name) {
		return s.rec(name, StatusNoSession, start, fmt.Sprintf("no session %s", name))
	}
	found, err := s.instances()
	if err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	if _, there := found[instanceOf(name)]; there {
		if err := s.L(nil, out, out, "delete", "--force", instanceOf(name)); err != nil {
			return s.rec(name, StatusFailed, start, fmt.Sprintf("limactl delete %s: %v", instanceOf(name), err))
		}
	}
	dir := filepath.Join(s.Dir, name)
	if err := os.RemoveAll(dir); err != nil {
		return s.rec(name, StatusFailed, start, err.Error())
	}
	r := s.rec(name, StatusOK, start, "")
	r.StateDir = dir
	return r
}

// List answers what the hypervisor says about each session dotf holds: ok for
// a running guest, stale for a record whose guest is gone or stopped. It is
// never nil, so --json writes [].
func (s LinuxSessions) List() ([]Record, error) {
	found, err := s.instances()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	sessions := []Record{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st := StatusStale
		if found[instanceOf(e.Name())].Status == "Running" {
			st = StatusOK
		}
		r := NewRecord(e.Name(), linuxOS, st)
		r.StateDir = filepath.Join(s.Dir, e.Name())
		sessions = append(sessions, r)
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Session < sessions[j].Session })
	return sessions, nil
}
