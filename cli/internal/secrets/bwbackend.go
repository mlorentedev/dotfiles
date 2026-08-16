package secrets

import (
	"errors"
	"fmt"
	"strings"
)

// BWBackend is a matched read+write pair whose halves authenticate through the SAME
// Bitwarden subject, chosen once and reused for every operation in a command.
//
// It exists because of what BUG-084 (#993) actually was. The read path moved to the
// `bw serve` daemon and the write path stayed on the CLI shellout, so the two halves of
// one command asked different subjects: reads resolved fine against an unlocked daemon
// while writes failed "Vault is locked" against a CLI session nobody had. That is not a
// bug you fix once — it is a shape that recurs the moment any third path moves alone.
//
// Pinning makes it unrepresentable. A caller cannot hold a daemon reader and a shellout
// writer, because both come out of one branch of one decision. Compare the per-call
// alternative (BWFallbackReader's own policy, which re-probes on every Field call): it
// self-heals if the daemon appears mid-run, but a daemon that LOCKS between two calls of
// one command splits that command across two subjects — precisely this bug, in a
// narrower window. For a multi-step mutation like `rotate` (read old → write new → read
// back), a consistent subject matters more than self-healing: a rotation that half
// applies is worse than one that fails whole.
type BWBackend struct {
	// Reader, Writer and Syncer are always ALL daemon-backed or ALL shellout-backed.
	//
	// Syncer is here, and not left as its own var, for the reason this whole type
	// exists: sync is the third path, and a third path that moves alone reproduces the
	// bug. `rotate` writes, syncs, then reads back to prove the write took — if the
	// sync addresses a different cache than the read, that proof is meaningless and
	// the command reports a stale value as a failed rotation.
	Reader BWReader
	Writer BWWriteClient
	Syncer BWSyncer

	// Name labels the chosen subject for diagnostics ("bw serve daemon" / "bw CLI").
	Name string

	// Daemon reports whether the daemon was chosen — the single fact tests assert
	// on to prove the two halves agree (AC5).
	Daemon bool
}

// SelectBWBackend probes the daemon ONCE and returns a matched pair.
//
// The daemon is chosen only when it is both reachable and unlocked. Anything else —
// not running, running but locked, unreachable — selects the CLI shellout, which is
// still the correct choice: an operator with an ambient BW_SESSION can work that way,
// and it is the only backend that can export (see BWExport).
//
// When the shellout is selected, its errors are decorated with the remediation implied
// by the daemon state observed HERE, at selection time. That is the difference between
// "Vault is locked." — which sends the operator to `bw unlock`, a command that prints a
// session key to a shell nobody is reading and does not fix the failure — and an error
// naming `dotf secrets unlock`, which does.
// BWSyncer refreshes the local cache a backend answers reads from. Both backends have
// one and they are NOT interchangeable: the daemon caches independently of the `bw`
// CLI, so syncing one leaves the other stale. Which is why it is pinned alongside the
// reader and writer rather than chosen separately.
type BWSyncer interface {
	Sync() error
}

// BWCLISync is the shellout BWSyncer: `bw sync`, refreshing the CLI's own cache — the
// one the shellout reader and writer actually consult.
type BWCLISync struct {
	Bin string // bw binary name/path; "" → "bw"
}

func (s BWCLISync) Sync() error {
	bin := s.Bin
	if bin == "" {
		bin = "bw"
	}
	if _, err := bwRun(bin, "sync"); err != nil {
		return fmt.Errorf("bw sync: %w", err)
	}
	return nil
}

func SelectBWBackend(c BWServeClient) BWBackend {
	st, err := c.Status()
	switch {
	case err == nil && st == "unlocked":
		return BWBackend{
			Reader: BWServeReader{Client: c},
			Writer: BWServeWriter{Client: c},
			Syncer: c,
			Name:   "bw serve daemon",
			Daemon: true,
		}
	case err == nil:
		// Reachable but not unlocked ("locked", "unauthenticated", ...). The daemon
		// exists, so the remediation is to unlock the one already running.
		return shelloutBackend(fmt.Sprintf("the bw serve daemon is running but %s", st))
	default:
		return shelloutBackend("no bw serve daemon is running")
	}
}

// shelloutBackend builds the CLI-backed pair, decorating both halves with why the
// daemon was not used so a lock failure can name the fix instead of the symptom.
func shelloutBackend(daemonState string) BWBackend {
	return BWBackend{
		Reader: lockHintReader{Reader: BWGet{}, daemonState: daemonState},
		Writer: lockHintWriter{Writer: BWPut{}, daemonState: daemonState},
		Syncer: BWCLISync{},
		Name:   "bw CLI",
		Daemon: false,
	}
}

// ErrBWVaultLocked is returned when a CLI-backed operation failed because the vault is
// locked for the `bw` binary's own session — the failure BUG-084 made unfixable-looking,
// because the daemon may be unlocked at the same moment.
var ErrBWVaultLocked = errors.New("bitwarden vault is locked")

// isVaultLocked reports whether err is bw's locked-vault failure. Matched on message
// because the CLI communicates it only as stderr text ("Vault is locked.") — there is no
// exit code or structured field that distinguishes it from any other failure.
func isVaultLocked(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "vault is locked")
}

// lockHint rewrites a locked-vault error into one that names the remediation that
// actually works, given the daemon state observed at selection time. Any other error is
// returned untouched — this decorates one specific confusion, it does not editorialise.
func lockHint(err error, daemonState string) error {
	if !isVaultLocked(err) {
		return err
	}
	return fmt.Errorf("%w: %s — run `dotf secrets unlock` (not `bw unlock`, which only "+
		"prints a session key to your shell and leaves this command's vault locked): %w",
		ErrBWVaultLocked, daemonState, err)
}

// lockHintReader decorates a shellout BWReader with lockHint.
type lockHintReader struct {
	Reader      BWReader
	daemonState string
}

func (r lockHintReader) Field(item, field string) (string, error) {
	v, err := r.Reader.Field(item, field)
	return v, lockHint(err, r.daemonState)
}

// lockHintWriter decorates a shellout BWWriteClient with lockHint on all three methods.
type lockHintWriter struct {
	Writer      BWWriteClient
	daemonState string
}

func (w lockHintWriter) SetField(item, field, value string) error {
	return lockHint(w.Writer.SetField(item, field, value), w.daemonState)
}

func (w lockHintWriter) CreateItem(item, field, value, folderID string) error {
	return lockHint(w.Writer.CreateItem(item, field, value, folderID), w.daemonState)
}

func (w lockHintWriter) ResolveFolder(name string) (string, error) {
	id, err := w.Writer.ResolveFolder(name)
	return id, lockHint(err, w.daemonState)
}
