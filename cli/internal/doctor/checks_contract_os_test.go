package doctor

import (
	"bytes"
	"strings"
	"testing"
)

// TestContractScopeNamesTheOSItReads proves the section header says which
// contract key this OS reads, and for darwin that undeclared keys read linux:
// before #2013 P2 darwin was mapped to linux silently, and its report named no
// OS at all.
func TestContractScopeNamesTheOSItReads(t *testing.T) {
	for goos, want := range map[string]string{
		"windows": ", windows",
		"linux":   ", linux",
		"darwin":  ", darwin; undeclared keys read linux",
		"":        ", linux",
	} {
		if got := contractScope(&System{GOOS: goos}); got != want {
			t.Errorf("contractScope(GOOS=%q) = %q, want %q", goos, got, want)
		}
	}
}

// TestCheckContractPath_Dialects proves the PATH-entries check reads the OS
// dialect for the running platform, not a hardcoded "linux" — the root cause of
// the false-positive drift banner on Windows (Linux entries expanded with a
// Windows home).
func TestCheckContractPath_Dialects(t *testing.T) {
	contract := &Contract{RequiredPathEntries: map[string][]string{
		"linux":   {"$HOME/.local/bin"},
		"windows": {`$env:USERPROFILE\scripts`},
	}}

	t.Run("linux dialect", func(t *testing.T) {
		home := t.TempDir()
		// expandHome does literal token substitution, so the expanded entry keeps
		// the contract's forward slash ("$HOME/.local/bin") — build the PATH entry
		// the same way rather than with filepath.Join, whose separator is host-OS.
		linuxEntry := home + "/.local/bin"
		env := map[string]string{"HOME": home, "PATH": linuxEntry}
		var buf bytes.Buffer
		checkContractPath(newSys(env, nil, nil), contract, capture(&buf)) // GOOS "" → linux
		out := buf.String()
		if !strings.Contains(out, linuxEntry+" in PATH") {
			t.Errorf("linux entry should be checked + pass\n%s", out)
		}
		if strings.Contains(out, "scripts") {
			t.Errorf("windows entry must NOT be checked on linux\n%s", out)
		}
	})

	t.Run("windows dialect", func(t *testing.T) {
		home := t.TempDir() // POSIX temp dir → no drive colon to collide with the ':' list separator on the test host
		winEntry := home + `\scripts`
		env := map[string]string{"USERPROFILE": home, "PATH": winEntry}
		s := newSys(env, nil, nil)
		s.GOOS = "windows"
		var buf bytes.Buffer
		checkContractPath(s, contract, capture(&buf))
		out := buf.String()
		if !strings.Contains(out, winEntry+" in PATH") {
			t.Errorf("windows entry should be checked + pass (and $env:USERPROFILE expanded)\n%s", out)
		}
		if strings.Contains(out, ".local/bin") {
			t.Errorf("linux entry must NOT be checked on windows\n%s", out)
		}
	})
}

// TestCheckContractEnvVars_WindowsDialect proves env-var defaults and OS scoping
// follow the running platform: on Windows a linux-scoped var skips, the
// windows-scoped var validates, and a defaulted var resolves its WINDOWS default.
func TestCheckContractEnvVars_WindowsDialect(t *testing.T) {
	home := t.TempDir()
	contract := &Contract{EnvVars: []ContractEnvVar{
		{Name: "HOME", RequiredOn: "linux", Validation: "path_exists"},
		{Name: "USERPROFILE", RequiredOn: "windows", Validation: "path_exists"},
		{Name: "DOTFILES_DIR", Default: map[string]string{
			"linux":   "$HOME/linuxonly",
			"windows": `$env:USERPROFILE\winonly`,
		}, Validation: "path_exists"},
	}}
	env := map[string]string{"USERPROFILE": home} // DOTFILES_DIR unset → falls back to the dialect default
	s := newSys(env, nil, nil)
	s.GOOS = "windows"

	var buf bytes.Buffer
	checkContractEnvVars(s, contract, capture(&buf), false)
	out := buf.String()

	if !strings.Contains(out, "HOME (linux-scoped, skipped on windows)") {
		t.Errorf("a linux-scoped var must SKIP on windows\n%s", out)
	}
	if !strings.Contains(out, "USERPROFILE="+home) {
		t.Errorf("the windows-scoped USERPROFILE must validate on windows\n%s", out)
	}
	if !strings.Contains(out, "winonly") || strings.Contains(out, "linuxonly") {
		t.Errorf("DOTFILES_DIR must resolve the WINDOWS default, not the linux one\n%s", out)
	}
}

// TestCheckContractOnDarwin proves darwin reads its own key where the contract
// declares one and the linux key where it does not, and that a linux-scoped
// var (HOME) is checked on darwin rather than skipped.
func TestCheckContractOnDarwin(t *testing.T) {
	home := t.TempDir()
	darwinSys := func(env map[string]string) *System {
		s := newSys(env, nil, nil)
		s.GOOS = "darwin"
		return s
	}

	t.Run("a darwin key wins over linux", func(t *testing.T) {
		contract := &Contract{RequiredPathEntries: map[string][]string{
			"linux":  {"$HOME/linuxonly"},
			"darwin": {"$HOME/darwinonly"},
		}}
		var buf bytes.Buffer
		checkContractPath(darwinSys(map[string]string{"HOME": home, "PATH": home + "/darwinonly"}), contract, capture(&buf))
		out := buf.String()
		if !strings.Contains(out, home+"/darwinonly in PATH") || strings.Contains(out, "linuxonly") {
			t.Errorf("darwin must read its own entries, not linux's\n%s", out)
		}
		if !strings.Contains(out, "PATH entries (contract, darwin; undeclared keys read linux)") {
			t.Errorf("the header must name darwin\n%s", out)
		}
	})

	t.Run("no darwin key reads linux", func(t *testing.T) {
		contract := &Contract{
			RequiredPathEntries: map[string][]string{"linux": {"$HOME/.local/bin"}},
			EnvVars: []ContractEnvVar{
				{Name: "HOME", RequiredOn: "linux", Validation: "path_exists"},
				{Name: "USERPROFILE", RequiredOn: "windows"},
				{Name: "DOTFILES_DIR", Default: map[string]string{"linux": "$HOME/posix"}},
			},
		}
		s := darwinSys(map[string]string{"HOME": home, "PATH": home + "/.local/bin"})
		var buf bytes.Buffer
		checkContractPath(s, contract, capture(&buf))
		checkContractEnvVars(s, contract, capture(&buf), false)
		out := buf.String()
		for _, want := range []string{
			home + "/.local/bin in PATH",
			"HOME=" + home + " (path exists)",
			"USERPROFILE (windows-scoped, skipped on darwin)",
			home + "/posix",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q\n%s", want, out)
			}
		}
	})
}

// TestCheckContractPathWarnsOnAnOSWithNoKey: an OS the contract declares no
// entries for (directly or by fallback) warns, rather than printing an empty
// section that reads as a clean PATH.
func TestCheckContractPathWarnsOnAnOSWithNoKey(t *testing.T) {
	contract := &Contract{RequiredPathEntries: map[string][]string{"linux": {"$HOME/.local/bin"}}}
	s := newSys(map[string]string{"HOME": t.TempDir()}, nil, nil)
	s.GOOS = "freebsd"
	var buf bytes.Buffer
	checkContractPath(s, contract, capture(&buf))
	if out := buf.String(); !strings.Contains(out, "no required PATH entries declared for freebsd") {
		t.Errorf("want a WARN naming the OS with no key\n%s", out)
	}
}
