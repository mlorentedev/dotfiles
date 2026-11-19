package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// HARNESS-201. The deny globs of ai/claude/settings.json are the first layer in
// front of `dotf harness push-guard`, and the only layer left when that hook is
// down (a Claude hook that crashes or whose binary is missing fails open). They
// must therefore stay exact: refuse every plain force, and not refuse the one
// lease form the hook verifies, because deny wins over any hook decision.
//
// The matcher below emulates Claude's documented Bash(...) rule syntax: `*`
// matches any run of characters, and a trailing `:*` is a word-boundary prefix.
// It is NOT Claude's own matcher, which is not available to a test; a row that
// passes here proves the rule text says what it is meant to, not that Claude
// reads it so. The hook, tested separately, does not depend on it.

func loadClaudeTemplate(t *testing.T) (allow, deny []string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ai", "claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Permissions struct {
			Allow []string `json:"allow"`
			Deny  []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Permissions.Allow, doc.Permissions.Deny
}

// bashRuleMatches reports whether a `Bash(<pattern>)` rule matches command.
func bashRuleMatches(rule, command string) bool {
	inner, ok := strings.CutPrefix(rule, "Bash(")
	if !ok || !strings.HasSuffix(inner, ")") {
		return false
	}
	pattern := strings.TrimSuffix(inner, ")")
	if prefix, isPrefix := strings.CutSuffix(pattern, ":*"); isPrefix {
		return command == prefix || strings.HasPrefix(command, prefix+" ")
	}
	parts := strings.Split(pattern, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$").MatchString(command)
}

func deniedBy(deny []string, command string) []string {
	var hit []string
	for _, r := range deny {
		if bashRuleMatches(r, command) {
			hit = append(hit, r)
		}
	}
	return hit
}

// The exact set of git-push and landing deny rules. A later edit that widens or
// drops one fails here and has to say so in the diff.
func TestClaudeTemplate_PushDenyRulesArePinned(t *testing.T) {
	_, deny := loadClaudeTemplate(t)
	var got []string
	for _, r := range deny {
		if strings.Contains(r, "git push") || strings.Contains(r, "git * push") || strings.Contains(r, "land:push") {
			got = append(got, r)
		}
	}
	want := []string{
		"Bash(git push *--force *)",
		"Bash(git push *--force)",
		"Bash(git push -f*)",
		"Bash(git push * -f*)",
		"Bash(git push *--force-with-lease *)",
		"Bash(git push *--force-with-lease)",
		"Bash(git push *--force-with-lease=main:* *)",
		"Bash(git push *--force-with-lease=master:* *)",
		"Bash(git push *--mirror*)",
		"Bash(git * push *--force *)",
		"Bash(git * push *--force)",
		"Bash(git * push -f*)",
		"Bash(git * push * -f*)",
		"Bash(git * push *--force-with-lease *)",
		"Bash(git * push *--force-with-lease)",
		"Bash(npm run land:push*)",
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("push deny rules changed.\n got: %q\nwant: %q", got, want)
	}
	for _, r := range deny {
		if MalformedRule(r) {
			t.Errorf("deny rule %q is malformed and would deny nothing", r)
		}
	}
}

// Nothing in allow may grant a push: the lease form is "not denied", and the
// permission flow decides about the prompt. A rule here would also bypass the
// hook's refusals if the hook were down.
func TestClaudeTemplate_AllowGrantsNoPush(t *testing.T) {
	allow, _ := loadClaudeTemplate(t)
	for _, r := range allow {
		low := strings.ToLower(r)
		if strings.Contains(low, "push") || strings.Contains(low, "force") || strings.Contains(low, "land:") {
			t.Errorf("allow rule %q grants a push or a landing", r)
		}
		if r == "Bash" || r == "Bash(*)" || r == "Bash(git:*)" || r == "Bash(git *)" || r == "Bash(npm:*)" || r == "Bash(npm run:*)" {
			t.Errorf("allow rule %q is wide enough to grant a push", r)
		}
	}
}

const sha40 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestClaudeTemplate_DenyGlobsCatchPlainForcesAndSpareTheLease(t *testing.T) {
	_, deny := loadClaudeTemplate(t)
	lease := "--force-with-lease=feat/x:" + sha40
	denied := []string{
		"git push --force origin feat/x",
		"git push origin feat/x --force",
		"git push -f origin feat/x",
		"git push origin -f feat/x",
		"git push --force-with-lease origin feat/x",
		"git push origin feat/x --force-with-lease",
		"git push origin --force-with-lease=main:" + sha40 + " main",
		"git push --force-with-lease=master:" + sha40 + " origin master",
		"git push --mirror origin",
		"git -C /repo push --force origin feat/x",
		"git -C /repo push origin feat/x --force",
		"git -C /repo push -f origin feat/x",
		"git -c core.editor=true push --force-with-lease origin feat/x",
		"npm run land:push",
		"npm run land:push -- --dry-run",
	}
	for _, c := range denied {
		if len(deniedBy(deny, c)) == 0 {
			t.Errorf("%q is not denied by any rule", c)
		}
	}
	spared := []string{
		"git push origin " + lease + " feat/x",
		"git push " + lease + " origin feat/x",
		"git push origin feat/x " + lease,
		"git -C /repo push origin " + lease + " feat/x",
		"git push origin feat/x",
		"git push -u origin feat/x",
		"git push origin --delete old-branch",
		"npm run land",
		"npm test",
	}
	for _, c := range spared {
		if hit := deniedBy(deny, c); len(hit) != 0 {
			t.Errorf("%q is denied by %q: the lease form must reach the hook", c, hit)
		}
	}
}

// What the globs cannot see is the hook's job. Recorded as a test so nobody
// "fixes" the gap by widening a glob until it denies the lease too.
func TestClaudeTemplate_PlusRefspecIsLeftToTheHook(t *testing.T) {
	_, deny := loadClaudeTemplate(t)
	if hit := deniedBy(deny, "git push origin +feat/x"); len(hit) != 0 || len(deniedBy(deny, "git push -uf origin feat/x")) != 0 {
		t.Errorf("a +refspec is denied by %q; it belongs to the hook, and a glob wide enough to catch it also catches the lease", hit)
	}
}
