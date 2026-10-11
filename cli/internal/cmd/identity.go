package cmd

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/gitconfig"
	"github.com/mlorentedev/dotfiles/cli/internal/identity"
	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
)

func newIdentityCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "identity",
		Short: "The part of a machine converge cannot produce: age key, Bitwarden, GitHub login, vault",
	}
	var plan bool
	restore := &cobra.Command{
		Use:   "restore",
		Short: "Walk the identity restore chain on a terminal; print it elsewhere",
		Long: `restore walks the identity restore of docs/runbooks/guide-new-machine.md in
order: the age key, the Bitwarden login, the bw serve unlock (then every
registry secret is verified, no value printed), the GitHub login and the
knowledge vault. A step already done is passed. For one that is not, it says why
the step matters and what it runs, and waits: Enter runs it, s skips it. A
skipped or failed step stops the chain, since every later step needs it; run
restore again to resume.

The age key is the one step it cannot run: it names where the key comes from
and waits until the file exists.

Without a terminal (CI, a scheduled run) it prints what is left and asks
nothing. It always exits 0: the identity is yours to restore, and doctor reports
it. dotf converge ends with the same walk, so a new machine needs no second
command.`,
		Example:      "  dotf identity restore\n  dotf identity restore --plan",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			identityEpilogue(cmd.OutOrStdout(), plan)
			return nil
		},
	}
	restore.Flags().BoolVar(&plan, "plan", false, "print what is left and run nothing")
	root.AddCommand(restore)
	return root
}

// identityEpilogue walks the identity restore on a terminal and prints what is
// left everywhere else (#2013 D12). A plan never walks it. It returns nothing,
// so it cannot change the result of the converge it ends.
func identityEpilogue(w io.Writer, plan bool) {
	steps := identity.Steps(identityPaths())
	if plan {
		identity.Plan(w, steps, identityFacts())
		return
	}
	in, ok := identityTerminal()
	if !ok {
		identity.Plan(w, steps, identityFacts())
		return
	}
	defer func() {
		if in != os.Stdin {
			_ = in.Close()
		}
	}()
	identity.Guide(identity.Terminal{In: in, Out: w}, steps, identityFacts, identityExec(in))
}

func identityPaths() identity.Paths {
	self, err := os.Executable()
	if err != nil {
		self = "dotf"
	}
	return identity.Paths{
		AgeKey:   ageKeyPath(),
		Vault:    env.ResolvePath("VAULT_PATH"),
		VaultURL: identity.DefaultVaultURL,
		Dotf:     self,
	}
}

// identityTerminal is the terminal the guide talks on, ok false when there is
// none. stdout must be one. Input is stdin when that is a terminal too, else
// the controlling terminal: `curl … | bash` leaves stdin a pipe while the
// person is still at the keyboard. Windows has no /dev/tty; there only a
// terminal stdin counts.
var identityTerminal = func() (*os.File, bool) {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return nil, false
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return os.Stdin, true
	}
	if runtime.GOOS == "windows" {
		return nil, false
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, false
	}
	if !term.IsTerminal(int(tty.Fd())) {
		_ = tty.Close()
		return nil, false
	}
	return tty, true
}

// identityExec runs a step's command on the terminal: bw and gh prompt on it.
var identityExec = func(in *os.File) identity.Exec {
	return func(argv []string) error {
		c := exec.Command(argv[0], argv[1:]...) //nolint:gosec // the chain's fixed commands
		c.Stdin, c.Stdout, c.Stderr = in, os.Stdout, os.Stderr
		return c.Run()
	}
}

// identityFacts reads what is restored. Every read is local, and none prints
// or keeps a secret: the gh login is read by exit status only.
var identityFacts = func() identity.Facts {
	daemon, _ := secrets.BWServeClient{}.Status()
	vault := env.ResolvePath("VAULT_PATH")
	return identity.Facts{
		AgeKey:     isRegularFile(ageKeyPath()),
		BWLoggedIn: daemon == "locked" || daemon == "unlocked" || bwHoldsALogin(),
		BWUnlocked: daemon == "unlocked",
		// The stored login, as converge's git-config step reads it: a token in
		// the environment is not one the credential helper can use (#2319).
		GHLoggedIn: ghStoredLogin(),
		Vault:      vault != "" && isDirectory(filepath.Join(vault, ".git")),
	}
}

func ghStoredLogin() bool {
	_, err := gitconfig.StoredLoginRunner("gh", "auth", "status", "--hostname", "github.com")
	return err == nil
}

// bwHoldsALogin asks bw itself when no daemon answers. Its status is local
// state; only the status field is read.
func bwHoldsALogin() bool {
	out, err := exec.Command("bw", "status").Output()
	if err != nil {
		return false
	}
	var st struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(out, &st) == nil && (st.Status == "locked" || st.Status == "unlocked")
}

func isRegularFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}
