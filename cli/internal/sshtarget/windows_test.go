package sshtarget

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// recorder is a Runner and a Root that runs nothing: it records each argv and
// answers from a table keyed by the argv's first words.
type recorder struct {
	calls   []string
	answers map[string]answer
}

type answer struct {
	out string
	err error
}

func (r *recorder) run(argv ...string) ([]byte, error) {
	line := strings.Join(argv, " ")
	r.calls = append(r.calls, line)
	for prefix, a := range r.answers {
		if strings.HasPrefix(line, prefix) {
			return []byte(a.out), a.err
		}
	}
	return nil, nil
}

func (r *recorder) Ready() error                       { return nil }
func (r *recorder) Run(argv ...string) ([]byte, error) { return r.run(argv...) }

func (r *recorder) scripts() []string {
	var s []string
	for _, c := range r.calls {
		if strings.HasPrefix(c, "powershell.exe") && !strings.Contains(c, "ConvertTo-Json") {
			s = append(s, c)
		}
	}
	return s
}

func TestElevatedReadyReadsTheIntegrityLevelSID(t *testing.T) {
	// A Spanish Windows names the label in Spanish; the SID is the same.
	elevated := "Etiqueta obligatoria\\Nivel obligatorio alto  Etiqueta  S-1-16-12288\n"
	filtered := "Etiqueta obligatoria\\Nivel obligatorio medio Etiqueta  S-1-16-8192\n"
	for _, tc := range []struct {
		out  string
		want error
	}{{elevated, nil}, {filtered, ErrNotElevated}} {
		r := &recorder{answers: map[string]answer{"whoami /groups": {out: tc.out}}}
		if err := (Elevated{Exec: r.run}).Ready(); !errors.Is(err, tc.want) {
			t.Errorf("Ready() = %v, want %v", err, tc.want)
		}
	}
}

func TestElevatedInstallAdminOnlySetsTheACLBeforeTheRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh", "administrators_authorized_keys")
	var seenBeforeRename bool
	r := &recorder{}
	e := Elevated{Exec: func(argv ...string) ([]byte, error) {
		if argv[0] == "icacls" {
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				seenBeforeRename = true
			}
		}
		return r.run(argv...)
	}}
	if err := e.InstallAdminOnly(path, []byte(testKey+"\n")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != testKey+"\n" {
		t.Errorf("content = %q", got)
	}
	acl := strings.Join(r.calls, "|")
	for _, want := range []string{"/setowner *S-1-5-32-544", "/inheritance:r /grant:r *S-1-5-32-544:F /grant:r *S-1-5-18:F"} {
		if !strings.Contains(acl, want) {
			t.Errorf("icacls calls %q lack %q", acl, want)
		}
	}
	if !seenBeforeRename {
		t.Error("the ACL must be set on the temporary file, before it replaces the real one")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "ssh", ".*.tmp")); len(left) > 0 {
		t.Errorf("temporary files left behind: %v", left)
	}
}

func TestWindowsServiceReadsSCNumbersNotNames(t *testing.T) {
	// sc.exe on a Spanish Windows: the names are translated, the codes are not.
	qcAuto := "[SC] QueryServiceConfig CORRECTO\n\nNOMBRE_SERVICIO: sshd\n        START_TYPE         : 2   AUTO_START\n"
	qcManual := "        START_TYPE         : 3   DEMAND_START\n"
	running := "        STATE              : 4  RUNNING\n"
	stopped := "        STATE              : 1  STOPPED\n"
	for _, tc := range []struct {
		qc, query string
		want      bool
	}{{qcAuto, running, true}, {qcManual, running, false}, {qcAuto, stopped, false}} {
		r := &recorder{answers: map[string]answer{"sc.exe qc": {out: tc.qc}, "sc.exe query": {out: tc.query}}}
		got, err := WindowsService{Run: r.run}.Enabled()
		if err != nil || got != tc.want {
			t.Errorf("Enabled() = %v, %v; want %v", got, err, tc.want)
		}
	}
}

