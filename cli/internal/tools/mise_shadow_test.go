package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// miseLsEntry is one entry of `mise ls --json <tool>` as mise 2026.11.6 prints
// it: one per installed version, and the one mise runs carries active and the
// config that requested it in source.path.
func miseLsEntry(version, requested, source string, active bool) string {
	entry := map[string]any{"version": version, "installed": true, "active": active}
	if requested != "" {
		entry["requested_version"] = requested
	}
	if source != "" {
		// Marshalled, so a Windows path's backslashes are escaped as mise does.
		entry["source"] = map[string]string{"type": "mise.toml", "path": source}
	}
	out, err := json.Marshal(entry)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// shadowMise wraps fakeMise with the one command it does not know,
// `mise ls --json <tool>`, answered from ls (tool -> raw output). A tool with
// no row makes the command fail, as mise does for a tool it cannot resolve.
func shadowMise(f *fakeMise, ls map[string]string) Runner {
	return func(name string, args ...string) ([]byte, error) {
		if name == "mise" && len(args) == 3 && args[0] == "ls" && args[1] == "--json" {
			out, ok := ls[args[2]]
			if !ok {
				return []byte("mise ERROR not found"), errors.New("exit status 1")
			}
			return []byte(out), nil
		}
		return f.run(name, args...)
	}
}

// When a config that outranks dotfiles' conf.d entry pins another version, mise
// install "succeeds" and the tool still runs the other version. The failure
// names the file, the version it pins and the fix (#3524); where mise does not
// say who resolved the tool, it claims no cause.
func TestSync_NamesTheConfigThatShadowsAPin(t *testing.T) {
	const fix = "remove the pin there, or let that file's owner stop pinning a tool versions.conf owns"
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".config", "mise")
	own := filepath.Join(cfgDir, "conf.d", "dotfiles.toml")
	foreign := filepath.Join(cfgDir, "config.toml")
	pins := []MiseTool{{"node", "24.22.1"}, {"python", "3.13.1"}}
	// mise runs 24.19.0 and 3.12.13 whatever dotfiles pins: install adds the
	// pinned versions beside them and the foreign config keeps winning.
	pending := []MiseTool{{"node", "24.19.0"}, {"python", "3.12.13"}}
	nodeLs := "[" + miseLsEntry("24.22.1", "", "", false) + "," + miseLsEntry("24.19.0", "24.19.0", foreign, true) + "]"
	pythonLs := "[" + miseLsEntry("3.12.13", "3.12.13", foreign, true) + "]"

	for _, tc := range []struct {
		name    string
		ls      map[string]string
		want    []string // every one must appear in the error
		notWant []string // none may appear
	}{
		{
			name: "shadowed by a config outside conf.d",
			ls:   map[string]string{"node": nodeLs, "python": pythonLs},
			want: []string{
				"not running at their pin: node, python",
				foreign + " pins node 24.19.0, python 3.12.13",
				"outranks " + own,
				fix,
			},
		},
		{
			name: "only one of the two tools is shadowed",
			ls:   map[string]string{"node": nodeLs, "python": "[" + miseLsEntry("3.12.13", "3.13.1", own, true) + "]"},
			want: []string{"not running at their pin: node, python", foreign + " pins node 24.19.0", fix},
			// python's active entry is dotfiles' own file: no claim about it.
			notWant: []string{"python 3.12.13"},
		},
		{
			name:    "dotfiles' own file is the active one: a genuine mismatch",
			ls:      map[string]string{"node": "[" + miseLsEntry("24.19.0", "24.22.1", own, true) + "]", "python": "[" + miseLsEntry("3.12.13", "3.13.1", own, true) + "]"},
			want:    []string{"not running at their pin: node, python"},
			notWant: []string{"outranks", "pins", fix},
		},
		{
			name:    "mise output unparseable",
			ls:      map[string]string{"node": "not json at all", "python": `{"unexpected": "shape"}`},
			want:    []string{"not running at their pin: node, python"},
			notWant: []string{"outranks", "pins", fix},
		},
		{
			name:    "no entry is active",
			ls:      map[string]string{"node": "[" + miseLsEntry("24.19.0", "", "", false) + "]", "python": "[]"},
			want:    []string{"not running at their pin: node, python"},
			notWant: []string{"outranks", "pins", fix},
		},
		{
			name:    "mise ls fails",
			ls:      map[string]string{},
			want:    []string{"not running at their pin: node, python"},
			notWant: []string{"outranks", "pins", fix},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeMise{installed: map[string]string{}, pending: pending}
			s := MiseSync{ConfigDir: cfgDir, Run: shadowMise(f, tc.ls)}
			_, err := s.Apply(pins)
			if err == nil {
				t.Fatal("want an error, got none")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q lacks %q", err, w)
				}
			}
			for _, w := range tc.notWant {
				if strings.Contains(err.Error(), w) {
					t.Errorf("error %q must not contain %q", err, w)
				}
			}
		})
	}
}

