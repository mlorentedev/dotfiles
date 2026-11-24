package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// HARNESS-203 (#3612) gives agents the lease push as `dotf pr push-lease`
// instead of relaxing the force-push deny rules. These tests pin both halves in
// the template: the deny rules still refuse every raw forced push, and no
// deny, allow or ask rule names the command, so it runs under the ordinary
// permission prompt.

// leaseRuleMatches emulates how Claude Code matches a Bash(...) rule against a
// command, erring towards a match: "prefix:*" is a plain prefix, any other "*"
// matches any run of characters, spaces included, and the rest is literal.
func leaseRuleMatches(rule, command string) bool {
	body, ok := strings.CutPrefix(rule, "Bash(")
	if !ok || !strings.HasSuffix(body, ")") {
		return false
	}
	body = strings.TrimSuffix(body, ")")
	if prefix, ok := strings.CutSuffix(body, ":*"); ok {
		return strings.HasPrefix(command, prefix)
	}
	pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(body), `\*`, ".*") + "$"
	return regexp.MustCompile(pattern).MatchString(command)
}

func templatePermissions(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ai", "claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Permissions map[string][]string `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Permissions["deny"]) == 0 {
		t.Fatal("the template has no deny rules: these tests check nothing")
	}
	return doc.Permissions
}

const leaseSHA = "0123456789abcdef0123456789abcdef01234567"

func TestPushLeaseRules_RawForcedPushStaysDenied(t *testing.T) {
	deny := templatePermissions(t)["deny"]
	for _, want := range []string{"Bash(git push *--force*)", "Bash(git push -f*)", "Bash(git push * -f*)"} {
		found := false
		for _, rule := range deny {
			found = found || rule == want
		}
		if !found {
			t.Errorf("deny lost %s; HARNESS-203 moves the lease push into dotf instead of relaxing it", want)
		}
	}
	for _, raw := range []string{
		"git push --force-with-lease=refs/heads/feat/x:" + leaseSHA + " origin HEAD:refs/heads/feat/x",
		"git push origin --force-with-lease feat/x",
		"git push -f origin feat/x",
	} {
		denied := false
		for _, rule := range deny {
			denied = denied || leaseRuleMatches(rule, raw)
		}
		if !denied {
			t.Errorf("no deny rule matches %q", raw)
		}
	}
}

func TestPushLeaseRules_NoRuleNamesTheCommand(t *testing.T) {
	perms := templatePermissions(t)
	commands := []string{
		"dotf pr push-lease --expect " + leaseSHA,
		"dotf pr push-lease --expect " + leaseSHA + " --remote origin",
		"dotf pr push-lease --remote origin --expect " + leaseSHA,
	}
	for _, list := range []string{"deny", "allow", "ask"} {
		for _, rule := range perms[list] {
			for _, command := range commands {
				if leaseRuleMatches(rule, command) {
					t.Errorf("%s rule %s matches %q; the command must run under the ordinary prompt", list, rule, command)
				}
			}
		}
	}
}

// The matcher is what the two tests above trust, so it must match where
// Claude Code would.
func TestPushLeaseRules_MatcherMatches(t *testing.T) {
	tests := []struct {
		rule, command string
		want          bool
	}{
		{"Bash(dotf spec:*)", "dotf spec init X-1", true},
		{"Bash(dotf spec:*)", "dotf pr push-lease", false},
		{"Bash(git push *--force*)", "git push origin --force-with-lease=a:b", true},
		{"Bash(git push *--force*)", "git pushed --force", false},
		{"Bash(*--show-token*)", "gh auth status --show-token", true},
		{"Bash(git push -f*)", "git push -f", true},
		{"Bash(git push -f*)", "dotf pr push-lease --expect x", false},
		{"Read(./x)", "x", false},
	}
	for _, tt := range tests {
		if got := leaseRuleMatches(tt.rule, tt.command); got != tt.want {
			t.Errorf("leaseRuleMatches(%q, %q) = %v, want %v", tt.rule, tt.command, got, tt.want)
		}
	}
}
