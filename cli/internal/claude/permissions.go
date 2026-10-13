package claude

import "strings"

// permissionLists are the settings.json keys under "permissions" that hold
// rules.
var permissionLists = []string{"allow", "deny", "ask"}

// MalformedRule reports whether a permission rule is one Claude Code rejects
// at startup: a `Tool(...)` rule whose pattern ends in the `:*` prefix form
// and holds another `*`. Such a rule matches nothing, so as a deny rule it
// denies nothing (lesson 399, #2349).
func MalformedRule(rule string) bool {
	open, end := strings.IndexByte(rule, '('), len(rule)-1
	if open < 0 || end <= open || rule[end] != ')' {
		return false
	}
	pattern := rule[open+1 : end]
	prefix, ok := strings.CutSuffix(pattern, ":*")
	return ok && strings.Contains(prefix, "*")
}

// MalformedRules lists the malformed rules in a decoded settings document, as
// "<list>: <rule>", in list order.
func MalformedRules(doc map[string]any) []string {
	var found []string
	walkRules(doc, func(list string, rules []any) []any {
		for _, r := range rules {
			if s, ok := r.(string); ok && MalformedRule(s) {
				found = append(found, list+": "+s)
			}
		}
		return rules
	})
	return found
}

// DropMalformedRules removes every malformed rule from doc and reports whether
// it removed any. Every other rule, and its order, is kept.
func DropMalformedRules(doc map[string]any) bool {
	changed := false
	walkRules(doc, func(_ string, rules []any) []any {
		kept := rules[:0:0]
		for _, r := range rules {
			if s, ok := r.(string); ok && MalformedRule(s) {
				changed = true
				continue
			}
			kept = append(kept, r)
		}
		return kept
	})
	return changed
}

// walkRules calls f on each rule list under "permissions" and stores what it
// returns back in place.
func walkRules(doc map[string]any, f func(list string, rules []any) []any) {
	perms, ok := doc["permissions"].(map[string]any)
	if !ok {
		return
	}
	for _, list := range permissionLists {
		if rules, ok := perms[list].([]any); ok {
			perms[list] = f(list, rules)
		}
	}
}
