// Package identity is the part of a machine converge cannot produce: the age
// key, the Bitwarden session, the GitHub login and the knowledge vault (#2013
// D10, D12). Each is restored once per machine, in the order of
// docs/runbooks/guide-new-machine.md, because each step needs the one before.
//
// doctor cites the Restore texts when --scope machine skips an identity check;
// the guide walks the same chain on a terminal and prints it everywhere else.
// The package decides nothing about how a fact is read or a command is run:
// the caller injects both, so the chain is tested without a terminal.
package identity

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
)

// What each identity check needs, as doctor's SKIP line under --scope machine
// names it. The step numbers are the runbook's.
const (
	RestoreAge   = "restore the age key from the offline backup (docs/runbooks/guide-new-machine.md, step 1)"
	RestoreBW    = "run `bw login`, then `dotf secrets unlock` (docs/runbooks/guide-new-machine.md, step 2)"
	RestoreGH    = "run `gh auth login` (docs/runbooks/guide-new-machine.md, step 3)"
	RestoreVault = "clone the knowledge vault (docs/runbooks/guide-new-machine.md, step 4)"
)

// Facts is what is restored already. Every field is read locally; none needs
// a secret's value.
type Facts struct {
	AgeKey     bool // the age identity file exists
	BWLoggedIn bool // bw holds a login (locked or not)
	BWUnlocked bool // the bw serve daemon answers unlocked
	GHLoggedIn bool // gh holds a stored login for github.com
	Vault      bool // the vault checkout exists
}

// Step is one link of the chain. Run is what the guide runs for it, in order;
// a step with no Run is done by hand (Manual says how) and the guide waits.
// After runs once the step is done and only reports: what it finds is not
// this chain's to fix, so it never stops it. Runbook is the step of
// docs/runbooks/guide-new-machine.md it belongs to, the number doctor's SKIP
// lines cite too; the unlock shares step 2 with the login.
type Step struct {
	Name    string
	Runbook int
	Why     string
	Manual  string
	Run     [][]string
	After   [][]string
	Done    func(Facts) bool
}

// Paths are where the restored identity lands on this machine.
type Paths struct {
	AgeKey   string // the age identity file
	Vault    string // the knowledge vault checkout
	VaultURL string // where the vault is cloned from
	Dotf     string // the dotf to run the secrets and converge steps with
}

// DefaultVaultURL is where the knowledge vault is cloned from (the runbook's
// step 4).
const DefaultVaultURL = "https://github.com/mlorentedev/knowledge.git"

// Steps is the restore chain in the runbook's order.
func Steps(p Paths) []Step {
	vault := Step{
		Name:    "knowledge vault",
		Runbook: 4,
		Why:     "the agents' memory, and what doctor's vault checks read; git-config first wires gh's credential helper the private clone authenticates through",
		Run:     [][]string{{p.Dotf, "converge", "--only", "git-config"}, {"git", "clone", p.VaultURL, p.Vault}},
		Done:    func(f Facts) bool { return f.Vault },
	}
	if p.Vault == "" {
		// Never a clone into the working directory: say what is missing.
		vault.Run = nil
		vault.Manual = "VAULT_PATH resolves to nothing on this machine: set it (machine.json, ADR-025), run `dotf converge --only git-config`, then clone " + p.VaultURL + " there"
	}
	return []Step{
		{
			Name:    "age key",
			Runbook: 1,
			Why:     "it decrypts the offline floor; nothing below starts without it",
			Manual:  "restore it from the offline backup to " + p.AgeKey + " (docs/runbooks/guide-secrets-governance.md, § RECOVER, step 1)",
			Done:    func(f Facts) bool { return f.AgeKey },
		},
		{
			Name:    "Bitwarden login",
			Runbook: 2,
			Why:     "every `dotf secrets` read resolves through Bitwarden (ADR-028)",
			// --quiet keeps the session key bw prints on stdout off the
			// terminal; its prompts are on stderr.
			Run:  [][]string{{"bw", "login", "--quiet"}},
			Done: func(f Facts) bool { return f.BWLoggedIn },
		},
		{
			Name:    "Bitwarden unlock",
			Runbook: 2,
			Why:     "the bw serve daemon every terminal shares; then each registry secret is resolved, no value printed",
			Run:     [][]string{{p.Dotf, "secrets", "unlock"}},
			// A secret missing from the store is a gap in the registry, not in
			// this machine's identity: reported, never a stop.
			After: [][]string{{p.Dotf, "secrets", "verify"}},
			Done:  func(f Facts) bool { return f.BWUnlocked },
		},
		{
			Name:    "GitHub login",
			Runbook: 3,
			Why:     "the vault is a private repository, cloned through gh's credential helper, which git-config wires only once gh holds a login",
			Run:     [][]string{{"gh", "auth", "login"}},
			Done:    func(f Facts) bool { return f.GHLoggedIn },
		},
		vault,
	}
}

