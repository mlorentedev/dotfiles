package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillField(t *testing.T) {
	cases := []struct {
		name, record, want string
	}{
		{"the first value after the key", "---\nname: x\ndescription:   a demo \n---\n", "a demo "},
		{"quotes are kept", "---\ndescription: \"quoted\"\n---\n", "\"quoted\""},
		{"a key in the body is not frontmatter", "---\nname: x\n---\ndescription: body\n", ""},
		{"a block scalar yields its indicator", "---\ndescription: >-\n  folded\n---\n", ">-"},
		{"a prefixed key does not match", "---\ndescription_long: no\n---\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := skillField([]byte(tc.record), "description"); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReplaceCatalogRegion(t *testing.T) {
	file := "# Copilot\n\n<!-- BEGIN HARNESS GENERATED -->\nold\n<!-- END HARNESS GENERATED -->\n\ntail"
	got, ok := replaceCatalogRegion(file, "<!-- BEGIN NEW -->", "- **a** — x\n")
	want := "# Copilot\n\n<!-- BEGIN NEW -->\n- **a** — x\n<!-- END HARNESS GENERATED -->\n\ntail\n"
	if !ok || got != want {
		t.Errorf("got %v:\n%s\nwant:\n%s", ok, got, want)
	}
}

func TestReplaceCatalogRegion_RefusesBadMarkers(t *testing.T) {
	for name, file := range map[string]string{
		"no region":        "# Copilot\n",
		"two BEGIN lines":  "<!-- BEGIN HARNESS GENERATED -->\n<!-- BEGIN HARNESS GENERATED x -->\n<!-- END HARNESS GENERATED -->\n",
		"END before BEGIN": "<!-- END HARNESS GENERATED -->\n<!-- BEGIN HARNESS GENERATED -->\n",
		"no END":           "<!-- BEGIN HARNESS GENERATED -->\n",
		"END with a tail":  "<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED --> x\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := replaceCatalogRegion(file, "<!-- B -->", ""); ok {
				t.Error("an invalid region was accepted")
			}
		})
	}
}

func catalogRepo(t *testing.T) (repo, home, file string) {
	t.Helper()
	repo, home = skillsRepo(t)
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"skills":{
		"vault_subpath":"00_meta/skills","record_dir":"harness/skills","deploy":[],
		"catalog":{"agent":"copilot","file":".copilot/copilot-instructions.md"}}}`)
	file = filepath.Join(home, ".copilot", "copilot-instructions.md")
	return repo, home, file
}

func TestDeploySkills_WritesTheCatalogIntoItsRegion(t *testing.T) {
	repo, home, file := catalogRepo(t)
	writeFile(t, file, "# Copilot\n\n<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n")
	out, err := DeploySkills(repo, home, none, false)
	if err != nil {
		t.Fatal(err)
	}
	bullets := "- **demo** — a demo skill\n"
	want := "# Copilot\n\n" + GeneratedBeginPrefix + " (sha256:" + PresenceSHA(bullets) +
		") — skill catalog from vault 00_meta/skills; edit there + re-run setup, do NOT edit between markers -->\n" +
		bullets + GeneratedEndMarker + "\n"
	if got, _ := os.ReadFile(file); string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if !hasCatalogOutcome(out, SkillDeployed) {
		t.Errorf("outcomes: %+v", out)
	}
	again, err := DeploySkills(repo, home, none, false)
	if err != nil || !hasCatalogOutcome(again, SkillCurrent) {
		t.Errorf("second run: %+v, %v", again, err)
	}
}

func TestDeploySkills_SkipsAnAbsentOrUnmarkedCatalogFile(t *testing.T) {
	repo, home, file := catalogRepo(t)
	out, err := DeploySkills(repo, home, none, false)
	if err != nil || !hasCatalogOutcome(out, SkillSkipped) {
		t.Fatalf("absent: %+v, %v", out, err)
	}
	if _, err := os.Stat(file); err == nil {
		t.Error("the catalog created a file that was absent")
	}
	writeFile(t, file, "# Copilot, no region\n")
	if out, err := DeploySkills(repo, home, none, false); err != nil || !hasCatalogOutcome(out, SkillSkipped) {
		t.Errorf("unmarked: %+v, %v", out, err)
	}
	if got, _ := os.ReadFile(file); string(got) != "# Copilot, no region\n" {
		t.Error("an unmarked file was rewritten")
	}
}

func TestDeploySkills_DryRunLeavesTheCatalogFile(t *testing.T) {
	repo, home, file := catalogRepo(t)
	seed := "<!-- BEGIN HARNESS GENERATED -->\n<!-- END HARNESS GENERATED -->\n"
	writeFile(t, file, seed)
	out, err := DeploySkills(repo, home, none, true)
	if err != nil || !hasCatalogOutcome(out, SkillDeployed) {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	if got, _ := os.ReadFile(file); string(got) != seed {
		t.Error("a dry run wrote the catalog")
	}
}

func hasCatalogOutcome(out []SkillOutcome, status string) bool {
	for _, o := range out {
		if o.Catalog && o.Status == status && strings.HasSuffix(o.Path, "copilot-instructions.md") {
			return true
		}
	}
	return false
}

// The shell's `*/` glob compared `name/`, so `a-b/` sorted before `a/`; the
// catalog walks names in byte order on every OS.
func TestCatalogBullets_ByteOrder(t *testing.T) {
	recDir := t.TempDir()
	for _, n := range []string{"a-b", "a"} {
		writeFile(t, filepath.Join(recDir, n, "SKILL.md"), "---\nname: "+n+"\ndescription: d\n---\n")
	}
	names, err := recordNames(recDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalogBullets(recDir, names, "copilot"); got != "- **a** — d\n- **a-b** — d\n" {
		t.Errorf("got:\n%s", got)
	}
}
