package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/secrets"
	"github.com/spf13/cobra"
)

// fakeExp is the cmd-package BWExporter fake (the secrets-package fake lives in another
// package and is not importable here).
type fakeExp struct {
	data []byte
	err  error
}

func (f fakeExp) Export() ([]byte, error) { return f.data, f.err }

// stubBackupSeams replaces the four backup seams with a fully-faked, reversible (base64)
// round-trip — no real bw, no age binary, no key — and restores them after the test.
func stubBackupSeams(t *testing.T, exp secrets.BWExporter) {
	t.Helper()
	oe, oen, orc, od := bwExporter, ageEncryptor, ageRecipient, ageDecryptor
	ose, ou, ot, orp := bwSessionExporter, bwUnlock, stdinIsTerminal, readPassword
	bwExporter = exp
	bwSessionExporter = func(string) secrets.BWExporter { return exp }
	bwUnlock = func([]byte) (string, error) { return "", fmt.Errorf("bwUnlock not stubbed") }
	stdinIsTerminal = func() bool { return false }
	readPassword = func() ([]byte, error) { return nil, fmt.Errorf("readPassword not stubbed") }
	ageEncryptor = func(pt []byte, _ string) ([]byte, error) {
		enc := make([]byte, base64.StdEncoding.EncodedLen(len(pt)))
		base64.StdEncoding.Encode(enc, pt)
		return enc, nil
	}
	ageRecipient = func(string) (string, error) { return "age1fake", nil }
	ageDecryptor = func(path, _ string) ([]byte, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		dec := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(dec, data)
		if err != nil {
			return nil, err
		}
		return dec[:n], nil
	}
	t.Cleanup(func() {
		bwExporter, ageEncryptor, ageRecipient, ageDecryptor = oe, oen, orc, od
		bwSessionExporter, bwUnlock, stdinIsTerminal, readPassword = ose, ou, ot, orp
	})
}

func useRepoSensitiveDir(t *testing.T, dir string, err error) {
	t.Helper()
	old := repoSensitiveDir
	repoSensitiveDir = func() (string, error) { return dir, err }
	t.Cleanup(func() { repoSensitiveDir = old })
}

func TestSecretsBackup_HappyPath(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{data: []byte(`{"items":[{"id":"11111111-2222-3333-4444-555555555555","revisionDate":"2026-08-15T03:07:00.000Z","name":"a"}]}`)})
	useRepoSensitiveDir(t, dir, nil)

	var out bytes.Buffer
	cmd := newSecretsBackupCmd()
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("backup: %v", err)
	}

	escrow := filepath.Join(dir, "dr", secrets.EscrowFileName)
	if _, err := os.Stat(escrow); err != nil {
		t.Errorf("escrow not written under the checkout's sensitive/dr: %v", err)
	}
	if !strings.Contains(out.String(), "verified") {
		t.Errorf("expected a success message, got %q", out.String())
	}
}

func TestSecretsBackup_OutFlagOverridesDest(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{data: []byte(`{"items":[{"id":"11111111-2222-3333-4444-555555555555","revisionDate":"2026-08-15T03:07:00.000Z","name":"a"}]}`)})
	// repoSensitiveDir would fail loud; --out must bypass it entirely.
	useRepoSensitiveDir(t, "", fmt.Errorf("no checkout"))

	cmd := newSecretsBackupCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--out", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("backup --out: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, secrets.EscrowFileName)); err != nil {
		t.Errorf("escrow not written to --out dir: %v", err)
	}
}

func TestSecretsBackup_LockedBw_Errors(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{err: fmt.Errorf("bw export: vault is locked")})
	useRepoSensitiveDir(t, dir, nil)

	cmd := newSecretsBackupCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error when bw is locked")
	}
	if _, err := os.Stat(filepath.Join(dir, "dr", secrets.EscrowFileName)); !os.IsNotExist(err) {
		t.Errorf("no escrow should be written when bw is locked")
	}
}

// TestSecretsBackup_LockedBw_RemedyRerunsTheInvocation is #1647. The remedy is
// pasted verbatim, so it must re-run the command that failed. It was a fixed
// string that dropped --out, and the escrow landed in the shared checkout instead
// of the worktree the operator named.
func TestSecretsBackup_LockedBw_RemedyRerunsTheInvocation(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{err: fmt.Errorf("bw export: %w", secrets.ErrBWVaultLocked)})
	useRepoSensitiveDir(t, dir, nil)
	const session = `BW_SESSION="$(bw unlock --raw)" `

	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"default destination": {nil, session + "dotf secrets backup\n"},
		"--out survives":      {[]string{"--out", "/wt/sensitive/dr"}, session + "dotf secrets backup --out='/wt/sensitive/dr'"},
		"a value is quoted":   {[]string{"--out=/my wt/it's"}, session + `dotf secrets backup --out='/my wt/it'\''s'`},
	} {
		t.Run(name, func(t *testing.T) {
			root := New("dev", "")
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append([]string{"secrets", "backup"}, tc.args...))
			err := root.Execute()
			if err == nil || !strings.Contains(err.Error()+"\n", tc.want) {
				t.Fatalf("the remedy must re-run the failed invocation\nwant: %s\ngot:  %v", tc.want, err)
			}
		})
	}
}

