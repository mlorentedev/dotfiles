package harness

import (
	"path/filepath"
	"strings"
	"testing"
)

// checkRepo is a checkout with one enforced record, one target holding its
// region, and one skill and one agent record that validate.
func checkRepo(t *testing.T) string {
	t.Helper()
	repo := enforcedRepo(t, map[string]string{"rule": "Rule.\n<!-- full-only:begin -->\nwhy\n<!-- full-only:end -->\n\n\n"})
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{
		"enforced":[{"id":"rule"},{"id":"spare","opt_out":{"T.md":"not here","doctrine":"nor here"}}],
		"targets":[{"file":"T.md","inject":["rule"]}],
		"doctrine":{"inject":["rule"]},
		"skills":{"record_dir":"harness/skills","schema":"s.json"},
		"agents":{"record_dir":"harness/agents","schema":"a.json"}}`)
	writeFile(t, filepath.Join(repo, "T.md"), "top\n"+GeneratedBeginPrefix+" (sha256:0) -->\nRule.\nwhy\n"+GeneratedEndMarker+"\ntail\n")
	writeFile(t, filepath.Join(repo, "s.json"), `{"required":["name","description"]}`)
	writeFile(t, filepath.Join(repo, "a.json"), `{"required":["name","kind"]}`)
	writeFile(t, filepath.Join(repo, "harness", "skills", "demo", "SKILL.md"), "---\nname: demo\ndescription: d\n---\nbody\n")
	writeFile(t, filepath.Join(repo, "harness", "agents", "demo", "AGENT.md"), "---\nname: demo\nkind: invocable\n---\n")
	return repo
}

func problems(report []CheckLine) []string {
	var out []string
	for _, l := range report {
		if l.Problem {
			out = append(out, l.Text)
		}
	}
	return out
}

func TestCheckDrift_ACleanCheckoutPasses(t *testing.T) {
	report, err := CheckDrift(checkRepo(t))
	if err != nil || HasDrift(report) {
		t.Fatalf("problems: %v, %v", problems(report), err)
	}
	if len(report) != 5 || !strings.Contains(report[0].Text, "spare excluded from T.md (not here)") {
		t.Errorf("report: %+v", report)
	}
}

func TestCheckDrift_EachKindOfDriftIsAProblem(t *testing.T) {
	cases := []struct{ name, file, body, want string }{
		{"an edited region", "T.md", "x\n" + GeneratedBeginPrefix + " -->\nRule!\nwhy\n" + GeneratedEndMarker + "\n", "[DRIFT] T.md"},
		{"two BEGIN lines", "T.md", GeneratedBeginPrefix + "\n" + GeneratedBeginPrefix + "\n" + GeneratedEndMarker + "\n", "(found 2/1)"},
		{"END before BEGIN", "T.md", GeneratedEndMarker + "\n" + GeneratedBeginPrefix + "\n", "END marker precedes"},
		{"an END line with a tail", "T.md", GeneratedBeginPrefix + "\nRule.\nwhy\n" + GeneratedEndMarker + " x\n", "[DRIFT] T.md"},
		{"no frontmatter", "harness/skills/demo/SKILL.md", "body\n", "missing YAML frontmatter"},
		{"an unclosed frontmatter", "harness/skills/demo/SKILL.md", "---\nname: demo\n", "unterminated frontmatter"},
		{"an empty required key", "harness/agents/demo/AGENT.md", "---\nname: demo\nkind:   \n---\n", `"kind" missing or empty`},
		{"a key only in the body", "harness/agents/demo/AGENT.md", "---\nname: demo\n---\nkind: x\n", `"kind" missing or empty`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := checkRepo(t)
			writeFile(t, filepath.Join(repo, filepath.FromSlash(tc.file)), tc.body)
			report, err := CheckDrift(repo)
			if err != nil {
				t.Fatal(err)
			}
			if p := strings.Join(problems(report), "\n"); !strings.Contains(p, tc.want) {
				t.Errorf("want %q among:\n%s", tc.want, p)
			}
		})
	}
}

func TestCheckDrift_ACRLFRecordValidatesAsLF(t *testing.T) {
	repo := checkRepo(t)
	writeFile(t, filepath.Join(repo, "harness", "agents", "demo", "AGENT.md"), "---\r\nname: demo\r\nkind: invocable\r\n---\r\n")
	report, err := CheckDrift(repo)
	if err != nil || HasDrift(report) {
		t.Errorf("problems: %v, %v", problems(report), err)
	}
}

func TestCheckDrift_ACRLFTargetMatchesItsRender(t *testing.T) {
	repo := checkRepo(t)
	writeFile(t, filepath.Join(repo, "T.md"), "top\r\n"+GeneratedBeginPrefix+" (sha256:0) -->\r\nRule.\r\nwhy\r\n"+GeneratedEndMarker+"\r\ntail\r\n")
	report, err := CheckDrift(repo)
	if err != nil || HasDrift(report) {
		t.Errorf("problems: %v, %v", problems(report), err)
	}
}

func TestCheckDrift_ASchemaProblemNamesItsSchema(t *testing.T) {
	repo := checkRepo(t)
	writeFile(t, filepath.Join(repo, "a.json"), "{not json")
	report, err := CheckDrift(repo)
	if err != nil {
		t.Fatal(err)
	}
	if p := strings.Join(problems(report), "\n"); !strings.Contains(p, "[ERROR] agent schema ") || !strings.Contains(p, "not valid JSON") {
		t.Errorf("problems:\n%s", p)
	}
}

func TestCheckDrift_AGapNamesTheIdAndTheSurface(t *testing.T) {
	repo := checkRepo(t)
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"enforced":[{"id":"rule"}],
		"targets":[{"file":"T.md","inject":["rule"]}],"doctrine":{"inject":[]}}`)
	report, err := CheckDrift(repo)
	if err != nil {
		t.Fatal(err)
	}
	if p := problems(report); len(p) != 1 || p[0] != `[GAP] enforced region "rule" reaches neither surface "doctrine" nor an opt_out for it` {
		t.Errorf("problems: %v", p)
	}
}

func TestFirstDifference(t *testing.T) {
	d := firstDifference("a\nb\n", "a\nc\nd\n")
	if len(d) != 3 || !strings.Contains(d[0], "line 2") || d[1] != "    - b" || d[2] != "    + c" {
		t.Errorf("got %q", d)
	}
	if d := firstDifference("a\n", "a\nb\n"); d[1] != "    - <end of region>" {
		t.Errorf("a short region: %q", d)
	}
}
