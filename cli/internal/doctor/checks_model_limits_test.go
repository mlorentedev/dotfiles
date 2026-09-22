package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// limitsFixture writes a declaration into a fake checkout and a catalog into a
// fake $HOME, and returns what checkModelLimits needs to read both.
//
// Both halves are written per-test rather than copied from the repo, unlike
// pinFixture next door. The difference is deliberate: that check asserts the
// SHIPPED registries stay coherent, so a fixture copy would let them drift. This
// one asserts a COMPARISON, and pinning it to today's real numbers would make
// every test fail the day a provider legitimately raises a limit.
func limitsFixture(t *testing.T, declaration, catalog string) (*System, *Config) {
	t.Helper()
	repo := t.TempDir()
	home := t.TempDir()

	declDir := filepath.Join(repo, "ai", "pi")
	if err := os.MkdirAll(declDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if declaration != "" {
		if err := os.WriteFile(filepath.Join(declDir, "models.json"), []byte(declaration), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if catalog != "" {
		catDir := filepath.Join(home, ".cache", "opencode")
		if err := os.MkdirAll(catDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(catDir, "models.json"), []byte(catalog), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return piSys(home), &Config{RepoDir: repo, DotfilesDir: repo}
}

func limitsRun(t *testing.T, sys *System, cfg *Config) (string, map[Status]int) {
	t.Helper()
	var buf bytes.Buffer
	rep := NewReport(&buf, true)
	checkModelLimits(sys, cfg, rep)
	rep.flush()
	return buf.String(), rep.totals
}

func decl(provider, id string, ctx, out int) string {
	return `{"providers":{"` + provider + `":{"models":[{"id":"` + id + `","contextWindow":` +
		strconv.Itoa(ctx) + `,"maxTokens":` + strconv.Itoa(out) + `}]}}}`
}

func cat(provider, id string, ctx, out int) string {
	return `{"` + provider + `":{"models":{"` + id + `":{"limit":{"context":` +
		strconv.Itoa(ctx) + `,"output":` + strconv.Itoa(out) + `}}}}}`
}

// A declaration ABOVE the provider's real limit is the request-breaking
// direction, so it fails. These are the measured 2026-09-21 values for
// nan/deepseek-v4-flash, where 2^20 read as "1M" and is 48576 over the real one.
func TestModelLimitsFailsOnAnOverstatedContextWindow(t *testing.T) {
	sys, cfg := limitsFixture(t,
		decl("nan", "deepseek-v4-flash", 1048576, 384000),
		cat("nan", "deepseek-v4-flash", 1000000, 384000))

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusFail] != 1 {
		t.Fatalf("want exactly 1 FAIL, got %d\n%s", totals[StatusFail], out)
	}
	if totals[StatusWarn] != 0 {
		t.Errorf("an overstatement must not also warn, got %d\n%s", totals[StatusWarn], out)
	}
	if !strings.Contains(out, "1048576") || !strings.Contains(out, "1000000") {
		t.Errorf("the report must name both numbers so it can be acted on:\n%s", out)
	}
}

// A declaration BELOW the real limit forfeits capability without breaking a
// request, so it warns. Measured value: nan/qwen3.8-flash shipped 16384 against
// a real 131072, throwing away 8x the output.
func TestModelLimitsWarnsOnAnUnderstatedOutputCap(t *testing.T) {
	sys, cfg := limitsFixture(t,
		decl("nan", "qwen3.8-flash", 262144, 16384),
		cat("nan", "qwen3.8-flash", 262144, 131072))

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusWarn] != 1 {
		t.Fatalf("want exactly 1 WARN, got %d\n%s", totals[StatusWarn], out)
	}
	if totals[StatusFail] != 0 {
		t.Errorf("an understatement breaks nothing and must not fail: %d\n%s", totals[StatusFail], out)
	}
}

// THE REGRESSION THIS FILE EXISTS FOR. `qwen3.8-flash` is published by BOTH nan
// (262144 context) and openrouter (1000000). An id-only lookup reports the
// honest nan declaration as 4x wrong — which is exactly what an unscoped first
// draft of this check did while it was being written.
func TestModelLimitsResolvesTheSameIdPerProvider(t *testing.T) {
	declaration := `{"providers":{
		"nan":{"models":[{"id":"qwen3.8-flash","contextWindow":262144,"maxTokens":131072}]},
		"openrouter":{"models":[{"id":"qwen3.8-flash","contextWindow":1000000,"maxTokens":131072}]}}}`
	catalog := `{
		"nan":{"models":{"qwen3.8-flash":{"limit":{"context":262144,"output":131072}}}},
		"openrouter":{"models":{"qwen3.8-flash":{"limit":{"context":1000000,"output":131072}}}}}`

	sys, cfg := limitsFixture(t, declaration, catalog)
	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusFail] != 0 || totals[StatusWarn] != 0 {
		t.Fatalf("both declarations are correct for their own provider, got %d FAIL / %d WARN\n%s",
			totals[StatusFail], totals[StatusWarn], out)
	}
	if totals[StatusPass] != 1 {
		t.Errorf("want a PASS naming the comparison, got %d\n%s", totals[StatusPass], out)
	}
}

// An absent catalog must SKIP, never PASS: "nothing to compare against" and
// "everything matches" print the same clean report and mean opposite things.
func TestModelLimitsSkipsWhenTheCatalogIsNotCached(t *testing.T) {
	sys, cfg := limitsFixture(t, decl("nan", "qwen3.6", 262144, 65536), "")

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusSkip] != 1 {
		t.Fatalf("want a SKIP, got %d\n%s", totals[StatusSkip], out)
	}
	if totals[StatusPass] != 0 {
		t.Errorf("an uncomparable state must never read as agreement: %d\n%s", totals[StatusPass], out)
	}
}

// A model the catalog does not publish is not a finding — a private or brand-new
// model has no published truth to disagree with. It must also not count as a
// comparison, or an all-unknown declaration would report a vacuous PASS.
func TestModelLimitsIgnoresAModelTheCatalogDoesNotPublish(t *testing.T) {
	sys, cfg := limitsFixture(t,
		decl("nan", "some-private-preview", 999, 999),
		cat("nan", "qwen3.6", 262144, 65536))

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusFail] != 0 || totals[StatusWarn] != 0 {
		t.Fatalf("an unpublished model is not drift, got %d FAIL / %d WARN\n%s",
			totals[StatusFail], totals[StatusWarn], out)
	}
	if totals[StatusSkip] != 1 {
		t.Errorf("nothing was actually compared, so the check must say so: %d SKIP\n%s",
			totals[StatusSkip], out)
	}
}

// An unparseable declaration fails loudly. Silently reporting no drift is the
// failure mode this repository's prohibited-pattern table is largely about: a
// check whose broken path answers "found nothing".
func TestModelLimitsFailsOnAnUnparseableDeclaration(t *testing.T) {
	sys, cfg := limitsFixture(t, `{"providers":`, cat("nan", "qwen3.6", 262144, 65536))

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusFail] != 1 {
		t.Fatalf("want a FAIL for a declaration that could not be read, got %d\n%s",
			totals[StatusFail], out)
	}
}

// A catalog entry with no published output limit is not a claim that the limit is
// zero. Comparing against it would report every such model as overstated.
func TestModelLimitsIgnoresAnUnpublishedLimit(t *testing.T) {
	sys, cfg := limitsFixture(t,
		decl("nan", "glm5.3-flash", 1000000, 131072),
		cat("nan", "glm5.3-flash", 1000000, 0))

	out, totals := limitsRun(t, sys, cfg)

	if totals[StatusFail] != 0 || totals[StatusWarn] != 0 {
		t.Fatalf("an unpublished limit is not drift, got %d FAIL / %d WARN\n%s",
			totals[StatusFail], totals[StatusWarn], out)
	}
}
