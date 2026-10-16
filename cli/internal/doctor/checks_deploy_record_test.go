package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

// #2570: doctor reads the merge records deploy writes, so a key the source
// retired is drift exactly when deploy would remove it, and not without the
// record that says the repo put it there.
func TestCheckDeployManifest_ARetiredMergeKeyIsDrift(t *testing.T) {
	repo, home := deployManifestRepo(t), t.TempDir()
	raw, err := os.ReadFile(filepath.Join(repo, "ai", "deploy.json"))
	if err != nil {
		t.Fatal(err)
	}
	man, err := deploy.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	m := *man.Lookup("m")
	records := deploy.RecordsDir(filepath.Join(home, ".local", "state", "dotfiles"))
	writeFile(t, filepath.Join(repo, "ai", "m.json"), `{"model":"m","autoUpdate":false,"old":1}`)
	if _, err := deploy.DeployWith(m, repo, home, func(string) string { return "" }, nil, false, records); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "ai", "m.json"), `{"model":"m","autoUpdate":false}`)

	if out := runCheckDeployManifest(t, repo, home, nil); !strings.Contains(out, "drift: m ") {
		t.Errorf("a retired key deploy would remove must read as drift:\n%s", out)
	}
	if err := os.RemoveAll(records); err != nil {
		t.Fatal(err)
	}
	if out := runCheckDeployManifest(t, repo, home, nil); strings.Contains(out, "drift: m ") {
		t.Errorf("with no record there is no history to retire from, so no drift:\n%s", out)
	}
}
