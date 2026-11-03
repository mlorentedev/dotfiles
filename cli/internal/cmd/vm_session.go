package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

// linuxSessions builds the Linux session store. A variable so a test points it
// at a fake limactl: a refusal test must never reach the hypervisor (AC9).
var linuxSessions = func() (vm.LinuxSessions, error) {
	if _, err := exec.LookPath("limactl"); err != nil {
		return vm.LinuxSessions{}, withExitCode(2, errors.New("limactl is not on PATH: run `dotf converge --only tools`"))
	}
	state, err := env.StateDir()
	if err != nil {
		return vm.LinuxSessions{}, err
	}
	return vm.LinuxSessions{
		Dir: filepath.Join(state, "vm", "linux", "sessions"),
		L:   execLimactl,
		Now: time.Now,
	}, nil
}

// windowsSessions builds the Windows session store; a variable for the same
// reason. `down` and `status` reach it too, so a missing QEMU refuses them.
var windowsSessions = func() (vm.Session, error) {
	host, firmware, _, layout, err := windowsGuest()
	if err != nil {
		return nil, err
	}
	return vm.WindowsSessions{
		H:       host,
		Machine: vm.Machine{Layout: layout, Firmware: firmware, CPUs: 4, MemoryMB: 8192},
		Agent:   10 * time.Minute,
		Poll:    500 * time.Millisecond,
		Grace:   15 * time.Second,
		Now:     time.Now,
	}, nil
}

// sessionGuest is what the shared verbs need to know about a guest.
type sessionGuest struct {
	os   string
	open func() (vm.Session, error)
	up   string // `up`'s description
	down string // `down`'s description
	exec string // `exec`'s description
	// script names what runs --script, for the flag's help.
	script string
	// example is one guest path, for cp's help.
	example string
}

func linuxGuest() sessionGuest {
	return sessionGuest{
		os: "linux",
		open: func() (vm.Session, error) {
			s, err := linuxSessions()
			if err != nil {
				return nil, err
			}
			return s, nil
		},
		up:   "up boots the Linux guest (Ubuntu 24.04, nothing shared with the host) and keeps\nit until `down`. Without --session the name is linux-<unix time>; it is printed.\nA name already held is busy.",
		down: "down deletes the session's Lima instance and its state, and clears a stale\nsession. It acts only on a session `up` created.",
		exec: "exec runs the program after -- directly, with no shell, or --script <file>\n" +
			"copied in and run by sh. Its stdin is forwarded. --elevated runs it under sudo.\n\n" +
			"--secret <id> resolves a registry value and writes it on the guest's stdin, as\n" +
			"the variable the registry names; it is never in argv or on the guest's disk,\n" +
			"and the guest's output is masked for it. A value the guest got elsewhere is\n" +
			"not masked, nor is piped stdin. A guest command that exits non-zero is status\n" +
			"ok with its code in exit, and dotf exits 1. A code of 255 can also be ssh\n" +
			"losing the guest. --json bounds each stream at 1 MiB (truncated says so);\n" +
			"without it stdout and stderr are the guest's.",
		script:  "sh",
		example: "/tmp/report.json",
	}
}

func windowsSessionGuest() sessionGuest {
	return sessionGuest{
		os:   "windows",
		open: func() (vm.Session, error) { return windowsSessions() },
		up:   "up boots a throwaway overlay of clean-base and keeps it until `down`. Without\n--session the name is windows-<unix time>; it is printed. The host runs one\nWindows guest, so a second up is busy whatever its name.",
		down: "down stops the session's guest and deletes its overlay and state, and clears a\nstale session. clean-base never changes. It acts only on a session `up` created;\n`dotf vm windows stop` is for a guest `--keep` left running.",
		exec: "exec runs the program after -- directly, with no shell, or --script <file>\n" +
			"copied in and run by PowerShell, as the guest's test user. The output arrives\n" +
			"when the command ends, through a scheduled task and polling, so a call is\n" +
			"slower than on Linux; duration_ms shows it.\n\n" +
			"The guest agent has no stdin, so piped stdin is unsupported: `cp` the data in.\n" +
			"--secret is unsupported too: a Windows session receives no credential. A guest\n" +
			"command that exits non-zero is status ok with its code in exit, and dotf exits\n" +
			"1. On --timeout the guest kills the command and its children. --json bounds\n" +
			"each stream at 1 MiB (truncated says so).\n\n" +
			"--elevated runs in a high-integrity token. UAC consents without a prompt in\n" +
			"this guest, so a pass proves what an administrator can do, not that a real\n" +
			"machine would not stop at a prompt.",
		script:  "PowerShell",
		example: `C:\Users\Public\report.json`,
	}
}

