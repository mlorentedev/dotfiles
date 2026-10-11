package env

import (
	"encoding/xml"
	"errors"
	"strings"
	"testing"
)

// fakeLaunchd is the user's launchd session as launchctl reports it: getenv
// on an unset name exits 0 and prints nothing (measured on macOS 27).
type fakeLaunchd map[string]string

func (f fakeLaunchd) run(args ...string) (string, error) {
	switch args[0] {
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