// exitErr is a failed command as exec reports it: an error with its exit code.
type exitErr int

func (e exitErr) Error() string { return "exit status " + strconv.Itoa(int(e)) }
func (e exitErr) ExitCode() int { return int(e) }

func TestWindowsServiceNamesTheCapabilityWhenSSHDIsMissing(t *testing.T) {
	r := &recorder{answers: map[string]answer{"sc.exe qc": {out: "[SC] OpenService FAILED 1060:", err: exitErr(1060)}}}
	_, err := WindowsService{Run: r.run}.Enabled()
	if err == nil || !strings.Contains(err.Error(), "Add-WindowsCapability -Online -Name OpenSSH.Server") {
		t.Fatalf("Enabled() = %v, want the capability named", err)
	}
	// Another failure whose output carries 1060 is reported as itself.
	r = &recorder{answers: map[string]answer{"sc.exe qc": {out: "[SC] OpenService FAILED 5: Access is denied (pid 1060)", err: exitErr(5)}}}
	_, err = WindowsService{Run: r.run}.Enabled()
	if err == nil || strings.Contains(err.Error(), "not installed") || !strings.Contains(err.Error(), "Access is denied") {
		t.Fatalf("Enabled() = %v, want sc's own error, not a missing capability", err)
	}
}

func TestWindowsServiceReloadsOnlyARunningSSHD(t *testing.T) {
	for _, tc := range []struct {
		state   string
		restart bool
	}{{"STATE : 4 RUNNING", true}, {"STATE : 1 STOPPED", false}} {
		r := &recorder{answers: map[string]answer{"sc.exe query": {out: tc.state}}}
		if err := (WindowsService{Run: r.run}).Reload(r); err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(strings.Join(r.calls, "|"), "Restart-Service"); got != tc.restart {
			t.Errorf("%s: restarted = %v, want %v", tc.state, got, tc.restart)
		}
	}
}

// The OpenSSH.Server capability's own rule: open to every address.
const (
	fwCapabilityDefault = `{"adapter":true,"exists":true,"enabled":"True","action":"Allow","remote":["Any"],"iface":["Any"],"disabled":[],"others":[]}`
	fwConverged         = `{"adapter":true,"exists":true,"enabled":"True","action":"Allow","remote":["100.64.0.0/255.192.0.0"],"iface":["Tailscale"],"disabled":[],"others":[]}`
	fwAbsent            = `{"adapter":true,"exists":false,"enabled":"","action":"","remote":[],"iface":[],"disabled":[],"others":[]}`
)

func fwRecorder(state string) *recorder {
	return &recorder{answers: map[string]answer{"powershell.exe -NoProfile -NonInteractive -Command $ErrorActionPreference = 'Stop'; $r = Get-NetFirewallRule": {out: state}}}
}

func TestWindowsFirewallNarrowsTheCapabilityRuleAndUndoesIt(t *testing.T) {
	r := fwRecorder(fwCapabilityDefault)
	w := WindowsFirewall{Run: r.run}
	if differs, err := w.Differs(); err != nil || !differs {
		t.Fatalf("Differs() = %v, %v; the capability's open rule must differ", differs, err)
	}
	undo, err := w.Apply(r)
	if err != nil {
		t.Fatal(err)
	}
	applied := strings.Join(r.scripts(), "|")
	if !strings.Contains(applied, "Set-NetFirewallRule -Name 'OpenSSH-Server-In-TCP' -Enabled True -Action Allow -InterfaceAlias 'Tailscale' -RemoteAddress '100.64.0.0/10'") {
		t.Fatalf("apply ran %q", applied)
	}
	if err := undo(); err != nil {
		t.Fatal(err)
	}
	if last := r.scripts()[len(r.scripts())-1]; !strings.Contains(last, "-InterfaceAlias @('Any') -RemoteAddress @('Any')") {
		t.Errorf("undo must restore the rule it read, ran %q", last)
	}
}

