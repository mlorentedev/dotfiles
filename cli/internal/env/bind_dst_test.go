package env

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mlorentedev/dotfiles/cli/internal/deploy"
	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// A bind target spells its settings file twice: `dst`, which this binary
// resolves, and `file`, which a binary older than `dst` joins to the home
// (#2279). Under the contract default the two must name one file on every OS,
// or an older binary binds a file the harness does not read.
func TestBindDstMatchesItsFileUnderTheContractDefault(t *testing.T) {
	c, err := loadContract(filepath.Join("..", "..", "..", "env-contract.json"))
	if err != nil {
		t.Fatalf("load real env-contract.json: %v", err)
	}
	targets, err := harness.LoadBindTargets(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join("home", "me")
	norm := func(p string) string { return filepath.ToSlash(strings.ReplaceAll(p, `\`, "/")) }
	checked := 0
	for _, tg := range targets {
		if tg.Dst == "" {
			continue
		}
		for _, goos := range []string{"linux", "darwin", "windows"} {
			defaults := map[string]string{}
			for _, rv := range Resolve(c, nil, goos, home) {
				defaults[rv.Name] = rv.Value
			}
			got, err := deploy.ExpandDst(tg.Dst, home, func(n string) string { return defaults[n] })
			if err != nil {
				t.Fatalf("%s on %s: %v", tg.Agent, goos, err)
			}
			if want := filepath.Join(home, filepath.FromSlash(tg.File)); norm(got) != norm(want) {
				t.Errorf("%s on %s: dst %q resolves to %q, but file names %q", tg.Agent, goos, tg.Dst, got, want)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no bind target declares dst; the claude target lost it (#2279)")
	}
}
