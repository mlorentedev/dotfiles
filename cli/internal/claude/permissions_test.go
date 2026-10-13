package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMalformedRule(t *testing.T) {
	for rule, want := range map[string]bool{
		"Bash(git push *--force*:*)":  true,
		"Bash(rm -rf ~/*:*)":          true,
		"Bash(*dotf secrets show*:*)": true,
		"Bash(git push *--force*)":    false, // wildcard form
		"Bash(git status:*)":          false, // prefix form
		"Bash(git status)":            false,
		"Read(~/.ssh/**)":             false,
		"WebFetch":                    false,
		"Bash(:*)":                    false,
		"Bash(a*:*":                   false, // no closing paren: not a Tool(...) rule
	} {
		if got := MalformedRule(rule); got != want {
			t.Errorf("MalformedRule(%q) = %v, want %v", rule, got, want)
		}
	}
}

// Every list is swept, every other rule keeps its place, and a non-string
// entry is left alone.
func TestDropMalformedRules_KeepsEveryOtherRuleInOrder(t *testing.T) {
	doc := map[string]any{"permissions": map[string]any{
		"deny":        []any{"Bash(rm -rf ~/*:*)", "Bash(rm -rf ~/*)", "Bash(git reset --hard*:*)", 7.0},
		"allow":       []any{"Bash(git status:*)", "Bash(ls *:*)"},
		"ask":         []any{"Bash(npm publish:*)"},
		"defaultMode": "default",
	}}
	if got, want := MalformedRules(doc), []string{
		"allow: Bash(ls *:*)",
		"deny: Bash(rm -rf ~/*:*)",
		"deny: Bash(git reset --hard*:*)",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("MalformedRules = %q, want %q", got, want)
	}
	if !DropMalformedRules(doc) {
		t.Fatal("DropMalformedRules reported no change")
	}
	perms := doc["permissions"].(map[string]any)
	if got, want := perms["deny"], []any{"Bash(rm -rf ~/*)", 7.0}; !reflect.DeepEqual(got, want) {
		t.Errorf("deny = %v, want %v", got, want)
	}
	if got, want := perms["allow"], []any{"Bash(git status:*)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("allow = %v, want %v", got, want)
	}
	if got, want := perms["ask"], []any{"Bash(npm publish:*)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ask = %v, want %v", got, want)
	}
	if DropMalformedRules(doc) {
		t.Error("a second pass reported a change")
	}
	if DropMalformedRules(map[string]any{"model": "x"}) {
		t.Error("a document without permissions reported a change")
	}
}

// The template doctor's --fix compares against carries no rule it would
// remove: the predicate and tests/claude-settings-template.bats, which guards
// the template with a regex, have to agree on it.
func TestMalformedRules_TheTemplateCarriesNone(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "ai", "claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["permissions"].(map[string]any); !ok {
		t.Fatal("the template has no permissions block: this test checks nothing")
	}
	if bad := MalformedRules(doc); len(bad) > 0 {
		t.Errorf("ai/claude/settings.json carries rules Claude Code rejects: %q", bad)
	}
}
