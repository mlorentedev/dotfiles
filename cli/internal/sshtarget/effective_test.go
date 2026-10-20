package sshtarget

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

// realDropIn is the drop-in the target ships, rendered for user dotf, so the
// fixtures are held to the file the owner's machines run and not to a copy.
func realDropIn(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../ssh/sshd-workstation.conf")
	if err != nil {
		t.Fatal(err)
	}
	return Target{User: "dotf", Template: b}.DropIn()
}

func sshdDump(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The two dumps are measured: OpenSSH 10.3p1 on macOS, `sshd -T -f` over a
// config whose Include reads the drop-in, and over one whose Include matches
// no file. The second is #2763: `sshd -t` passes it, the dump shows the
// defaults.
func TestCheckEffectiveTellsADropInThatWasReadFromOneThatWasNot(t *testing.T) {
	dropIn := realDropIn(t)
	if err := checkEffective(dropIn, sshdDump(t, "sshd-T-macos-10.3p1-effective.txt"), "/etc/ssh/sshd_config.d/01-workstation.conf"); err != nil {
		t.Errorf("a dump that carries the drop-in failed: %v", err)
	}
	err := checkEffective(dropIn, sshdDump(t, "sshd-T-macos-10.3p1-ignored.txt"), "/etc/ssh/sshd_config.d/01-workstation.conf")
	if err == nil {
		t.Fatal("a dump of sshd's defaults passed")
	}
	for _, want := range []string{"authenticationmethods any", "publickey", "did not read the drop-in /etc/ssh/sshd_config.d/01-workstation.conf"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

func TestCheckEffectiveNamesEachDirectiveItChecks(t *testing.T) {
	dropIn := realDropIn(t)
	good := string(sshdDump(t, "sshd-T-macos-10.3p1-effective.txt"))
	for _, c := range []struct{ line, with, want string }{
		{"passwordauthentication no", "passwordauthentication yes", "passwordauthentication yes"},
		{"kbdinteractiveauthentication no", "kbdinteractiveauthentication yes", "kbdinteractiveauthentication yes"},
		{"permitrootlogin no", "permitrootlogin prohibit-password", "permitrootlogin prohibit-password"},
		{"allowusers dotf@100.64.0.0/10\n", "", "allowusers unset"},
		{"allowusers dotf@100.64.0.0/10", "allowusers dotf@0.0.0.0/0", "allowusers dotf@0.0.0.0/0"},
	} {
		dump := strings.Replace(good, c.line, c.with, 1)
		if dump == good {
			t.Fatalf("fixture has no %q", c.line)
		}
		err := checkEffective(dropIn, []byte(dump), "drop-in")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("with %q: error = %v, want it to name %q", c.with, err, c.want)
		}
	}
}

// AllowUsers prints one line per pattern and accumulates across files, so a
// pattern the main config adds does not fail the drop-in's own.
func TestCheckEffectiveAcceptsAnExtraAllowUsersPattern(t *testing.T) {
	dump := string(sshdDump(t, "sshd-T-macos-10.3p1-effective.txt")) + "allowusers root@10.0.0.1\n"
	if err := checkEffective(realDropIn(t), []byte(dump), "drop-in"); err != nil {
		t.Errorf("an extra AllowUsers pattern failed the proof: %v", err)
	}
}

// Keywords and values are compared without case, as sshd prints them lowered
// and Windows prints the account it resolved.
func TestCheckEffectiveIgnoresCase(t *testing.T) {
	dump := "PasswordAuthentication NO\nAllowUsers DOTF@100.64.0.0/10\n"
	if err := checkEffective([]byte("PasswordAuthentication no\nAllowUsers dotf@100.64.0.0/10\n"), []byte(dump), "d"); err != nil {
		t.Error(err)
	}
}

// sshd reads `Keyword=value` in a config file and prints it as `keyword value`.
func TestCheckEffectiveReadsTheEqualsFormOfTheDropIn(t *testing.T) {
	dropIn := []byte("AllowUsers=dotf@100.64.0.0/10\nPasswordAuthentication=no\n")
	if err := checkEffective(dropIn, []byte("passwordauthentication no\nallowusers dotf@100.64.0.0/10\n"), "d"); err != nil {
		t.Error(err)
	}
	if err := checkEffective(dropIn, []byte("passwordauthentication yes\nallowusers dotf@100.64.0.0/10\n"), "d"); err == nil {
		t.Error("a dump that disagrees with the equals form passed")
	}
}

// The proof runs in the apply, where root is ready, right after the drop-in
// is validated, and a dump that disagrees undoes the whole run.
func TestApplyFailsAndUndoesWhenSSHDDidNotReadTheDropIn(t *testing.T) {
	m := newMachine(t)
	m.effective = func() []byte { return []byte("passwordauthentication yes\nallowusers\n") }
	tg := m.target()
	p, _ := tg.Inspect()
	err := tg.Apply(p)
	if err == nil || !strings.Contains(err.Error(), "did not read the drop-in") || !strings.Contains(err.Error(), "undone") {
		t.Fatalf("Apply = %v, want the unread drop-in named and the run undone", err)
	}
	if _, err := os.Stat(m.paths.DropIn); err == nil {
		t.Error("drop-in still installed after the undo")
	}
	if m.loaded {
		t.Error("sshd was enabled after a drop-in it never read")
	}
}

func TestApplyReadsTheDumpAfterValidatingTheDropIn(t *testing.T) {
	m := newMachine(t)
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err != nil {
		t.Fatal(err)
	}
	calls := slices.Clone(m.rootCalls)
	at := func(call string) int {
		return slices.IndexFunc(calls, func(c string) bool { return c == call })
	}
	if at("/usr/sbin/sshd -t") < 0 || at("/usr/sbin/sshd -T") < at("/usr/sbin/sshd -t") {
		t.Errorf("root ran %q, want sshd -t then sshd -T", calls)
	}
}

// Prove runs on a converged plan and in every headless converge's
// post-condition, where root cannot be asked for. It must stay without it.
func TestAConvergedRunDoesNotReadTheDumpAsRoot(t *testing.T) {
	m := newMachine(t)
	tg := m.target()
	p, _ := tg.Inspect()
	if err := tg.Apply(p); err != nil {
		t.Fatal(err)
	}
	m.forbidRoot = true
	again, err := tg.Inspect()
	if err != nil || !again.Converged() {
		t.Fatalf("second Inspect = %v, %v; want converged", again.Pending, err)
	}
	if err := tg.Apply(again); err != nil {
		t.Errorf("a converged Apply asked for root: %v", err)
	}
}

func TestApplyNamesAFailedDump(t *testing.T) {
	m := newMachine(t)
	m.failRoot = "/usr/sbin/sshd -T"
	tg := m.target()
	p, _ := tg.Inspect()
	err := tg.Apply(p)
	if err == nil || !strings.Contains(err.Error(), "read sshd's effective config") {
		t.Fatalf("Apply = %v, want the failed dump named", err)
	}
}

func TestSystemdEffectiveCreatesTheMissingPrivsepDir(t *testing.T) {
	l := &linux{}
	if _, err := (Systemd{Run: l.user}).Effective(l); err != nil {
		t.Fatalf("Effective = %v on a valid config with no /run/sshd", err)
	}
	want := []string{"/usr/sbin/sshd -T", "install -d -m 0755 /run/sshd", "/usr/sbin/sshd -T"}
	if !slices.Equal(l.rootCalls, want) {
		t.Errorf("root ran %q, want %q", l.rootCalls, want)
	}
}

func TestWindowsEffectiveDumpsTheConfigItValidates(t *testing.T) {
	var got []string
	root := rootFunc(func(argv ...string) ([]byte, error) {
		got = argv
		return []byte("passwordauthentication no\n"), nil
	})
	out, err := WindowsService{Config: `C:\ProgramData\ssh\sshd_config`}.Effective(root)
	if err != nil || !strings.Contains(string(out), "passwordauthentication no") {
		t.Fatalf("Effective = %q, %v", out, err)
	}
	if want := []string{"sshd", "-T", "-f", `C:\ProgramData\ssh\sshd_config`}; !slices.Equal(got, want) {
		t.Errorf("ran %q, want %q", got, want)
	}
	_, err = WindowsService{Config: "c"}.Effective(rootFunc(func(...string) ([]byte, error) { return []byte("boom"), errors.New("exit status 255") }))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("a failing sshd -T = %v, want its output", err)
	}
}
