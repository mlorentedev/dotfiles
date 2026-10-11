package hooks

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Runner runs a git subcommand and returns its stdout: the seam a caller
// outside this package drives Inspect and Apply through.
type Runner = gitRunner

// Wiring is what git's global core.hooksPath points at, judged against the
// deploy mirror.
type Wiring int

const (
	WiringUnset      Wiring = iota // no core.hooksPath: the guard does not run
	WiringMirror                   // the deploy mirror
	WiringEquivalent               // another GUARD dispatcher, a checkout under development
	WiringForeign                  // someone else's hooks, preserved and never clobbered
)

// State is what an install would change on this machine.
type State struct {
	Dest        string // the deploy mirror, <DotfilesDir>/git-hooks
	MirrorStale bool   // the mirror is absent or differs from what a deploy writes
	Wiring      Wiring
	HooksPath   string // core.hooksPath as git reports it; "" when unset
}

// Changes counts what Apply would change: the mirror, and the wiring when it
// is unset. A foreign hooksPath is left as it is, so it is not one.
func (s State) Changes() int {
	n := 0
	if s.MirrorStale {
		n++
	}
	if s.Wiring == WiringUnset {
		n++
	}
	return n
}

// Inspect reports what Install would change, and changes nothing. goos decides
// whether the executable bit is part of a current mirror: on Windows chmod is
// inert and git-for-windows runs the hooks through sh.
func Inspect(ctx context.Context, run Runner, o Options, goos string) (State, error) {
	if err := o.resolveHome(); err != nil {
		return State{}, err
	}
	if err := checkDotfilesDir(o.DotfilesDir, o.homeDir); err != nil {
		return State{}, err
	}
	if _, err := os.Stat(filepath.Join(o.Source, "pre-commit")); err != nil {
		return State{}, fmt.Errorf("source %s has no pre-commit dispatcher", o.Source)
	}
	st := State{Dest: filepath.Join(o.DotfilesDir, "git-hooks")}
	current, err := mirrorCurrent(o.Source, st.Dest, goos != "windows")
	if err != nil {
		return State{}, err
	}
	st.MirrorStale = !current
	if b, err := run(ctx, "config", "--global", "--get", "core.hooksPath"); err == nil {
		st.HooksPath = strings.TrimSpace(string(b))
	}
	st.Wiring = wiringOf(st.HooksPath, st.Dest)
	return st, nil
}

// Apply is Install through a given runner: deploy the mirror, then wire
// core.hooksPath when it is unset.
func Apply(ctx context.Context, run Runner, o Options) error {
	if err := o.resolveHome(); err != nil {
		return err
	}
	return install(ctx, run, o)
}

func (o *Options) resolveHome() error {
	if o.homeDir != "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	o.homeDir = home
	return nil
}

// wiringOf classifies a hooksPath. A path naming the mirror counts as the
// mirror before the mirror exists, so a first plan does not call it foreign.
func wiringOf(current, dest string) Wiring {
	switch {
	case current == "":
		return WiringUnset
	case samePath(current, dest), filepath.Clean(current) == filepath.Clean(dest):
		return WiringMirror
	case isGuardDispatcher(current):
		return WiringEquivalent
	}
	return WiringForeign
}

// mirrorCurrent reports whether dest already holds what deployHooks would
// write from src: the same regular files, each with the bytes copyTree writes
// (CR stripped from text), nothing more, and the entrypoints and lib/*.sh
// executable when execBits. The source itself, mirrored onto itself, is current.
func mirrorCurrent(src, dest string, execBits bool) (bool, error) {
	if same, err := sameDir(src, dest); err != nil || same {
		return same, err
	}
	want, err := treeFiles(src)
	if err != nil {
		return false, err
	}
	got, err := treeFiles(dest)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(got) != len(want) {
		return false, nil
	}
	for rel, body := range want {
		if isText(body) {
			body = stripCR(body)
		}
		have, ok := got[rel]
		if !ok || !bytes.Equal(have, body) {
			return false, nil
		}
		if execBits && wantsExec(rel) && !isExecutable(filepath.Join(dest, rel)) {
			return false, nil
		}
	}
	return true, nil
}

// treeFiles reads every regular file under root, keyed by its slash path.
func treeFiles(root string) (map[string][]byte, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		files[filepath.ToSlash(rel)] = body
		return nil
	})
	return files, err
}

// wantsExec names the files markExecutable sets the bit on.
func wantsExec(rel string) bool {
	for _, e := range entrypoints {
		if rel == e {
			return true
		}
	}
	return path.Dir(rel) == "lib" && strings.HasSuffix(rel, ".sh")
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().Perm()&0o111 == 0o111
}