// The doctor reads the same shadowing from a plan, before any converge runs:
// ShadowNote is the one function both use, and a plan with nothing missing
// asks mise nothing.
func TestShadowNote_AsksOnlyAboutTheToolsThatFail(t *testing.T) {
	cfgDir := t.TempDir()
	foreign := filepath.Join(t.TempDir(), "config.toml")
	var asked []string
	run := func(name string, args ...string) ([]byte, error) {
		asked = append(asked, strings.Join(args, " "))
		return []byte("[" + miseLsEntry("24.19.0", "24.19.0", foreign, true) + "]"), nil
	}
	s := MiseSync{ConfigDir: cfgDir, Run: run}

	pins := []MiseTool{{"node", "24.22.1"}}
	if cause, fix := s.ShadowNote(pins, nil); cause != "" || fix != "" || len(asked) != 0 {
		t.Fatalf("nothing missing: want no note and no mise call, got %q %q after %v", cause, fix, asked)
	}
	cause, fix := s.ShadowNote(pins, []string{"node"})
	if !strings.Contains(cause, foreign+" pins node 24.19.0") || fix == "" {
		t.Errorf("ShadowNote = %q, %q", cause, fix)
	}
	if want := []string{"ls --json node"}; len(asked) != 1 || asked[0] != want[0] {
		t.Errorf("asked mise %v, want %v", asked, want)
	}
}

