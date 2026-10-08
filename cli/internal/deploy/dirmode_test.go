package deploy

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

// Two entries share ~/.ssh: a 0600 config and a 0644 public key. The directory
// is private because one of them is, whichever entry creates it. Before #2053
// `dotf deploy ssh-pubkey` alone created ~/.ssh 0755.
const sshManifest = `{"version":4,"configs":[
	{"name":"ssh-config","src":"ssh/config","dst":"{HOME}/.ssh/config","mode":"0600"},
	{"name":"ssh-pubkey","src":"ssh/id.pub","dst":"{HOME}/.ssh/id.pub","mode":"0644"},
	{"name":"rc","src":"rc","dst":"{HOME}/.tool/rc","mode":"0644"}
]}`

func TestDeploy_TheDirectoryModeDoesNotDependOnWhichEntryCreatesIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful here")
	}
	man, err := ParseManifest([]byte(sshManifest))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, f := range []string{"ssh/config", "ssh/id.pub", "rc"} {
		p := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		entry, dir string
		want       os.FileMode
	}{
		{"ssh-pubkey", ".ssh", 0o700}, // the public entry alone still makes a private ~/.ssh
		{"ssh-config", ".ssh", 0o700},
		{"rc", ".tool", 0o755}, // a directory only public entries share stays 0755
	} {
		t.Run(tc.entry, func(t *testing.T) {
			home := t.TempDir()
			if _, err := Deploy(*man.Lookup(tc.entry), root, home, noResolve, nil, false); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(home, tc.dir))
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tc.want {
				t.Errorf("%s created %s %o, want %o", tc.entry, tc.dir, got, tc.want)
			}
		})
	}
}

func TestManifest_PrivateDirsNamesEveryDirectoryAPrivateEntryShares(t *testing.T) {
	man, err := ParseManifest([]byte(sshManifest))
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	all := func(Config) bool { return true }
	got, err := man.PrivateDirs(home, noResolve, all)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("nothing is deployed yet, so no directory holds a private file: %v", got)
	}

	for _, f := range []string{".ssh/config", ".ssh/id.pub", ".tool/rc"} {
		p := filepath.Join(home, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err = man.PrivateDirs(home, noResolve, all)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{filepath.Join(home, ".ssh")}; !slices.Equal(got, want) {
		t.Errorf("PrivateDirs = %v, want %v", got, want)
	}

	// An entry that does not apply here (another OS, an absent tool) deploys
	// nothing, whatever its directory holds.
	got, err = man.PrivateDirs(home, noResolve, func(c Config) bool { return c.Name != "ssh-config" })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("a filtered-out private entry must not mark its directory: %v", got)
	}
}

// The shipped manifest's two mixed directories are private for every entry in
// them, so the public entry alone cannot create either one open.
func TestParseManifest_ShippedMixedDirectoriesArePrivateForEveryEntry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ManifestRel))
	if err != nil {
		t.Skipf("manifest not reachable from the test's working directory: %v", err)
	}
	m, err := ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ssh-config", "ssh-pubkey", "pi", "pi-mcp", "pi-nan-provider", "pi-compaction"} {
		c := m.Lookup(name)
		if c == nil {
			t.Errorf("the shipped manifest no longer declares %s", name)
			continue
		}
		if c.dirMode != 0o700 {
			t.Errorf("%s: directory mode %o, want 0700", name, c.dirMode)
		}
	}
}

// TightenDir is the one rule deploy and doctor --fix apply to a directory that
// holds a private deployed file (#2161): drop group and other access, never add
// a bit, and leave a directory that grants nothing alone.
func TestTightenDir_RemovesGroupAndOtherAccessOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory modes; Windows is asserted by TestTightenDir_IsANoOpOnWindows")
	}
	for _, tc := range []struct {
		have, want os.FileMode
		changed    bool
	}{
		{0o755, 0o700, true},
		{0o750, 0o700, true},
		{0o711, 0o700, true},
		{0o555, 0o500, true}, // owner bits are kept as they are: no write is added
		{0o700, 0o700, false},
		{0o500, 0o500, false},
	} {
		for _, dryRun := range []bool{true, false} {
			dir := filepath.Join(t.TempDir(), "d")
			if err := os.Mkdir(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(dir, tc.have); err != nil {
				t.Fatal(err)
			}
			from, to, changed, err := TightenDir(dir, runtime.GOOS, dryRun)
			if err != nil {
				t.Fatalf("%04o dry=%v: %v", tc.have, dryRun, err)
			}
			if changed != tc.changed || from != tc.have || to != tc.want {
				t.Errorf("%04o dry=%v: got from=%04o to=%04o changed=%v, want to=%04o changed=%v",
					tc.have, dryRun, from, to, changed, tc.want, tc.changed)
			}
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatal(err)
			}
			wantOnDisk := tc.want
			if dryRun {
				wantOnDisk = tc.have
			}
			if got := info.Mode().Perm(); got != wantOnDisk {
				t.Errorf("%04o dry=%v: directory is %04o on disk, want %04o", tc.have, dryRun, got, wantOnDisk)
			}
			_ = os.Chmod(dir, 0o700) // let t.TempDir clean up a 0500
		}
	}
}

func TestTightenDir_AnAbsentDirectoryIsNotAChange(t *testing.T) {
	_, _, changed, err := TightenDir(filepath.Join(t.TempDir(), "absent"), runtime.GOOS, false)
	if err != nil || changed {
		t.Errorf("absent directory: changed=%v err=%v, want neither", changed, err)
	}
}

// Windows reports 0777 for every directory and has no group or other bits to
// clear; a tightening there would print on every deploy and change nothing.
func TestTightenDir_IsANoOpOnWindows(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, changed, err := TightenDir(dir, "windows", false)
	if err != nil || changed {
		t.Errorf("windows: changed=%v err=%v, want neither", changed, err)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(dir); info.Mode().Perm() != 0o755 {
			t.Errorf("windows must leave the directory alone, it is %04o", info.Mode().Perm())
		}
	}
}
