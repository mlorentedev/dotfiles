package pwsh

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseModules(t *testing.T) {
	cases := []struct {
		name string
		conf string
		want []Module
		err  string // substring; "" means no error
	}{
		{
			name: "marked pins in file order, unmarked pins ignored",
			conf: "GO_VERSION=1.26.0\n# pwsh-module: PSScriptAnalyzer\nPSSCRIPTANALYZER_VERSION=1.25.0\n# other comment\n# pwsh-module: Pester\r\nPESTER_VERSION=6.2.0\r\n",
			want: []Module{{Name: "PSScriptAnalyzer", Version: "1.25.0"}, {Name: "Pester", Version: "6.2.0"}},
		},
		{name: "no marker is no module", conf: "PESTER_VERSION=6.2.0\n"},
		{name: "marker at end of file", conf: "# pwsh-module: Pester", err: "line 1"},
		{name: "marker over a non-pin line", conf: "# pwsh-module: Pester\n# PESTER_VERSION=6.2.0\n", err: "line 1"},
		{name: "near-miss marker", conf: "GO_VERSION=1\n#pwsh-module: Pester\nPESTER_VERSION=6.2.0\n", err: "line 2"},
		{name: "near-miss marker in another case", conf: "# PWSH-MODULE: Pester\nPESTER_VERSION=6.2.0\n", err: "line 1"},
		{name: "a name PowerShell would read as code", conf: "# pwsh-module: Pester';rm\nPESTER_VERSION=6.2.0\n", err: "module name"},
		{name: "a version that is not one", conf: "# pwsh-module: Pester\nPESTER_VERSION=6.2.0'\n", err: "version"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseModules([]byte(c.conf))
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) {
					t.Fatalf("err = %v, want one containing %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// fakePwsh answers the import check with the modules named in out, and
// records every script it was handed.
type fakePwsh struct {
	out     string
	err     error
	scripts []string
}

func (f *fakePwsh) run(name string, args ...string) ([]byte, error) {
	if name != Command || len(args) != 4 || args[0] != "-NoProfile" || args[1] != "-NonInteractive" || args[2] != "-Command" {
		return nil, errors.New("unexpected invocation: " + name + " " + strings.Join(args, " "))
	}
	f.scripts = append(f.scripts, args[3])
	return []byte(f.out), f.err
}

var pins = []Module{{Name: "PSScriptAnalyzer", Version: "1.25.0"}, {Name: "Pester", Version: "6.2.0"}}

func TestMissing_NamesTheModulesThatDoNotImportAtTheirPin(t *testing.T) {
	f := &fakePwsh{out: "WARNING: noise\nMISSING Pester\n"}
	got, err := Missing(f.run, pins)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []Module{pins[1]}) {
		t.Errorf("got %v, want only Pester", got)
	}
	if len(f.scripts) != 1 || !strings.Contains(f.scripts[0], "Import-Module -Name 'PSScriptAnalyzer' -RequiredVersion '1.25.0'") {
		t.Errorf("one pwsh call importing each module at its pin, got %q", f.scripts)
	}
}

func TestMissing_APwshThatFailsIsAnErrorNotAnAnswer(t *testing.T) {
	f := &fakePwsh{err: errors.New("exit status 1")}
	if _, err := Missing(f.run, pins); err == nil {
		t.Fatal("a pwsh that could not run must not read as nothing missing")
	}
}

func TestMissing_NoModulesRunsNothing(t *testing.T) {
	f := &fakePwsh{}
	if got, err := Missing(f.run, nil); err != nil || got != nil || len(f.scripts) != 0 {
		t.Errorf("got %v, %v, %d calls", got, err, len(f.scripts))
	}
}

func TestInstall_UsesCIsFlagsForEachModule(t *testing.T) {
	f := &fakePwsh{}
	if err := Install(f.run, pins); err != nil {
		t.Fatal(err)
	}
	for _, m := range pins {
		want := "Install-Module -Name '" + m.Name + "' -RequiredVersion '" + m.Version + "' -Scope CurrentUser -Force -SkipPublisherCheck -ErrorAction Stop"
		if !strings.Contains(f.scripts[0], want) {
			t.Errorf("script lacks %q:\n%s", want, f.scripts[0])
		}
	}
}

func TestInstall_AFailureCarriesPwshsOutput(t *testing.T) {
	f := &fakePwsh{out: "No match was found for the specified search criteria", err: errors.New("exit status 1")}
	err := Install(f.run, pins)
	if err == nil || !strings.Contains(err.Error(), "No match was found") {
		t.Errorf("err = %v, want pwsh's own message", err)
	}
}

func TestLintScript_PinsTheAnalyzerAndCIsParameters(t *testing.T) {
	s := LintScript("1.25.0")
	for _, want := range []string{
		"Import-Module -Name 'PSScriptAnalyzer' -RequiredVersion '1.25.0'",
		"-Settings '" + SettingsFile + "'",
		"-Severity " + Severity,
		"$env:" + LintFilesEnv,
		"exit 1",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("lint script lacks %q:\n%s", want, s)
		}
	}
}

func TestLintable(t *testing.T) {
	got := Lintable([]string{"a.ps1", "dir/b.PSM1", "c.sh", "d.ps1.bak", "e.psd1"})
	if !reflect.DeepEqual(got, []string{"a.ps1", "dir/b.PSM1"}) {
		t.Errorf("got %v", got)
	}
}