const sessionVerbs = "A session is a named guest that `up` holds until `down`. With --json\n" +
	"stdout is exactly one record: {version, session, os, status, exit, stdout,\n" +
	"stderr, truncated, duration_ms, state_dir}. Read status and exit, never the\n" +
	"process code. Status is ok, failed, busy, unsupported, no-session, timeout or\n" +
	"stale (a session record whose guest is gone: down clears it). Exit codes: 0 ok;\n" +
	"1 failure or timeout; 2 refused before any work (busy, stale, bad flag);\n" +
	"3 no such session. Progress goes to stderr."

func addSessionVerbs(c *cobra.Command, g sessionGuest) {
	c.AddCommand(newVMUpCmd(g), newVMDownCmd(g), newVMStatusCmd(g), newVMExecCmd(g), newVMCpCmd(g))
}

func sessionFlags(c *cobra.Command, session *string, asJSON *bool, usage string) {
	c.Flags().StringVar(session, "session", "", usage)
	c.Flags().BoolVar(asJSON, "json", false, "write one JSON record to stdout")
}

// openFor opens the guest's session store. One that cannot open still owes
// --json its one record (ADR-055 decision 7): a failed one, while the exit code
// stays the one the open error carries.
func (g sessionGuest) openFor(cmd *cobra.Command, session string, asJSON bool) (vm.Session, error) {
	s, err := g.open()
	if err != nil && asJSON {
		rec := vm.NewRecord(session, g.os, vm.StatusFailed)
		rec.Stderr = err.Error()
		if werr := writeSessionJSON(cmd, rec); werr != nil {
			// Joined, so the open error's exit code still decides.
			return nil, errors.Join(err, werr)
		}
	}
	return s, err
}

// writeSessionJSON writes v on one line of stdout, with no HTML escaping.
func writeSessionJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// emitSession writes the record (a line with the session name when not --json)
// and turns the record's process code into the exit code.
func emitSession(cmd *cobra.Command, rec vm.Record, asJSON bool) error {
	if asJSON {
		if err := writeSessionJSON(cmd, rec); err != nil {
			return err
		}
	} else if rec.Status == vm.StatusOK {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), rec.Session)
	}
	if code := rec.ProcessCode(); code != 0 {
		return withExitCode(code, errors.New(rec.Summary()))
	}
	return nil
}

func newVMUpCmd(g sessionGuest) *cobra.Command {
	var session string
	var asJSON bool
	c := &cobra.Command{
		Use:          "up",
		Short:        "Create a guest and hold it as a named session",
		Long:         g.up + "\n\n" + sessionVerbs,
		Example:      "  dotf vm " + g.os + " up --session sshd-proof --json",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if session != "" {
				if err := vm.ValidSessionName(session); err != nil {
					return withExitCode(2, err)
				}
			}
			s, err := g.openFor(cmd, session, asJSON)
			if err != nil {
				return err
			}
			return emitSession(cmd, s.Up(cmd.Context(), session, cmd.ErrOrStderr(), cmd.ErrOrStderr()), asJSON)
		},
	}
	sessionFlags(c, &session, &asJSON, "session name (default "+g.os+"-<unix time>)")
	return c
}

func newVMDownCmd(g sessionGuest) *cobra.Command {
	var session string
	var asJSON bool
	c := &cobra.Command{
		Use:          "down",
		Short:        "Delete a session's guest and its state",
		Long:         g.down + "\n\n" + sessionVerbs,
		Example:      "  dotf vm " + g.os + " down --session sshd-proof",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := vm.ValidSessionName(session); err != nil {
				return withExitCode(2, err)
			}
			s, err := g.openFor(cmd, session, asJSON)
			if err != nil {
				return err
			}
			return emitSession(cmd, s.Down(cmd.Context(), session, cmd.ErrOrStderr()), asJSON)
		},
	}
	sessionFlags(c, &session, &asJSON, "session name (required)")
	return c
}

func newVMStatusCmd(g sessionGuest) *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:          "status",
		Short:        "List the sessions and what the hypervisor says about each",
		Long:         "status lists every session `up` created: ok when its guest runs, stale when it\ndoes not. With --json it writes an array of records, [] when there is none; a\nfailure writes one failed record instead.\n\n" + sessionVerbs,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := g.openFor(cmd, "", asJSON)
			if err != nil {
				return err
			}
			list, err := s.List(cmd.Context())
			if err != nil {
				rec := vm.NewRecord("", g.os, vm.StatusFailed)
				rec.Stderr = err.Error()
				return emitSession(cmd, rec, asJSON)
			}
			if asJSON {
				return writeSessionJSON(cmd, list)
			}
			for _, r := range list {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", r.Session, r.Status)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "write a JSON array of records to stdout")
	return c
}
