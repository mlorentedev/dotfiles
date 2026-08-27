package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinFixture builds a dotfiles dir carrying the two registries plus a fake $HOME
// holding a deployed pi settings.json with the given enabledModels/defaultModel.
func pinFixture(t *testing.T, defaultModel string, enabled []string) (*System, *Config, string) {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "harness"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The real registries, so the fixture cannot drift from what ships.
	for _, f := range []string{"model-pins.json", "model-map.json", "model-map.schema.json"} {
		src := filepath.Join(repoRootForDoctorTest(t), "harness", f)
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(root, "harness", f), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	settings := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := `{"defaultProvider":"nan","defaultModel":"` + defaultModel + `","enabledModels":["` +
		strings.Join(enabled, `","`) + `"]}`
	if err := os.WriteFile(settings, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}

	return piSys(home), &Config{DotfilesDir: root}, settings
}

func pinRun(t *testing.T, sys *System, cfg *Config) string {
	t.Helper()
	var buf bytes.Buffer
	rep := NewReport(&buf, true)
	checkModelPins(sys, cfg, rep)
	rep.flush()
	return buf.String()
}

// AC5 — a deployed catalog id that is a frozen snapshot of a live model is
// reported. This is the exact value measured on the real machine 2026-08-26.
func TestModelPinsReportsAFrozenSnapshot(t *testing.T) {
	sys, cfg, _ := pinFixture(t, "qwen3.6", []string{"nan/qwen3.6", "nan/deepseek-v4-flash-0731"})
	out := pinRun(t, sys, cfg)

	if !strings.Contains(out, "deepseek-v4-flash-0731") {
		t.Fatalf("the dead dated id was not reported:\n%s", out)
	}
	if !strings.Contains(out, "frozen snapshot") {
		t.Errorf("the diagnostic does not say WHY it is stale:\n%s", out)
	}
	if strings.Contains(out, cfg.DotfilesDir) || strings.Contains(out, sys.home()) {
		t.Errorf("a literal path was printed (ADR-025):\n%s", out)
	}
}

// AC6 — a retired provider reads differently from an unresolvable model id,
// because they are different problems with different fixes.
func TestModelPinsDistinguishesARetiredProvider(t *testing.T) {
	sys, cfg, _ := pinFixture(t, "qwen3.6", []string{"nan/qwen3.6", "openrouter/minimax/minimax-m3"})
	out := pinRun(t, sys, cfg)

	if !strings.Contains(out, "retired") {
		t.Fatalf("a retired-provider entry was not named as such:\n%s", out)
	}
	if strings.Contains(out, "frozen snapshot") {
		t.Errorf("a retired provider was misreported as a snapshot:\n%s", out)
	}
}

// AC4's deployed half, and a regression guard on a false positive this check
// SHIPPED WITH before a real run caught it.
//
// `nan/gemma4` is a live NaN model that the map does not route, exactly as
// `qwen3.8-flash` and `glm5.3-flash` are (#1244). The first version of this
// check reported every unrouted catalog id, which would have fired on all three
// — the very failure the registry's own $comment warns against.
func TestModelPinsDoesNotReportAnUnroutedCatalogModel(t *testing.T) {
	sys, cfg, _ := pinFixture(t, "qwen3.6", []string{
		"nan/qwen3.6", "nan/gemma4", "nan/qwen3.8-flash", "nan/glm5.3-flash",
	})
	out := pinRun(t, sys, cfg)

	for _, legit := range []string{"gemma4", "qwen3.8-flash", "glm5.3-flash"} {
		if strings.Contains(out, legit) {
			t.Errorf("%q is a catalog model the map does not route, which is not drift:\n%s", legit, out)
		}
	}
	if !strings.Contains(out, "[ OK ]") {
		t.Errorf("a clean machine must report OK:\n%s", out)
	}
}

// A dead SCALAR routing pin fails rather than warns: it decides what a real
// session runs on.
func TestModelPinsFailsOnADeadDefaultModel(t *testing.T) {
	sys, cfg, _ := pinFixture(t, "deepseek-v4-flash-0731", []string{"nan/qwen3.6"})
	out := pinRun(t, sys, cfg)

	if !strings.Contains(out, "[FAIL]") {
		t.Fatalf("a dead defaultModel must FAIL, not warn:\n%s", out)
	}
	if !strings.Contains(out, "what a real session runs on") {
		t.Errorf("the diagnostic does not say why this is worse than a catalog row:\n%s", out)
	}
}

// AC8 — the check never writes. Asserted rather than assumed, because the file
// it reads is pi's own runtime state and repairing it is an open question, not a
// default.
func TestModelPinsNeverWrites(t *testing.T) {
	sys, cfg, settings := pinFixture(t, "qwen3.6", []string{
		"nan/deepseek-v4-flash-0731", "openrouter/minimax/minimax-m3",
	})
	before, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	statBefore, err := os.Stat(settings)
	if err != nil {
		t.Fatal(err)
	}

	if out := pinRun(t, sys, cfg); !strings.Contains(out, "[WARN]") {
		t.Fatalf("fixture should have produced findings:\n%s", out)
	}

	after, err := os.ReadFile(settings)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("the check modified the deployed settings.json")
	}
	statAfter, err := os.Stat(settings)
	if err != nil {
		t.Fatal(err)
	}
	if !statBefore.ModTime().Equal(statAfter.ModTime()) {
		t.Error("the check touched the deployed settings.json")
	}
}

// AC7's doctor half — an unreadable registry is a loud FAIL, never a clean sweep.
func TestModelPinsFailsLoudlyWithoutARegistry(t *testing.T) {
	sys, _, _ := pinFixture(t, "qwen3.6", []string{"nan/qwen3.6"})
	out := pinRun(t, sys, &Config{DotfilesDir: t.TempDir()})

	if !strings.Contains(out, "[FAIL]") {
		t.Fatalf("a missing registry must FAIL:\n%s", out)
	}
	for _, banned := range []string{"[ OK ]", "all resolve"} {
		if strings.Contains(out, banned) {
			t.Errorf("a broken guard reported %q:\n%s", banned, out)
		}
	}
}

// An absent deployed file is a SKIP, not a failure: not every machine runs pi.
func TestModelPinsSkipsWhenNotDeployed(t *testing.T) {
	sys, cfg, settings := pinFixture(t, "qwen3.6", []string{"nan/qwen3.6"})
	if err := os.Remove(settings); err != nil {
		t.Fatal(err)
	}
	out := pinRun(t, sys, cfg)
	if !strings.Contains(out, "[SKIP]") {
		t.Fatalf("an undeployed site must SKIP:\n%s", out)
	}
}
