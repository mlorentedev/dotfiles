package schedule

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LaunchdPlist renders job as a launchd agent's property list. launchd expands
// no `~`, so every path in job is absolute, in POSIX spelling on every OS that
// renders it. The key order is fixed: env-persist compares its installed agent
// with this output byte for byte.
func LaunchdPlist(job Job) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + xmlEscape(job.Names.Launchd) + `</string>
	<key>ProgramArguments</key>
	<array>
`)
	for _, a := range append([]string{job.Program}, job.Args...) {
		b.WriteString("\t\t<string>" + xmlEscape(a) + "</string>\n")
	}
	b.WriteString("\t</array>\n")
	if len(job.Env) > 0 {
		b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
		for _, v := range job.Env {
			b.WriteString("\t\t<key>" + xmlEscape(v.Name) + "</key>\n\t\t<string>" + xmlEscape(v.Value) + "</string>\n")
		}
		b.WriteString("\t</dict>\n")
	}
	if job.RunAtLoad {
		b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	}
	if c := job.Daily; c != nil {
		b.WriteString("\t<key>StartCalendarInterval</key>\n\t<dict>\n" +
			"\t\t<key>Hour</key>\n\t\t<integer>" + strconv.Itoa(c.Hour) + "</integer>\n" +
			"\t\t<key>Minute</key>\n\t\t<integer>" + strconv.Itoa(c.Minute) + "</integer>\n\t</dict>\n")
	}
	if job.Log != "" {
		b.WriteString("\t<key>StandardOutPath</key>\n\t<string>" + xmlEscape(job.Log) + "</string>\n" +
			"\t<key>StandardErrorPath</key>\n\t<string>" + xmlEscape(job.Log) + "</string>\n")
	}
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String())
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Launchd is the darwin backend: a per-user agent in Dir, loaded into the
// gui/<UID> domain. Every agent under one of dotf's labels is dotf's, so a
// present agent is always managed.
type Launchd struct {
	Run Runner
	UID int
	Dir string // ~/Library/LaunchAgents
}

type launchdState struct {
	exists, stale, loaded bool
}

func (l Launchd) path(job Job) string { return filepath.Join(l.Dir, job.Names.Launchd+".plist") }
func (l Launchd) domain() string      { return "gui/" + strconv.Itoa(l.UID) }
func (l Launchd) service(job Job) string {
	return l.domain() + "/" + job.Names.Launchd
}

func (l Launchd) read(job Job) (launchdState, error) {
	have, err := os.ReadFile(l.path(job))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return launchdState{}, err
	}
	st := launchdState{exists: err == nil, stale: !bytes.Equal(have, LaunchdPlist(job))}
	st.loaded, err = l.loaded(job)
	return st, err
}

// launchdUnknown is the status `launchctl print` exits with for a service
// launchd does not know.
const launchdUnknown = 113

// loaded reports whether launchd knows the job. Only launchdUnknown means it
// does not: reading any other failure as "not loaded" would delete the plist
// of a running agent on opt-out and leave it running.
func (l Launchd) loaded(job Job) (bool, error) {
	_, err := l.Run("print", l.service(job))
	var exit interface{ ExitCode() int }
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exit) && exit.ExitCode() == launchdUnknown:
		return false, nil
	}
	return false, fmt.Errorf("launchctl print %s: %w", l.service(job), err)
}

func (l Launchd) Inspect(job Job) (State, error) {
	st, err := l.read(job)
	if err != nil {
		return State{}, err
	}
	out := State{Present: st.exists || st.loaded, Managed: true, Where: l.service(job)}
	if st.stale {
		out.Pending = append(out.Pending, "write "+l.path(job))
	}
	if st.stale || !st.loaded {
		out.Pending = append(out.Pending, "load "+l.service(job))
	}
	return out, nil
}

// Install writes the plist and has launchd load it. A loaded agent whose plist
// changed is booted out first: bootstrap refuses a loaded label, and launchd
// keeps the definition it read at load time. The bootout comes before the
// write, so a failed one leaves the old plist behind and the next run retries.
func (l Launchd) Install(job Job) error {
	st, err := l.read(job)
	if err != nil {
		return err
	}
	if st.stale && st.loaded {
		if _, err := l.Run("bootout", l.service(job)); err != nil {
			return fmt.Errorf("launchctl bootout %s: %w", l.service(job), err)
		}
	}
	if st.stale {
		if err := os.MkdirAll(l.Dir, 0o755); err != nil { //nolint:gosec // ~/Library/LaunchAgents is world-readable by convention
			return err
		}
		if err := writeAtomic(l.path(job), LaunchdPlist(job), 0o644); err != nil {
			return err
		}
	}
	if st.stale || !st.loaded {
		if _, err := l.Run("bootstrap", l.domain(), l.path(job)); err != nil {
			return fmt.Errorf("launchctl bootstrap %s %s: %w", l.domain(), l.path(job), err)
		}
	}
	return nil
}

// Remove boots the agent out and deletes its plist.
func (l Launchd) Remove(job Job) error {
	st, err := l.read(job)
	if err != nil {
		return err
	}
	if st.loaded {
		if _, err := l.Run("bootout", l.service(job)); err != nil {
			return fmt.Errorf("launchctl bootout %s: %w", l.service(job), err)
		}
	}
	return removeIfExists(l.path(job))
}

// writeAtomic writes data to path through a sibling temp file and a rename, so
// a reader never sees half a definition.
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
