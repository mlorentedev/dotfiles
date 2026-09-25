package doctor

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// triggerTargetsFixture builds a deploy dir holding a triggers file and a vault
// holding the named patterns, and returns the System and Config that point at
// them.
func triggerTargetsFixture(t *testing.T, triggersJSON string, vaultPatterns ...string) (*System, *Config) {
	t.Helper()
	deploy := t.TempDir()
	if triggersJSON != "" {
		mkdirAll(t, filepath.Join(deploy, "harness"))
		writeFile(t, filepath.Join(deploy, "harness", "triggers.json"), triggersJSON)
	}
	vault := t.TempDir()
	if len(vaultPatterns) > 0 {
		mkdirAll(t, filepath.Join(vault, "00_meta", "patterns"))
		for _, p := range vaultPatterns {
			writeFile(t, filepath.Join(vault, "00_meta", "patterns", p+".md"), "# "+p+"\n")
		}
	}
	sys := &System{Getenv: func(k string) string {
		if k == "VAULT_PATH" {
			return vault
		}
		return ""
	}}
	return sys, &Config{DotfilesDir: deploy}
}

const twoTriggers = `{"version":1,"triggers":[
  {"id":"golang","pattern":"pattern-language-standards","globs":["*.go"]},
  {"id":"iac","pattern":"pattern-terraform-standards","globs":["*.tf"]}]}`

// A trigger naming a pattern the vault lacks must FAIL and name both halves, so
// the fix is one edit rather than a search.
func TestCheckTriggerTargets_DanglingTriggerFailsAndIsNamed(t *testing.T) {
	sys, cfg := triggerTargetsFixture(t, twoTriggers, "pattern-language-standards")
	var buf bytes.Buffer
	rep := capture(&buf)

	checkTriggerTargets(sys, cfg, rep)

	if rep.Failures() != 1 {
		t.Fatalf("a dangling trigger must FAIL once, got %d\n%s", rep.Failures(), buf.String())
	}
	for _, want := range []string{"1 of 2", "iac -> pattern-terraform-standards"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("FAIL should mention %q, got: %s", want, buf.String())
		}
	}
	if strings.Contains(buf.String(), "golang ->") {
		t.Errorf("a trigger whose pattern exists must not be listed, got: %s", buf.String())
	}
}

func TestCheckTriggerTargets_EveryPatternPresentPasses(t *testing.T) {
	sys, cfg := triggerTargetsFixture(t, twoTriggers, "pattern-language-standards", "pattern-terraform-standards")
	var buf bytes.Buffer
	rep := capture(&buf)

	checkTriggerTargets(sys, cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("all patterns present must not FAIL, got %d\n%s", rep.Failures(), buf.String())
	}
	if !strings.Contains(buf.String(), "all 2 triggers name a pattern the vault has") {
		t.Errorf("want the PASS, got: %s", buf.String())
	}
}

// Without the vault the check has nothing to compare against. That is a SKIP: a
// PASS would claim the triggers were sound on a machine that never looked.
func TestCheckTriggerTargets_NoVaultSkipsRatherThanPasses(t *testing.T) {
	sys, cfg := triggerTargetsFixture(t, twoTriggers) // no patterns dir at all
	var buf bytes.Buffer
	rep := capture(&buf)

	checkTriggerTargets(sys, cfg, rep)

	if rep.Failures() != 0 {
		t.Fatalf("an absent vault must not FAIL, got %d", rep.Failures())
	}
	if strings.Contains(buf.String(), "name a pattern the vault has") {
		t.Errorf("an absent vault must not report the triggers as sound, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "unchecked") {
		t.Errorf("want a SKIP saying the targets were not checked, got: %s", buf.String())
	}
}

func TestCheckTriggerTargets_NoTriggersFileSkips(t *testing.T) {
	sys, cfg := triggerTargetsFixture(t, "", "pattern-language-standards")
	var buf bytes.Buffer
	rep := capture(&buf)

	checkTriggerTargets(sys, cfg, rep)

	if rep.Failures() != 0 || !strings.Contains(buf.String(), "unchecked") {
		t.Fatalf("no triggers file must SKIP, got failures=%d: %s", rep.Failures(), buf.String())
	}
}

// A file that does not parse is a defect in the thing being checked, not an
// absence: reading it as "nothing to check" would hide a broken router.
func TestCheckTriggerTargets_UnparseableTriggersFail(t *testing.T) {
	sys, cfg := triggerTargetsFixture(t, "{not json", "pattern-language-standards")
	var buf bytes.Buffer
	rep := capture(&buf)

	checkTriggerTargets(sys, cfg, rep)

	if rep.Failures() != 1 || !strings.Contains(buf.String(), "do not parse") {
		t.Fatalf("unparseable triggers must FAIL, got failures=%d: %s", rep.Failures(), buf.String())
	}
}

// HARNESS-148: a deployed document that names no pattern has shown nothing about
// its targets, so the check SKIPS instead of reporting "all 0 triggers name a
// pattern the vault has". That line was a PASS with nothing checked, the
// vacuous-pass shape lessons 287 and 292 name. All three ways of naming nothing
// are covered: no rules, every pattern empty, and no pattern key at all.
func TestCheckTriggerTargets_NamingNoPatternSkipsRatherThanPasses(t *testing.T) {
	for name, doc := range map[string]string{
		"no rules":       `{"version":1,"triggers":[]}`,
		"empty patterns": `{"version":1,"triggers":[{"id":"a","pattern":"","globs":["*.a"]}]}`,
		"no pattern key": `{"version":1,"triggers":[{"id":"a","globs":["*.a"]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			sys, cfg := triggerTargetsFixture(t, doc, "pattern-language-standards")
			var buf bytes.Buffer
			rep := capture(&buf)

			checkTriggerTargets(sys, cfg, rep)

			if rep.Failures() != 0 {
				t.Fatalf("naming no pattern must not FAIL, got %d\n%s", rep.Failures(), buf.String())
			}
			if strings.Contains(buf.String(), "name a pattern the vault has") {
				t.Errorf("a document naming no pattern must not read as sound, got: %s", buf.String())
			}
			if !strings.Contains(buf.String(), "unchecked") {
				t.Errorf("want a SKIP saying the targets were not checked, got: %s", buf.String())
			}
		})
	}
}
