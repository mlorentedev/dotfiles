package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
)

// A deploy entry under a directory the env contract names is spelled with that
// directory's token, not with {HOME}: a machine that overrides the directory
// would otherwise get the file in one place while the agent, doctor and the
// MCP writers read the other (#2402). The most specific variable wins, so
// ~/.config/opencode answers to OPENCODE_HOME, not to a broader parent.
func TestDeployDstUnderAContractDirUsesItsToken(t *testing.T) {
	c, err := loadContract(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatalf("load real env-contract.json: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(deploy.ManifestRel)))
	if err != nil {
		t.Fatal(err)
	}
	var man struct {
		Configs []deploy.Config `json:"configs"`
	}
	if err := json.Unmarshal(raw, &man); err != nil {
		t.Fatal(err)
	}
	home := "/home/me"
	checked := 0
	for _, goos := range []string{"linux", "darwin"} {
		dirs := map[string]string{} // home-relative directory -> variable
		for _, rv := range Resolve(c, nil, goos, home) {
			if rel, ok := strings.CutPrefix(filepath.ToSlash(rv.Value), home+"/"); ok && rel != "" {
				dirs[rel] = rv.Name
			}
		}
		for _, e := range man.Configs {
			rest, ok := strings.CutPrefix(e.Dst, "{HOME}/")
			if !ok {
				continue
			}
			checked++
			best := ""
			for dir := range dirs {
				if strings.HasPrefix(rest, dir+"/") && len(dir) > len(best) {
					best = dir
				}
			}
			if best != "" {
				t.Errorf("%s on %s: dst %q lies under %s's default; spell it {%s}/%s",
					e.Name, goos, e.Dst, dirs[best], dirs[best], strings.TrimPrefix(rest, best+"/"))
			}
		}
	}
	if checked == 0 {
		t.Fatal("no deploy entry is spelled under {HOME}; the test checks nothing")
	}
}

// ai/deploy.json once spelled these {HOME}/... because a first run deploys
// before machine.json exists (#2402). An absent machine.json resolves to the
// contract default, so the token names the same file on a fresh machine.
func TestContractDirsResolveOnAHomeWithoutMachineJSON(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTFILES_REPO_DIR", repo)
	home := t.TempDir()
	want := map[string]string{
		"COPILOT_HOME":  filepath.Join(home, ".copilot"),
		"OPENCODE_HOME": filepath.Join(home, ".config", "opencode"),
		"AGY_HOME":      filepath.Join(home, ".gemini", "antigravity-cli"),
	}
	for name, path := range want {
		if got := ResolvePathUnder(name, home); filepath.Clean(got) != path {
			t.Errorf("%s under a home with no machine.json = %q, want the contract default %q", name, got, path)
		}
	}
}