// Missing reports whether any step is still to do.
func Missing(steps []Step, f Facts) bool {
	for _, s := range steps {
		if !s.Done(f) {
			return true
		}
	}
	return false
}

// Plan prints the chain and what is left of it, and runs nothing.
func Plan(w io.Writer, steps []Step, f Facts) {
	if !Missing(steps, f) {
		_, _ = fmt.Fprintln(w, "identity: restored (age key, Bitwarden, GitHub login, vault)")
		return
	}
	_, _ = fmt.Fprintln(w, "identity: not restored yet; on a terminal, `dotf identity restore` walks you through it:")
	for _, s := range steps {
		if s.Done(f) {
			_, _ = fmt.Fprintf(w, "  [ OK ] %s\n", label(s))
			continue
		}
		_, _ = fmt.Fprintf(w, "  [TODO] %s: %s\n", label(s), how(s))
	}
	_, _ = fmt.Fprintln(w, "  then: dotf converge, then dotf doctor (docs/runbooks/guide-new-machine.md)")
}

// label names a step with the runbook step it belongs to, so the guide and
// doctor's SKIP lines count the same way.
func label(s Step) string {
	return fmt.Sprintf("%s (runbook step %d)", s.Name, s.Runbook)
}

func how(s Step) string {
	if s.Run == nil {
		return s.Manual
	}
	var cmds []string
	for _, argv := range slices.Concat(s.Run, s.After) {
		cmds = append(cmds, strings.Join(argv, " "))
	}
	return strings.Join(cmds, ", then ")
}

// Terminal is the guide's conversation: the person's answers and where the
// guide speaks.
type Terminal struct {
	In  io.Reader
	Out io.Writer
}

// Exec runs one command attached to the terminal.
type Exec func(argv []string) error

// Guide walks the chain on a terminal. A step already done is passed; for one
// that is not, it says why the step matters and what it runs, then waits:
// Enter runs it (or, for a manual step, means it is done), `s` skips it. A
// skipped, failed or still-missing step stops the guide, because every later
// step needs it. It never returns an error: the identity is the person's to
// restore, and whatever converge reported stands.
func Guide(t Terminal, steps []Step, probe func() Facts, run Exec) {
	in := bufio.NewReader(t.In)
	f := probe()
	if !Missing(steps, f) {
		Plan(t.Out, steps, f)
		return
	}
	_, _ = fmt.Fprintln(t.Out, "identity: restoring what converge cannot (docs/runbooks/guide-new-machine.md)")
	for _, s := range steps {
		if s.Done(f) {
			_, _ = fmt.Fprintf(t.Out, "  [ OK ] %s\n", label(s))
			continue
		}
		_, _ = fmt.Fprintf(t.Out, "  %s: %s\n     %s\n", label(s), s.Why, how(s))
		if !confirm(t.Out, in, s) {
			stop(t.Out, s, "skipped")
			return
		}
		for _, argv := range s.Run {
			if err := run(argv); err != nil {
				stop(t.Out, s, fmt.Sprintf("`%s` failed: %v", strings.Join(argv, " "), err))
				return
			}
		}
		if f = probe(); !s.Done(f) {
			stop(t.Out, s, "still not done")
			return
		}
		_, _ = fmt.Fprintf(t.Out, "     [ OK ] %s\n", s.Name)
		for _, argv := range s.After {
			if err := run(argv); err != nil {
				_, _ = fmt.Fprintf(t.Out, "     [WARN] `%s` failed: %v; the chain goes on, since nothing below needs it\n", strings.Join(argv, " "), err)
			}
		}
	}
	_, _ = fmt.Fprintln(t.Out, "identity: restored. Next: dotf converge, then dotf doctor")
}

// confirm reads one answer. End of input is a skip, never a hang.
func confirm(w io.Writer, in *bufio.Reader, s Step) bool {
	prompt := "Enter to run it, s to skip: "
	if s.Run == nil {
		prompt = "Enter once it is done, s to skip: "
	}
	_, _ = fmt.Fprint(w, "     "+prompt)
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		_, _ = fmt.Fprintln(w)
		return false
	}
	return !strings.EqualFold(strings.TrimSpace(line), "s")
}

func stop(w io.Writer, s Step, why string) {
	_, _ = fmt.Fprintf(w, "     %s: %s. The steps after it need it, so the guide stops here; `dotf identity restore` resumes it.\n", s.Name, why)
}