func TestWindowsFirewallCreatesAMissingRuleNarrow(t *testing.T) {
	r := fwRecorder(fwAbsent)
	undo, err := WindowsFirewall{Run: r.run}.Apply(r)
	if err != nil {
		t.Fatal(err)
	}
	if s := strings.Join(r.scripts(), "|"); !strings.Contains(s, "New-NetFirewallRule -Name 'OpenSSH-Server-In-TCP'") || !strings.Contains(s, "-InterfaceAlias 'Tailscale' -RemoteAddress '100.64.0.0/10'") {
		t.Fatalf("apply ran %q", s)
	}
	_ = undo()
	if last := r.scripts()[len(r.scripts())-1]; !strings.Contains(last, "Remove-NetFirewallRule -Name 'OpenSSH-Server-In-TCP'") {
		t.Errorf("undo of a created rule removes it, ran %q", last)
	}
}

func TestWindowsFirewallConvergedChangesNothing(t *testing.T) {
	r := fwRecorder(fwConverged)
	w := WindowsFirewall{Run: r.run}
	if differs, err := w.Differs(); err != nil || differs {
		t.Fatalf("Differs() = %v, %v; want converged", differs, err)
	}
	if err := w.Prove(); err != nil {
		t.Fatal(err)
	}
	if s := r.scripts(); len(s) > 0 {
		t.Errorf("a converged firewall ran %v", s)
	}
}

