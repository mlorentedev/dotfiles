package tools

import (
	"encoding/json"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Renovate moves the pins in versions.conf through the `# renovate:` line above
// each one (#3284). These tests read the committed files, so a pin added without
// its line, a group too large for one reviewable PR, or an annotation that
// disagrees with how dotf installs the tool fails here rather than in a PR
// Renovate opens on a Monday.

const renovateSkip = "# renovate: skip "

// renovateConfig is the part of renovate.json these tests read.
type renovateConfig struct {
	CustomManagers []struct {
		ManagerFilePatterns []string `json:"managerFilePatterns"`
		MatchStrings        []string `json:"matchStrings"`
	} `json:"customManagers"`
	PackageRules []struct {
		MatchDepTypes     []string `json:"matchDepTypes"`
		MatchPackageNames []string `json:"matchPackageNames"`
		MatchUpdateTypes  []string `json:"matchUpdateTypes"`
		GroupName         string   `json:"groupName"`
		Enabled           *bool    `json:"enabled"`
	} `json:"packageRules"`
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile("../../../" + rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	// A Windows checkout may carry CRLF; the regexes here, like Renovate's, read LF.
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func loadRenovateConfig(t *testing.T) renovateConfig {
	t.Helper()
	var cfg renovateConfig
	if err := json.Unmarshal([]byte(readRepoFile(t, "renovate.json")), &cfg); err != nil {
		t.Fatalf("parse renovate.json: %v", err)
	}
	return cfg
}

// managerRegex compiles the first match string of the custom manager whose file
// pattern names file. Renovate's JavaScript named groups, (?<name>…), are valid
// Go syntax, so the test runs the regex Renovate runs.
func managerRegex(t *testing.T, cfg renovateConfig, file string) *regexp.Regexp {
	t.Helper()
	for _, m := range cfg.CustomManagers {
		for _, p := range m.ManagerFilePatterns {
			if strings.Contains(p, regexp.QuoteMeta(file)) || strings.Contains(p, strings.ReplaceAll(file, ".", `\.`)) {
				return regexp.MustCompile(m.MatchStrings[0])
			}
		}
	}
	t.Fatalf("renovate.json has no custom manager for %s", file)
	return nil
}

// renovatePin is one pin of versions.conf and the annotation above it.
type renovatePin struct {
	name, value, annotation string
}

var pinLine = regexp.MustCompile(`^([A-Z0-9_]+)_VERSION=(\S+)$`)

// versionsConfPins pairs each pin with its `# renovate:` line: directly above
// it, or above the one marker line (`# mise:`, `# pwsh-module:`,
// `# x-release-please-start-version`) that has to stay adjacent to the pin.
func versionsConfPins(t *testing.T) []renovatePin {
	t.Helper()
	lines := strings.Split(readRepoFile(t, "versions.conf"), "\n")
	var pins []renovatePin
	for i, line := range lines {
		m := pinLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		pin := renovatePin{name: m[1], value: m[2]}
		for back := 1; back <= 2 && i-back >= 0; back++ {
			if above := lines[i-back]; strings.HasPrefix(above, "# renovate: ") {
				pin.annotation = above
				break
			}
		}
		pins = append(pins, pin)
	}
	if len(pins) == 0 {
		t.Fatal("versions.conf yielded no pins: the reader is broken, not the file")
	}
	return pins
}

func TestVersionsConf_EveryPinNamesItsUpstreamOrWhyItHasNone(t *testing.T) {
	for _, p := range versionsConfPins(t) {
		switch {
		case p.annotation == "":
			t.Errorf("%s_VERSION has no `# renovate:` line above it (or above its marker): add `# renovate: datasource=… depName=… depType=…` or `%s<reason>`", p.name, renovateSkip)
		case strings.HasPrefix(p.annotation, renovateSkip):
			if strings.TrimSpace(strings.TrimPrefix(p.annotation, renovateSkip)) == "" {
				t.Errorf("%s_VERSION is skipped without a reason", p.name)
			}
		}
	}
}

// The regex renovate.json runs must find every annotated pin at its value, and
// nothing else: a typo in an annotation would otherwise drop the pin silently.
func TestVersionsConf_RenovateFindsEveryAnnotatedPinAtItsValue(t *testing.T) {
	re := managerRegex(t, loadRenovateConfig(t), "versions.conf")
	found := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(readRepoFile(t, "versions.conf"), -1) {
		pin := pinLine.FindStringSubmatch(m[0][strings.LastIndex(m[0], "\n")+1:])
		found[pin[1]] = m[re.SubexpIndex("currentValue")]
	}
	for _, p := range versionsConfPins(t) {
		got, ok := found[p.name]
		skipped := strings.HasPrefix(p.annotation, renovateSkip)
		switch {
		case skipped && ok:
			t.Errorf("%s_VERSION is skipped but Renovate's regex still matches it", p.name)
		case !skipped && !ok:
			t.Errorf("%s_VERSION is annotated %q but Renovate's regex does not match it", p.name, p.annotation)
		case ok && got != p.value:
			t.Errorf("Renovate reads %s_VERSION as %q, the file says %q", p.name, got, p.value)
		}
	}
}

// specGateThreshold reads the threshold the spec-gate applies, so the group cap
// below follows the gate instead of copying its number.
func specGateThreshold(t *testing.T) int {
	t.Helper()
	m := regexp.MustCompile(`(?m)^THRESHOLD=(\d+)$`).FindStringSubmatch(readRepoFile(t, "scripts/check-spec-gate.sh"))
	if m == nil {
		t.Fatal("scripts/check-spec-gate.sh has no THRESHOLD= line")
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// Renovate's PRs are authored by its token's owner, so the spec-gate's bot skip
// never applies to them: a group whose every pin moves at once must stay under
// the gate's threshold. A pin move is two lines (one removed, one added).
// Measured on 2026-11-15: 29 pins in one PR were 58 lines and failed the gate.
func TestRenovate_NoGroupCanOutgrowTheSpecGate(t *testing.T) {
	cfg := loadRenovateConfig(t)
	var grouped []string
	for _, r := range cfg.PackageRules {
		if r.GroupName != "" && len(r.MatchDepTypes) > 0 {
			grouped = append(grouped, r.MatchDepTypes...)
		}
	}
	re := managerRegex(t, cfg, "versions.conf")
	perGroup := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(readRepoFile(t, "versions.conf"), -1) {
		perGroup[m[re.SubexpIndex("depType")]]++
	}
	perGroup["runtime"]++ // cli/go.mod's go line moves with GO_VERSION
	perGroup["catalog"] += strings.Count(readRepoFile(t, "packages.json"), `"type": "github-`) +
		strings.Count(readRepoFile(t, "packages.json"), `"type": "uv-tool"`)

	threshold := specGateThreshold(t)
	for group, pins := range perGroup {
		if !slices.Contains(grouped, group) {
			t.Errorf("depType %q has no group rule in renovate.json, so its pins fall into the preset's single group", group)
		}
		if lines := 2 * pins; lines >= threshold {
			t.Errorf("group %q moves %d pins, %d lines, and the spec-gate fails at %d: split it", group, pins, lines, threshold)
		}
	}
}

// The npm-backend CLIs install the package npmBackend names, and Renovate must
// look up that same package; mise's registry is not the source (it drifts).
func TestVersionsConf_NpmCLIsTrackThePackageDotfInstalls(t *testing.T) {
	annotations := map[string]string{}
	for _, p := range versionsConfPins(t) {
		annotations[strings.ToLower(p.name)] = p.annotation
	}
	for name, backend := range npmBackend {
		want := "datasource=npm depName=" + strings.TrimPrefix(backend, "npm:") + " "
		if got := annotations[name]; !strings.Contains(got+" ", want) {
			t.Errorf("%s installs %s, but its renovate line is %q", name, backend, got)
		}
	}
}

// Pins whose move breaks something no CI job sees stay out of Renovate.
func TestVersionsConf_PinsRenovateMustNotMoveAreSkipped(t *testing.T) {
	mustSkip := map[string]string{
		"PYTHON":   "venvs and uv tools are built against it by path (#3541)",
		"JAVA":     "names a directory nothing installs",
		"MAVEN":    "names a directory nothing installs",
		"MINIKUBE": "names a directory nothing installs",
		"GIT":      "a floor, not an install pin",
		"DOTF":     "release-please moves it",
	}
	// A file tool's pin needs its digest in a released dotf before it moves
	// (mise_files.go), so a Renovate PR could never land it in one step.
	for name := range fileTools {
		mustSkip[strings.ToUpper(strings.ReplaceAll(name, "-", "_"))] = "its sha256 must ship in a released dotf first"
	}
	lines := strings.Split(readRepoFile(t, "versions.conf"), "\n")
	for i, line := range lines {
		if line == MiseLatestMarker && i+1 < len(lines) {
			if m := pinLine.FindStringSubmatch(lines[i+1]); m != nil {
				mustSkip[m[1]] = "mise installs the latest release"
			}
		}
	}
	for _, p := range versionsConfPins(t) {
		if why, ok := mustSkip[p.name]; ok && !strings.HasPrefix(p.annotation, renovateSkip) {
			t.Errorf("%s_VERSION must be `%s…` (%s), got %q", p.name, renovateSkip, why, p.annotation)
		}
	}
}

// CI builds dotf with cli/go.mod's go line, so GO_VERSION never moves alone, and
// only by patch: a Go minor has broken the pinned golangci-lint.
func TestRenovate_GoMovesByPatchTogetherWithGoMod(t *testing.T) {
	cfg := loadRenovateConfig(t)
	m := managerRegex(t, cfg, "cli/go.mod").FindStringSubmatch(readRepoFile(t, "cli/go.mod"))
	if m == nil {
		t.Fatal("renovate.json's cli/go.mod manager does not match cli/go.mod's go line")
	}
	for _, p := range versionsConfPins(t) {
		if p.name == "GO" && p.value != m[1] {
			t.Errorf("GO_VERSION is %s but cli/go.mod says go %s: they move together", p.value, m[1])
		}
	}
	for _, r := range cfg.PackageRules {
		if slices.Contains(r.MatchPackageNames, "go") && r.Enabled != nil && !*r.Enabled &&
			slices.Contains(r.MatchUpdateTypes, "minor") && slices.Contains(r.MatchUpdateTypes, "major") {
			return
		}
	}
	t.Error("renovate.json has no rule disabling Go's minor and major updates")
}
