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
