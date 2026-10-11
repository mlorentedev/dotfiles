package env

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// LaunchctlRunner runs `launchctl <args>` and returns its stdout.
type LaunchctlRunner func(args ...string) (string, error)

// ExecLaunchctl runs the real launchctl. stderr is left out of the result:
// callers read stdout as the answer.
func ExecLaunchctl(args ...string) (string, error) {
	out, err := exec.Command("launchctl", args...).Output() //nolint:gosec // fixed binary, args built by dotf
	return string(out), err
}

// LaunchdUserEnv is the macOS user scope (#2013 S3): the environment of the
// user's launchd session, which every app launched from the Dock, Finder or
// Spotlight inherits. A shell reads paths.sh; a GUI app reads only this.
//
// The scope is volatile: logout and reboot clear it. LaunchAgentPlist is what
// makes it persistent, by running `dotf env persist` at every login. So the
// ownership marker bounds the sweep within one session, and a fresh session
// has nothing left to sweep.
//
// The store has no build tag, so its tests run on every OS; only
// NewUserEnvStore on darwin hands it the real launchctl.
type LaunchdUserEnv struct{ Run LaunchctlRunner }

// Get reads name. `launchctl getenv` exits 0 and prints nothing for an unset
// name, so an empty value reads as absent; Set refuses one to keep that true.
func (s LaunchdUserEnv) Get(name string) (string, bool, error) {
	out, err := s.Run("getenv", name)
	if err != nil {
		return "", false, fmt.Errorf("launchctl getenv %s: %w", name, err)
	}
	v := strings.TrimRight(out, "\n")
	return v, v != "", nil
}

// ErrEmptyLaunchdValue is returned by Set for an empty value, which launchd
// cannot hold apart from an unset name.
var ErrEmptyLaunchdValue = errors.New("launchd cannot store an empty value apart from an unset name")

func (s LaunchdUserEnv) Set(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s: %w", name, ErrEmptyLaunchdValue)
	}
	if _, err := s.Run("setenv", name, value); err != nil {
		return fmt.Errorf("launchctl setenv %s: %w", name, err)
	}
	return nil
}

// Delete succeeds on an absent name, as UserEnvStore requires: `launchctl
// unsetenv` exits 0 there (measured on macOS 27).
func (s LaunchdUserEnv) Delete(name string) error {
	if _, err := s.Run("unsetenv", name); err != nil {
		return fmt.Errorf("launchctl unsetenv %s: %w", name, err)
	}
	return nil
}

// LaunchAgentLabel names the agent that re-applies the user scope at login.
const LaunchAgentLabel = "com.github.mlorentedev.dotf.env-persist"

// LaunchAgentPath is where the agent's plist lives for home.
func LaunchAgentPath(home string) string {
	return filepath.Join(home, "Library", "LaunchAgents", LaunchAgentLabel+".plist")
}

// LaunchAgentPlist runs `<home>/.local/bin/dotf env persist` once at every
// login. It names the installed dotf, the pinned release, and never the
// running binary: a `go run` build lives in a temp dir that outlives nothing.
// launchd expands no `~`, so every path is absolute.
func LaunchAgentPlist(home string) []byte {
	// launchd reads these on macOS, so they are POSIX paths on every OS that
	// renders them, never filepath's separator.
	dotf := path.Join(home, ".local", "bin", "dotf")
	log := path.Join(home, "Library", "Logs", "dotf-env-persist.log")
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + LaunchAgentLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + xmlEscape(dotf) + `</string>
		<string>env</string>
		<string>persist</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>` + xmlEscape(log) + `</string>
	<key>StandardErrorPath</key>
	<string>` + xmlEscape(log) + `</string>
</dict>
</plist>
`)
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