func TestSecretsBackup_NoCheckout_FailsLoud(t *testing.T) {
	stubBackupSeams(t, fakeExp{data: []byte(`{"items":[]}`)})
	useRepoSensitiveDir(t, "", fmt.Errorf("no dotfiles checkout found"))

	cmd := newSecretsBackupCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected a fail-loud refusal when no checkout is found (never the deployed copy)")
	}
}

// rerunLine must render every flag kind so the line parses back to the same
// invocation: a slice flag's String() is "[a,b]", which does not.
func TestRerunLine_RendersFlagsThatParseBack(t *testing.T) {
	c := &cobra.Command{Use: "x", Run: func(*cobra.Command, []string) {}}
	var tags []string
	var apply bool
	var out string
	c.Flags().StringSliceVar(&tags, "tag", nil, "")
	c.Flags().BoolVar(&apply, "apply", false, "")
	c.Flags().StringVar(&out, "out", "", "")
	if err := c.ParseFlags([]string{"--tag", "a", "--tag", "b c", "--apply"}); err != nil {
		t.Fatal(err)
	}
	want := `x --apply='true' --tag='a' --tag='b c'`
	if got := rerunLine(c); got != want {
		t.Fatalf("want %s\ngot  %s", want, got)
	}
}

// #1008 option 3: on a terminal, a locked bw CLI is unlocked by the command
// itself. One hidden prompt, the session handed to the exporter alone, and the
// password buffer scrubbed afterwards.
func TestSecretsBackup_LockedOnATerminal_AcquiresItsOwnSession(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{err: fmt.Errorf("bw export: %w", secrets.ErrBWVaultLocked)})
	useRepoSensitiveDir(t, dir, nil)

	pw := []byte("hunter2")
	var unlocks int
	var gotPW, gotSession string
	stdinIsTerminal = func() bool { return true }
	readPassword = func() ([]byte, error) { return pw, nil }
	bwUnlock = func(p []byte) (string, error) { unlocks++; gotPW = string(p); return "sess-123", nil }
	bwSessionExporter = func(s string) secrets.BWExporter {
		gotSession = s
		return fakeExp{data: []byte(`{"items":[{"id":"11111111-2222-3333-4444-555555555555","revisionDate":"2026-08-15T03:07:00.000Z","name":"a"}]}`)}
	}

	var out, errOut bytes.Buffer
	cmd := newSecretsBackupCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("backup: %v", err)
	}
	if unlocks != 1 || gotPW != "hunter2" || gotSession != "sess-123" {
		t.Errorf("unlocks=%d pw=%q session=%q, want one unlock with the typed password and its session", unlocks, gotPW, gotSession)
	}
	if !strings.Contains(errOut.String(), "master password") {
		t.Errorf("no prompt on stderr: %q", errOut.String())
	}
	if strings.Contains(out.String()+errOut.String(), "sess-123") || strings.Contains(out.String()+errOut.String(), "hunter2") {
		t.Errorf("a credential reached the output:\n%s%s", out.String(), errOut.String())
	}
	if string(pw) != string(make([]byte, len(pw))) {
		t.Errorf("password buffer not scrubbed: %q", pw)
	}
	if _, err := os.Stat(filepath.Join(dir, "dr", secrets.EscrowFileName)); err != nil {
		t.Errorf("escrow not written: %v", err)
	}
}

// Without a terminal (a scheduler) nothing prompts: the remedy line stays.
func TestSecretsBackup_LockedWithoutATerminal_NeverPrompts(t *testing.T) {
	stubBackupSeams(t, fakeExp{err: fmt.Errorf("bw export: %w", secrets.ErrBWVaultLocked)})
	useRepoSensitiveDir(t, t.TempDir(), nil)
	readPassword = func() ([]byte, error) { t.Fatal("prompted without a terminal"); return nil, nil }

	cmd := newSecretsBackupCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `BW_SESSION="$(bw unlock --raw)"`) {
		t.Errorf("want the remedy line, got %v", err)
	}
}

// A wrong password is bw's answer, surfaced; no escrow, no retry loop.
func TestSecretsBackup_UnlockFailureSurfaces(t *testing.T) {
	dir := t.TempDir()
	stubBackupSeams(t, fakeExp{err: fmt.Errorf("bw export: %w", secrets.ErrBWVaultLocked)})
	useRepoSensitiveDir(t, dir, nil)
	stdinIsTerminal = func() bool { return true }
	readPassword = func() ([]byte, error) { return []byte("wrong"), nil }
	var unlocks int
	bwUnlock = func([]byte) (string, error) { unlocks++; return "", fmt.Errorf("Invalid master password.") }

	cmd := newSecretsBackupCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Invalid master password.") || unlocks != 1 {
		t.Errorf("err=%v unlocks=%d, want bw's message after one attempt", err, unlocks)
	}
	if _, serr := os.Stat(filepath.Join(dir, "dr", secrets.EscrowFileName)); !os.IsNotExist(serr) {
		t.Errorf("an escrow was written after a failed unlock")
	}
}