// A symlinked config dir reaches dotfiles' own file under another spelling
// (macOS /var is /private/var): mise's answer is still "not shadowed".
func TestShadowNote_ComparesThePathThroughSymlinks(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("cannot create a symlink here: " + err.Error())
	}
	if err := os.MkdirAll(filepath.Join(real, "conf.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(real, "conf.d", "dotfiles.toml")
	if err := os.WriteFile(own, []byte("[tools]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(string, ...string) ([]byte, error) {
		return []byte("[" + miseLsEntry("24.19.0", "24.22.1", own, true) + "]"), nil
	}
	s := MiseSync{ConfigDir: link, Run: run}
	if cause, _ := s.ShadowNote([]MiseTool{{"node", "24.22.1"}}, []string{"node"}); cause != "" {
		t.Errorf("dotfiles' own file under another spelling is not a shadow, got %q", cause)
	}
}

// A foreign config is the cause only when it makes the tool run something
// below the pin. One that resolves the pinned version, or a newer one, did not
// make the tool fail, and telling the operator to remove it would send them
// after the wrong file (#3524: "pins a different version").
func TestShadowNote_NamesAFileOnlyWhenItResolvesBelowThePin(t *testing.T) {
	foreign := filepath.Join(t.TempDir(), "config.toml")
	pins := []MiseTool{{"node", "24.22.1"}}
	for _, tc := range []struct {
		name    string
		version string // what the foreign file makes mise run
		named   bool
	}{
		{"below the pin", "24.19.0", true},
		{"the pinned version, the failure is something else", "24.22.1", false},
		{"above the pin", "25.0.0", false},
		{"no resolved version", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := func(string, ...string) ([]byte, error) {
				return []byte("[" + miseLsEntry(tc.version, "", foreign, true) + "]"), nil
			}
			s := MiseSync{ConfigDir: t.TempDir(), Run: run}
			cause, fix := s.ShadowNote(pins, []string{"node"})
			if tc.named != (cause != "" && fix != "") {
				t.Errorf("ShadowNote = %q, %q; want a named cause: %v", cause, fix, tc.named)
			}
		})
	}
}

// miseLsCurrent is `mise ls --current --json` as mise 2026.11.6 prints it: one
// object, a tool's name to its entries.
func miseLsCurrent(entries map[string]string) string {
	var parts []string
	for tool, entry := range entries {
		parts = append(parts, `"`+tool+`":[`+entry+`]`)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// A pin another config holds is a drift whichever way the version differs: a
// shadow that is newer than the pin passes the sync's post-condition and
// nothing else reports it, so the Go 1.27.4 and golangci-lint 2.15.0 that no
// reviewed commit chose ran unseen (#3532). One call asks mise about every
// pin, and only a foreign file is named.
func TestSourceDrift_NamesEveryPinAnotherConfigHoldsWhicheverWayTheVersionDiffers(t *testing.T) {
	cfgDir := t.TempDir()
	foreign := filepath.Join(t.TempDir(), "config.toml")
	s := MiseSync{ConfigDir: cfgDir}
	own := s.ConfigPath()
	var asked []string
	s.Stdout = func(name string, args ...string) ([]byte, error) {
		asked = append(asked, name+" "+strings.Join(args, " "))
		return []byte(miseLsCurrent(map[string]string{
			"golangci-lint": miseLsEntry("2.15.0", "2.15.0", foreign, true),   // newer than the pin
			"node":          miseLsEntry("24.19.0", "24.19.0", foreign, true), // older than the pin
			"bats":          miseLsEntry("1.13.0", "1.13.0", foreign, true),   // the same version, still foreign
			"age":           miseLsEntry("1.2.1", "1.2.1", own, true),         // the sync's own file
			"jq":            miseLsEntry("1.7.1", "1.7.1", foreign, false),    // inactive: not what runs
			"stray":         miseLsEntry("9.9.9", "9.9.9", foreign, true),     // no pin asks for it
		})), nil
	}
	pins := []MiseTool{{"age", "1.2.1"}, {"bats", "1.13.0"}, {"golangci-lint", "2.12.2"}, {"jq", "1.7.1"}, {"node", "24.22.1"}}

	drifts, err := s.SourceDrift(pins)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"mise ls --current --json"}; len(asked) != 1 || asked[0] != want[0] {
		t.Errorf("asked mise %v, want one call %v", asked, want)
	}
	got := map[string]string{}
	for _, d := range drifts {
		if d.Path != foreign {
			t.Errorf("%s: source %q, want %q", d.Tool, d.Path, foreign)
		}
		got[d.Tool] = d.Version + " vs pin " + d.Pin
	}
	want := map[string]string{
		"bats":          "1.13.0 vs pin 1.13.0",
		"golangci-lint": "2.15.0 vs pin 2.12.2",
		"node":          "24.19.0 vs pin 24.22.1",
	}
	if len(got) != len(want) {
		t.Fatalf("drifts = %v, want %v", got, want)
	}
	for tool, w := range want {
		if got[tool] != w {
			t.Errorf("%s = %q, want %q", tool, got[tool], w)
		}
	}
}

// mise failing or answering in a shape this does not read is "could not check",
// never "no drift": the caller must be able to say so.
func TestSourceDrift_AFailedOrUnreadableAnswerIsAnErrorNotAClean(t *testing.T) {
	for name, run := range map[string]Runner{
		"mise fails":   func(string, ...string) ([]byte, error) { return []byte("boom"), errors.New("exit status 1") },
		"not json":     func(string, ...string) ([]byte, error) { return []byte("not json"), nil },
		"an array":     func(string, ...string) ([]byte, error) { return []byte("[]"), nil },
		"empty output": func(string, ...string) ([]byte, error) { return nil, nil },
	} {
		t.Run(name, func(t *testing.T) {
			s := MiseSync{ConfigDir: t.TempDir(), Stdout: run}
			if drifts, err := s.SourceDrift([]MiseTool{{"age", "1.2.1"}}); err == nil {
				t.Errorf("want an error, got drifts %v", drifts)
			}
		})
	}
}

// The sync's own file is recognised through a symlinked prefix (macOS /var is
// /private/var), and an entry with no source path claims nothing.
func TestSourceDrift_OwnFileThroughASymlinkAndAnEntryWithoutASourceAreNotDrift(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlinks unavailable: " + err.Error())
	}
	s := MiseSync{ConfigDir: link}
	if err := os.MkdirAll(filepath.Join(real, "conf.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	physical := filepath.Join(real, "conf.d", "dotfiles.toml")
	if err := os.WriteFile(physical, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s.Stdout = func(string, ...string) ([]byte, error) {
		return []byte(miseLsCurrent(map[string]string{
			"age": miseLsEntry("1.2.1", "1.2.1", physical, true),
			"jq":  miseLsEntry("1.7.1", "1.7.1", "", true),
		})), nil
	}
	drifts, err := s.SourceDrift([]MiseTool{{"age", "1.2.1"}, {"jq", "1.7.1"}})
	if err != nil || len(drifts) != 0 {
		t.Errorf("got %v, %v; want no drift", drifts, err)
	}
}
