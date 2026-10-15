package env

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeLaunchd is the user's launchd session as launchctl reports it: getenv
// on an unset name exits 0 and prints nothing (measured on macOS 27).
type fakeLaunchd map[string]string

func (f fakeLaunchd) run(args ...string) (string, error) {
	switch args[0] {
	case "managername":
		return "Aqua\n", nil
	case "getenv":
		if v, ok := f[args[1]]; ok {
			return v + "\n", nil
		}
		return "", nil
	case "setenv":
		f[args[1]] = args[2]
	case "unsetenv":
		delete(f, args[1])
	default:
		return "", errors.New("unexpected launchctl " + strings.Join(args, " "))
	}
	return "", nil
}

func TestLaunchdUserEnv_RoundTripsAndKeepsSpaces(t *testing.T) {
	s := LaunchdUserEnv{Run: fakeLaunchd{}.run}
	if _, ok, err := s.Get("VAULT_PATH"); err != nil || ok {
		t.Fatalf("an unset name must read as absent, got ok=%v err=%v", ok, err)
	}
	if err := s.Set("VAULT_PATH", "/Users/me/My Vault"); err != nil {
		t.Fatal(err)
	}
	if v, ok, _ := s.Get("VAULT_PATH"); !ok || v != "/Users/me/My Vault" {
		t.Fatalf("Get = %q, %v", v, ok)
	}
	if err := s.Delete("VAULT_PATH"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("VAULT_PATH"); err != nil {
		t.Fatalf("deleting an absent name must succeed (the sweep relies on it): %v", err)
	}
}

// launchd cannot tell an empty value from an unset name, so storing one would
// make every later run see drift and write it again.
func TestLaunchdUserEnv_RefusesAnEmptyValue(t *testing.T) {
	s := LaunchdUserEnv{Run: fakeLaunchd{}.run}
	if err := s.Set("VAULT_PATH", ""); !errors.Is(err, ErrEmptyLaunchdValue) {
		t.Fatalf("want ErrEmptyLaunchdValue, got %v", err)
	}
}

// Persist over the launchd store is idempotent: a second run changes nothing.
func TestLaunchdUserEnv_PersistTwiceChangesNothing(t *testing.T) {
	s := LaunchdUserEnv{Run: fakeLaunchd{}.run}
	vars := []ResolvedVar{{Name: "VAULT_PATH", Value: "/Users/me/vault"}, {Name: "DOTFILES_DIR", Value: "/Users/me/.dotfiles"}}
	if _, err := Persist(vars, s); err != nil {
		t.Fatal(err)
	}
	res, err := Persist(vars, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.Changed || r.Removed {
			t.Errorf("second run changed %s", r.Name)
		}
	}
}

func TestLaunchAgentPlist_RunsTheInstalledDotfAtLogin(t *testing.T) {
	plist := LaunchAgentPlist("/Users/a&b")
	var doc struct {
		Dict struct {
			Keys    []string `xml:"key"`
			Strings []string `xml:"string"`
			Args    []string `xml:"array>string"`
		} `xml:"dict"`
	}
	if err := xml.Unmarshal(plist, &doc); err != nil {
		t.Fatalf("the plist must be well-formed XML: %v", err)
	}
	want := []string{"/Users/a&b/.local/bin/dotf", "env", "persist"}
	if strings.Join(doc.Dict.Args, "|") != strings.Join(want, "|") {
		t.Errorf("ProgramArguments = %q, want %q", doc.Dict.Args, want)
	}
	if !strings.Contains(string(plist), "<key>RunAtLoad</key>\n\t<true/>") {
		t.Error("the agent must run at load, which is what re-applies the scope at login")
	}
	if doc.Dict.Strings[0] != LaunchAgentLabel {
		t.Errorf("Label = %q", doc.Dict.Strings[0])
	}
}

// backgroundLaunchd is launchctl as a process under tmux or herdr sees it
// (#2507): its own domain is "Background", so getenv and setenv reach that
// domain, and the login session is visible only through `print gui/<uid>`.
type backgroundLaunchd struct {
	own   fakeLaunchd
	print string // the raw `launchctl print gui/501` output
}

func (b backgroundLaunchd) run(args ...string) (string, error) {
	switch {
	case args[0] == "managername":
		return "Background\n", nil
	case args[0] == "print" && len(args) == 2 && args[1] == "gui/501":
		return b.print, nil
	}
	return b.own.run(args...)
}

func measuredPrint(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "launchctl-print-gui.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The defect of #2507: under herdr every variable read as missing, because
// getenv answered for the Background domain. Outside the login session the
// store reads the login session's environment block instead, and never the
// caller's own domain, even where that holds a value.
func TestLaunchdUserEnv_OutsideTheLoginSessionReadsTheLoginSession(t *testing.T) {
	b := backgroundLaunchd{own: fakeLaunchd{"VAULT_PATH": "/stale"}, print: measuredPrint(t)}
	s := LaunchdUserEnv{Run: b.run, UID: 501}
	for name, want := range map[string]string{
		"VAULT_PATH":       "/Users/me/My Vault",
		"DOTFILES_DIR":     "/Users/me/.dotfiles",
		"DOTF_MANAGED_ENV": "DOTFILES_DIR;VAULT_PATH",
	} {
		if v, ok, err := s.Get(name); err != nil || !ok || v != want {
			t.Errorf("Get(%s) = %q, %v, %v; want %q", name, v, ok, err, want)
		}
	}
	if _, ok, err := s.Get("AGE_KEY_PATH"); err != nil || ok {
		t.Errorf("a name the login session lacks must read as absent, got ok=%v err=%v", ok, err)
	}
}

// A session with nothing set prints no environment block; that is an empty
// scope, not an error.
func TestLaunchdUserEnv_NoEnvironmentBlockReadsAsEmpty(t *testing.T) {
	b := backgroundLaunchd{own: fakeLaunchd{}, print: "gui/501 = {\n\ttype = login\n\tsession = Aqua\n}\n"}
	s := LaunchdUserEnv{Run: b.run, UID: 501}
	if _, ok, err := s.Get("VAULT_PATH"); err != nil || ok {
		t.Fatalf("want absent, got ok=%v err=%v", ok, err)
	}
}

// Output that is not a domain listing at all means launchctl changed shape:
// that must surface, not read as "nothing set" and report drift.
func TestLaunchdUserEnv_UnrecognisedPrintIsAnError(t *testing.T) {
	b := backgroundLaunchd{own: fakeLaunchd{}, print: "Bad request.\n"}
	s := LaunchdUserEnv{Run: b.run, UID: 501}
	if _, _, err := s.Get("VAULT_PATH"); err == nil {
		t.Fatal("want an error for output with no gui/501 domain header")
	}
}

// setenv from Background lands in Background, where no Dock app reads it, so
// the store refuses rather than report a write the login session never saw.
func TestLaunchdUserEnv_OutsideTheLoginSessionRefusesToWrite(t *testing.T) {
	b := backgroundLaunchd{own: fakeLaunchd{}, print: measuredPrint(t)}
	s := LaunchdUserEnv{Run: b.run, UID: 501}
	if err := s.Set("VAULT_PATH", "/x"); !errors.Is(err, ErrOutsideLoginSession) {
		t.Errorf("Set: want ErrOutsideLoginSession, got %v", err)
	}
	if err := s.Delete("VAULT_PATH"); !errors.Is(err, ErrOutsideLoginSession) {
		t.Errorf("Delete: want ErrOutsideLoginSession, got %v", err)
	}
	if len(b.own) != 0 {
		t.Errorf("nothing may reach the caller's own domain, got %v", b.own)
	}
}
