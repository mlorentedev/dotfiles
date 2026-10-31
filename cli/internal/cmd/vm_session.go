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

const sessionVerbs = "A session is a named guest that `up` holds until `down`. With --json\n" +
	"stdout is exactly one record: {version, session, os, status, exit, stdout,\n" +
	"stderr, truncated, duration_ms, state_dir}. Read status and exit, never the\n" +
	"process code. Status is ok, failed, busy, unsupported, no-session, timeout or\n" +
	"stale (a session record whose guest is gone: down clears it). Exit codes: 0 ok;\n" +
	"1 failure or timeout; 2 refused before any work (busy, stale, bad flag);\n" +
	"3 no such session. Progress goes to stderr."

func addSessionVerbs(linux *cobra.Command) {
	linux.AddCommand(newVMUpCmd(), newVMDownCmd(), newVMStatusCmd(), newVMExecCmd(), newVMCpCmd())
}

func sessionFlags(c *cobra.Command, session *string, asJSON *bool, usage string) {
	c.Flags().StringVar(session, "session", "", usage)
	c.Flags().BoolVar(asJSON, "json", false, "write one JSON record to stdout")
}

// emitSession writes the record (a line with the session name when not --json)
// and turns the record's process code into the exit code.
func emitSession(cmd *cobra.Command, rec vm.Record, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		if err := enc.Encode(rec); err != nil {
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

func newVMUpCmd() *cobra.Command {
	var session string
	var asJSON bool
	c := &cobra.Command{
		Use:          "up",
		Short:        "Create a Linux guest and hold it as a named session",
		Long:         "up boots the Linux guest (Ubuntu 24.04, nothing shared with the host) and keeps\nit until `down`. Without --session the name is linux-<unix time>; it is printed.\nA name already held is busy.\n\n" + sessionVerbs,
		Example:      "  dotf vm linux up --session sshd-proof --json",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if session != "" {
				if err := vm.ValidSessionName(session); err != nil {
					return withExitCode(2, err)
				}
			}
			s, err := linuxSessions()
			if err != nil {
				return err
			}
			return emitSession(cmd, s.Up(cmd.Context(), session, cmd.ErrOrStderr(), cmd.ErrOrStderr()), asJSON)
		},
	}
	sessionFlags(c, &session, &asJSON, "session name (default linux-<unix time>)")
	return c
}

func newVMDownCmd() *cobra.Command {
	var session string
	var asJSON bool
	c := &cobra.Command{
		Use:          "down",
		Short:        "Delete a session's guest and its state",
		Long:         "down deletes the session's Lima instance and its state, and clears a stale\nsession. It acts only on a session `up` created.\n\n" + sessionVerbs,
		Example:      "  dotf vm linux down --session sshd-proof",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := vm.ValidSessionName(session); err != nil {
				return withExitCode(2, err)
			}
			s, err := linuxSessions()
			if err != nil {
				return err
			}
			return emitSession(cmd, s.Down(cmd.Context(), session, cmd.ErrOrStderr()), asJSON)
		},
	}
	sessionFlags(c, &session, &asJSON, "session name (required)")
	return c
}

func newVMStatusCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:          "status",
		Short:        "List the sessions and what the hypervisor says about each",
		Long:         "status lists every session `up` created: ok when its guest runs, stale when it\ndoes not. With --json it writes an array of records, [] when there is none; a\nfailure writes one failed record instead.\n\n" + sessionVerbs,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := linuxSessions()
			if err != nil {
				return err
			}
			list, err := s.List(cmd.Context())
			if err != nil {
				rec := vm.NewRecord("", "linux", vm.StatusFailed)
				rec.Stderr = err.Error()
				return emitSession(cmd, rec, asJSON)
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(list)
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
