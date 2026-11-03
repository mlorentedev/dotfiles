package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/mlorentedev/dotfiles/cli/internal/vm"
)

// execSecrets checks the ids now and resolves the values later. A variable so a
// test supplies a synthetic value: nothing here may reach a real store.
var execSecrets = resolveExecSecrets

// resolveExecSecrets refuses what is not one value on one variable (ShowEntry:
// a file secret would otherwise be materialized on the host) without decrypting
// anything, and returns the resolution for the moment a guest can serve it.
func resolveExecSecrets(ids []string) (func() ([]string, error), error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	entries := make([]secrets.Entry, 0, len(ids))
	only := map[string]bool{}
	for _, id := range ids {
		e, err := reg.ShowEntry(id)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
		only[e.Var] = true
	}
	return func() ([]string, error) { return secretLoader().EnvFor(entries, only) }, nil
}

type execFlags struct {
	session, script string
	secretIDs       []string
	timeout         time.Duration
	elevated        bool
	asJSON          bool
}

// options checks everything that needs no guest; every failure is a refusal,
// exit 2. The program is what follows `--`: it is never parsed as flags, and
// prose stays out of argv (ADR-055 decision 4).
func (f execFlags) options(cmd *cobra.Command, args []string, g sessionGuest) (vm.ExecOptions, error) {
	o := vm.ExecOptions{Session: f.session, Script: f.script, Timeout: f.timeout, Elevated: f.elevated, Stdin: cmd.InOrStdin()}
	if g.os == "windows" && !stdinGiven(o.Stdin) {
		o.Stdin = nil
	}
	if err := vm.ValidSessionName(f.session); err != nil {
		return o, err
	}
	if dash := cmd.ArgsLenAtDash(); len(args) > 0 && dash != 0 {
		return o, fmt.Errorf("the program follows --: dotf vm %s exec --session <name> -- <program> [args...]", g.os)
	}
	switch {
	case (len(args) == 0) == (f.script == ""):
		return o, errors.New("give a program after -- or --script <file>, one of the two")
	case f.timeout < 0:
		return o, errors.New("--timeout must not be negative")
	}
	o.Argv = args
	if f.script != "" {
		if _, err := os.Stat(f.script); err != nil {
			return o, fmt.Errorf("--script: %w", err)
		}
	}
	if len(f.secretIDs) > 0 {
		resolve, err := execSecrets(f.secretIDs)
		if err != nil {
			return o, fmt.Errorf("--secret: %w", err)
		}
		o.Secrets = resolve
	}
	return o, nil
}

// stdinGiven tells a caller's stdin from a terminal or /dev/null, which a
// harness hands every command and which carry nothing to forward.
func stdinGiven(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return true
	}
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice == 0
}

func newVMExecCmd(g sessionGuest) *cobra.Command {
	var f execFlags
	c := &cobra.Command{
		Use:          "exec --session <name> [flags] [-- <program> [args...]]",
		Short:        "Run one command in a session, with no shell, and report it in one record",
		Long:         g.exec + "\n\n" + sessionVerbs,
		Example:      "  dotf vm " + g.os + " exec --session proof --json -- uname -a\n  dotf vm " + g.os + " exec --session proof --timeout 10m --script check",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := f.options(cmd, args, g)
			if err != nil {
				return withExitCode(2, err)
			}
			s, err := g.openFor(cmd, f.session, f.asJSON)
			if err != nil {
				return err
			}
			if f.asJSON {
				o.Stdout, o.Stderr = io.Discard, io.Discard
				return emitSession(cmd, s.Exec(cmd.Context(), o), true)
			}
			o.Stdout, o.Stderr = cmd.OutOrStdout(), cmd.ErrOrStderr()
			rec := s.Exec(cmd.Context(), o)
			if code := rec.ProcessCode(); code != 0 {
				return withExitCode(code, errors.New(rec.Summary()))
			}
			return nil
		},
	}
	sessionFlags(c, &f.session, &f.asJSON, "session name (required)")
	c.Flags().StringVar(&f.script, "script", "", "host file to copy in and run with "+g.script)
	c.Flags().StringArrayVar(&f.secretIDs, "secret", nil, "registry id or variable to forward on stdin (repeatable)")
	c.Flags().BoolVar(&f.elevated, "elevated", false, "run as an administrator, for this call only")
	c.Flags().DurationVar(&f.timeout, "timeout", 0, "stop the command after this long (default none)")
	return c
}
