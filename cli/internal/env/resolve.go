package env

import "strings"

// ResolvedVar is one fully-resolved path: an env-var name and its absolute
// value (home expanded, override applied).
type ResolvedVar struct {
	Name  string
	Value string
}

// Resolve walks the contract in declaration order and produces the resolved
// value for every var that has a default for this OS. Vars without a default
// (HOME, USERPROFILE — OS-provided) are skipped: the generated file must never
// clobber them. The per-machine override (machine.json) wins over the default;
// rule #1 of the cascade (an explicit env var already set) is enforced by the
// generated file itself via `${VAR:-…}` / `if (-not $env:VAR)`, not here.
func Resolve(c *contract, m *machine, goos, home string) []ResolvedVar {
	var out []ResolvedVar
	for _, v := range c.EnvVars {
		raw := defaultFor(v.Default, goos)
		if raw == "" {
			continue
		}
		val := raw
		if m != nil {
			if ov, ok := m.Paths[v.Name]; ok && ov != "" {
				val = ov
			}
		}
		out = append(out, ResolvedVar{Name: v.Name, Value: expand(val, home)})
	}
	return out
}

// defaultFor selects the default for goos. darwin falls back to the linux
// (POSIX) default when no explicit darwin key exists.
func defaultFor(def map[string]string, goos string) string {
	if def == nil {
		return ""
	}
	if v, ok := def[goos]; ok && v != "" {
		return v
	}
	if goos == "darwin" {
		if v, ok := def["linux"]; ok {
			return v
		}
	}
	return ""
}

// expand substitutes the home placeholders used across the contract's per-OS
// defaults with the resolved home dir, yielding an absolute path. Override
// values from machine.json are already absolute, so expand is a no-op on them.
func expand(s, home string) string {
	for _, k := range []string{"${HOME}", "$HOME", "%USERPROFILE%", "$env:USERPROFILE"} {
		s = strings.ReplaceAll(s, k, home)
	}
	return s
}
