package env

// The env-contract keys every per-OS value by GOOS: a default, a list of PATH
// entries, a required_on scope. darwin is a key of its own, and where an entry
// declares none it reads the linux one: both are POSIX ($HOME paths, the sh
// profiles), so a darwin key carries only what differs and never a copy of the
// linux value (ADR-045 decision 7, #2013 P2). windows inherits nothing. Every
// reader of the contract resolves a key through this file, so the rule has one
// definition.

// ContractOS is the contract key goos reads first: goos itself, with "" (a
// System built without a GOOS, as tests do) read as linux.
func ContractOS(goos string) string {
	if goos == "" {
		return "linux"
	}
	return goos
}

// FallbackOS is the key goos reads when an entry declares none of its own:
// "linux" for darwin, "" (none) for every other OS.
func FallbackOS(goos string) string {
	if ContractOS(goos) == "darwin" {
		return "linux"
	}
	return ""
}

// ForOS returns m's value for goos and the key it came from: goos's own key
// when m declares it, even with an empty value (so a darwin "" opts darwin out
// of a linux default), else the fallback's. ok is false when neither is
// declared.
func ForOS[T any](m map[string]T, goos string) (v T, key string, ok bool) {
	for _, k := range []string{ContractOS(goos), FallbackOS(goos)} {
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			return v, k, true
		}
	}
	return v, "", false
}

// AppliesOn reports whether an entry scoped to scope (a contract key; "" is
// every OS) applies on goos. A linux-scoped entry applies on darwin, by the
// same inheritance as a value.
func AppliesOn(scope, goos string) bool {
	return scope == "" || scope == ContractOS(goos) || scope == FallbackOS(goos)
}