func TestWindowsFirewallRefusesWhatWouldStillExposePort22(t *testing.T) {
	for name, state := range map[string]string{
		"another rule open to all":      `{"adapter":true,"exists":true,"enabled":"True","action":"Allow","remote":["Any"],"iface":["Any"],"disabled":[],"others":[{"name":"MyOldSSH","remote":["Any"],"iface":["Any"]}]}`,
		"a profile turned off":          `{"adapter":true,"exists":true,"enabled":"True","action":"Allow","remote":["Any"],"iface":["Any"],"disabled":["Public"],"others":[]}`,
		"the range without the adapter": `{"adapter":true,"exists":true,"enabled":"True","action":"Allow","remote":["100.64.0.0/10"],"iface":["Tailscale"],"disabled":[],"others":[{"name":"CGNAT","remote":["100.64.0.0/10"],"iface":["Any"]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := fwRecorder(state)
			w := WindowsFirewall{Run: r.run}
			// The plan reads every input the refusal does, so it reports the
			// refusal rather than a change the apply will refuse (#2765).
			if differs, err := w.Differs(); err == nil || !strings.Contains(err.Error(), "nothing was changed") {
				t.Errorf("Differs() = %v, %v; want the refusal", differs, err)
			}
			if _, err := w.Apply(r); err == nil || !strings.Contains(err.Error(), "nothing was changed") {
				t.Fatalf("Apply() = %v, want a refusal", err)
			}
			if s := r.scripts(); len(s) > 0 {
				t.Errorf("a refusal changed the firewall: %v", s)
			}
		})
	}
}

func TestMeshOnly(t *testing.T) {
	for _, tc := range []struct {
		remote, iface []string
		want          bool
	}{
		{[]string{"100.64.0.0/10"}, []string{"Tailscale"}, true},
		{[]string{"100.64.0.0/255.192.0.0"}, []string{"Tailscale"}, true}, // how Windows reads 100.64.0.0/10 back
		{[]string{"100.64.0.0/255.0.0.0"}, []string{"Tailscale"}, false},
		{[]string{"100.64.0.0/255.0.255.0"}, []string{"Tailscale"}, false}, // not a contiguous mask
		{[]string{"LocalSubnet"}, []string{"Tailscale"}, false},
		{[]string{"100.100.1.0/24"}, []string{"Tailscale"}, true},
		{[]string{"100.64.0.5"}, []string{"Tailscale"}, true},
		{[]string{"100.0.0.0/8"}, []string{"Tailscale"}, false},
		{[]string{"Any"}, []string{"Tailscale"}, false},
		{[]string{"100.64.0.0/10"}, []string{"Any"}, false},
		{[]string{"100.64.0.0/10"}, []string{"Tailscale", "Ethernet"}, false},
		{nil, []string{"Tailscale"}, false},
	} {
		if got := meshOnly(tc.remote, tc.iface); got != tc.want {
			t.Errorf("meshOnly(%v, %v) = %v, want %v", tc.remote, tc.iface, got, tc.want)
		}
	}
}

func TestWindowsPathsUseTheCapabilityTree(t *testing.T) {
	p := WindowsPaths(`D:\PD`, `C:\Users\manu`)
	for got, want := range map[string]string{
		p.SSHDConfig:     `D:\PD\ssh\sshd_config`,
		p.HostKey:        `D:\PD\ssh\ssh_host_ed25519_key.pub`,
		p.AuthorizedKeys: `D:\PD\ssh\administrators_authorized_keys`,
		p.ClientKey:      `C:\Users\manu\.ssh\id_ed25519`,
	} {
		if got != want {
			t.Errorf("path %q, want %q", got, want)
		}
	}
	if !p.AdminKeys {
		t.Error("an administrator's key lives in administrators_authorized_keys: AdminKeys must be set")
	}
}

func TestWindowsAccount(t *testing.T) {
	for _, tc := range []struct {
		username, computer, want string
		fails                    bool
	}{
		{`DESKTOP-01\Manu`, "DESKTOP-01", "manu", false},
		{`desktop-01\manu`, "DESKTOP-01", "manu", false},
		{`Manu`, "DESKTOP-01", "manu", false},
		{`CORP\manu`, "DESKTOP-01", "", true},
	} {
		got, err := windowsAccount(tc.username, tc.computer)
		if got != tc.want || (err != nil) != tc.fails {
			t.Errorf("windowsAccount(%q, %q) = %q, %v; want %q, fails=%v", tc.username, tc.computer, got, err, tc.want, tc.fails)
		}
	}
}

// fileRecorder is a Root that writes files itself, as Elevated does, and
// records which of its writes ran.
type fileRecorder struct {
	recorder
	files map[string]string // path → "install" | "admin-only" | "removed"
}

func (f *fileRecorder) Install(path string, data []byte) error {
	f.files[path] = "install"
	return nil
}

func (f *fileRecorder) InstallAdminOnly(path string, data []byte) error {
	f.files[path] = "admin-only"
	return nil
}

func (f *fileRecorder) Remove(path string) error {
	f.files[path] = "removed"
	return nil
}

func TestAFileRootWritesTheKeysAdminOnlyAndTheDropInItself(t *testing.T) {
	fr := &fileRecorder{files: map[string]string{}}
	tg := Target{Root: fr, Paths: Paths{AuthorizedKeys: "keys", AdminKeys: true, DropIn: "drop"}}
	if err := tg.writeKeys([]byte(testKey + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := tg.installRoot("drop", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if fr.files["keys"] != "admin-only" || fr.files["drop"] != "install" {
		t.Fatalf("writes = %v", fr.files)
	}
	if err := tg.restoreKeys(nil); err != nil {
		t.Fatal(err)
	}
	if err := tg.restoreRoot("drop", nil); err != nil {
		t.Fatal(err)
	}
	if fr.files["keys"] != "removed" || fr.files["drop"] != "removed" {
		t.Errorf("undo of files that did not exist must remove them: %v", fr.files)
	}
	if len(fr.calls) > 0 {
		t.Errorf("a file root runs no install or rm: %v", fr.calls)
	}
}

func TestAdminKeysRefuseARootThatCannotSetTheirACL(t *testing.T) {
	tg := Target{Root: &recorder{}, Paths: Paths{AuthorizedKeys: filepath.Join(t.TempDir(), "keys"), AdminKeys: true}}
	if err := tg.writeKeys([]byte(testKey)); err == nil || !strings.Contains(err.Error(), "administrators only") {
		t.Fatalf("writeKeys = %v, want a refusal", err)
	}
	if _, err := os.Stat(tg.Paths.AuthorizedKeys); !errors.Is(err, os.ErrNotExist) {
		t.Error("a refused write must leave no file behind")
	}
}
