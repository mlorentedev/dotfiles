package env

import (
	"path"
	"strings"
)

// ResolvedVar is one fully-resolved path: an env-var name and its absolute
// value (home expanded, override applied).
type ResolvedVar struct {
	Name  string
	Value string
}

type pathResolver struct {
	vars    map[string]contractVar
	machine *machine
	goos    string
	home    string
	cache   map[string]string
	done    map[string]bool
}

// Resolve walks the contract in declaration order and produces the resolved
// value for every var that has a default for this OS. Vars without a default
// (HOME, USERPROFILE — OS-provided) are skipped: the generated file must never
// clobber them. The per-machine override (machine.json) wins over the default;
// rule #1 of the cascade (an explicit env var already set) is enforced by the
// generated file itself via `${VAR:-…}` / `if (-not $env:VAR)`, not here.
func Resolve(c *contract, m *machine, goos, home string) []ResolvedVar {
	byName := make(map[string]contractVar, len(c.EnvVars))
	for _, v := range c.EnvVars {
		byName[v.Name] = v
	}
	r := pathResolver{
		vars: byName, machine: m, goos: goos, home: home,
		cache: make(map[string]string, len(c.EnvVars)),
		done:  make(map[string]bool, len(c.EnvVars)),
	}

	var out []ResolvedVar
	for _, v := range c.EnvVars {
		if value, ok := r.resolve(v.Name); ok {
			out = append(out, ResolvedVar{Name: v.Name, Value: value})
		}
	}
	return out
}

func (r *pathResolver) resolve(name string) (string, bool) {
	if r.done[name] {
		value, ok := r.cache[name]
		return value, ok
	}
	r.done[name] = true
	v, ok := r.vars[name]
	if !ok {
		return "", false
	}
	if r.machine != nil {
		if override := r.machine.Paths[name]; override != "" {
			r.cache[name] = override
			return override, true
		}
	}
	if v.DerivedFrom != nil {
		if base, found := r.resolve(v.DerivedFrom.Key); found {
			value := joinForOS(r.goos, base, v.DerivedFrom.Path)
			r.cache[name] = value
			return value, true
		}
	}
	raw, _, _ := ForOS(v.Default, r.goos)
	if raw == "" {
		return "", false
	}
	value := expand(raw, r.home)
	r.cache[name] = value
	return value, true
}

func joinForOS(goos, base, child string) string {
	child = strings.Trim(strings.ReplaceAll(child, `\`, "/"), "/")
	if goos != "windows" {
		return path.Join(strings.ReplaceAll(base, `\`, "/"), child)
	}
	base = strings.TrimRight(base, `\/`)
	return base + `\` + strings.ReplaceAll(child, "/", `\`)
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
