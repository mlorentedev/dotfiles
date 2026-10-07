package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

const miseVersions = "# mise: cli\nJQ_VERSION=1.7.1\n# mise: cli\nAGE_VERSION=1.2.1\n"

// miseFixture is a machine where mise resolves jq and age; ageVersion is what
// the installed age reports, and synced writes the config the sync renders.
func miseFixture(t *testing.T, goos, ageVersion string, synced bool) (*System, *Config) {
	t.Helper()
	home := t.TempDir()
	versions := filepath.Join(home, "versions.conf")
	writeFile(t, versions, miseVersions)
	if synced {
		pins, err := tools.ParseMiseTools([]byte(miseVersions))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(home, ".config", "mise", "conf.d", "dotfiles.toml"), string(tools.RenderMiseConfig(pins)))
	}
	s := newSys(map[string]string{"HOME": home}, []string{"mise"}, map[string]string{
		"mise which jq":    "/m/jq\n",
		"/m/jq --version":  "jq-1.7.1",
		"mise which age":   "/m/age\n",
		"/m/age --version": "v" + ageVersion,
	})
	s.GOOS = goos
	return s, &Config{VersionsPath: versions}
}

func TestCheckMiseTools_AtTheirPinPassesOnEveryOS(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		s, cfg := miseFixture(t, goos, "1.2.1", true)
		var b bytes.Buffer
		rep := capture(&b)
		checkMiseTools(s, cfg, rep)
		if rep.Failures() != 0 || rep.Warnings() != 0 || !strings.Contains(b.String(), "2 CLI(s) at their pin") {
			t.Errorf("%s: want a clean pass\n%s", goos, b.String())
		}
	}
}

func TestCheckMiseTools_ACLIBelowItsPinFailsNamingTheSync(t *testing.T) {
	s, cfg := miseFixture(t, "darwin", "1.1.0", true)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep)
	if rep.Failures() != 1 || !strings.Contains(b.String(), "age") || !strings.Contains(b.String(), "dotf tools sync") {
		t.Errorf("want one FAIL naming age and the sync\n%s", b.String())
	}
}

func TestCheckMiseTools_AStaleRenderedConfigWarns(t *testing.T) {
	s, cfg := miseFixture(t, "linux", "1.2.1", false)
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "dotf tools sync") {
		t.Errorf("want one WARN naming the sync\n%s", b.String())
	}
}

// No mise: the pins are not managed, which the core-tools check does not see
// when a CLI is on PATH from elsewhere. Advisory, until mise ships in the
// catalog (PLAT-001c T1b).
func TestCheckMiseTools_NoMiseWarns(t *testing.T) {
	s, cfg := miseFixture(t, "linux", "1.2.1", true)
	s.LookPath = newSys(nil, nil, nil).LookPath
	var b bytes.Buffer
	rep := capture(&b)
	checkMiseTools(s, cfg, rep)
	if rep.Failures() != 0 || rep.Warnings() != 1 || !strings.Contains(b.String(), "mise not on PATH") {
		t.Errorf("want one WARN\n%s", b.String())
	}
}
