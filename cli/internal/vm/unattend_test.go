package vm

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type unattendDoc struct {
	Settings []struct {
		Pass       string `xml:"pass,attr"`
		Components []struct {
			Name string `xml:"name,attr"`
			Arch string `xml:"processorArchitecture,attr"`
			Raw  string `xml:",innerxml"`
		} `xml:"component"`
	} `xml:"settings"`
}

func readUnattend(t *testing.T) (unattendDoc, string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "unattend")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A file an earlier build left is not carried into the next one.
	if err := os.WriteFile(filepath.Join(dir, "stale.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteUnattend(dir); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("unattend dir holds %d files, want autounattend.xml and firstlogon.ps1", len(entries))
	}
	x, err := os.ReadFile(filepath.Join(dir, "autounattend.xml"))
	if err != nil {
		t.Fatal(err)
	}
	ps, err := os.ReadFile(filepath.Join(dir, "firstlogon.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	var doc unattendDoc
	if err := xml.Unmarshal(x, &doc); err != nil {
		t.Fatalf("autounattend.xml is not well-formed: %v", err)
	}
	return doc, string(x), string(ps)
}

// Every component targets arm64: Setup silently ignores a component whose
// architecture does not match the image, so a stray amd64 drops its settings.
func TestUnattend_EveryComponentIsArm64(t *testing.T) {
	doc, _, _ := readUnattend(t)
	passes := map[string]bool{}
	for _, s := range doc.Settings {
		passes[s.Pass] = true
		for _, c := range s.Components {
			if c.Arch != "arm64" {
				t.Errorf("%s/%s targets %q", s.Pass, c.Name, c.Arch)
			}
		}
	}
	for _, p := range []string{"windowsPE", "specialize", "oobeSystem"} {
		if !passes[p] {
			t.Errorf("no %s pass", p)
		}
	}
}

// HVF attaches no TPM, so Setup must be told to skip its TPM and Secure Boot
// checks before it runs them, in windowsPE.
func TestUnattend_BypassesTheTPMCheckInWindowsPE(t *testing.T) {
	doc, _, _ := readUnattend(t)
	var pe string
	for _, s := range doc.Settings {
		if s.Pass == "windowsPE" {
			for _, c := range s.Components {
				pe += c.Raw
			}
		}
	}
	for _, v := range []string{"BypassTPMCheck", "BypassSecureBootCheck"} {
		if !strings.Contains(pe, `HKLM\SYSTEM\Setup\LabConfig /v `+v+` /t REG_DWORD /d 1`) {
			t.Errorf("windowsPE does not set LabConfig %s", v)
		}
	}
}

// No credential reaches the image: every password is empty, and the only
// product key is Microsoft's published KMS client key for Pro, which picks the
// edition and activates nothing.
func TestUnattend_CarriesNoCredential(t *testing.T) {
	_, x, _ := readUnattend(t)
	for _, m := range regexp.MustCompile(`(?s)<Password>(.*?)</Password>`).FindAllStringSubmatch(x, -1) {
		if !strings.Contains(m[1], "<Value></Value>") {
			t.Errorf("a password is not empty: %s", m[1])
		}
	}
	keys := regexp.MustCompile(`[0-9A-Z]{5}(-[0-9A-Z]{5}){4}`).FindAllString(x, -1)
	for _, k := range keys {
		if k != "W269N-WFGWX-YVC9B-4J6C9-T83GX" {
			t.Errorf("product key %s is not the published KMS client key for Pro", k)
		}
	}
	if len(keys) != 1 {
		t.Errorf("found %d product keys, want the one edition key", len(keys))
	}
}

func TestUnattend_EnglishWithASpanishSecondKeyboard(t *testing.T) {
	_, x, _ := readUnattend(t)
	if strings.Count(x, "<InputLocale>0409:00000409;0c0a:0000040a</InputLocale>") != 2 {
		t.Error("windowsPE and oobeSystem must both set the US keyboard first and the Spanish one second")
	}
	for _, tag := range []string{"UILanguage", "SystemLocale", "UserLocale"} {
		for _, m := range regexp.MustCompile(`<`+tag+`>([^<]*)</`+tag+`>`).FindAllStringSubmatch(x, -1) {
			if m[1] != "en-US" {
				t.Errorf("%s is %q, want en-US", tag, m[1])
			}
		}
	}
}

// The first logon installs the ARM64 drivers the build's devices need and the
// guest agent, then makes a full shutdown: the build waits for QEMU to exit,
// and a hybrid shutdown would leave a hibernated kernel in clean-base.
func TestFirstLogon_InstallsTheAgentThenShutsDown(t *testing.T) {
	_, x, ps := readUnattend(t)
	if !strings.Contains(x, `firstlogon.ps1`) {
		t.Error("FirstLogonCommands does not run firstlogon.ps1")
	}
	for _, want := range []string{`'NetKVM', 'vioserial'`, `\w11\ARM64\*.inf`, `guest-agent\qemu-ga-x86_64.msi`, `Set-Service -Name 'QEMU-GA' -StartupType Automatic`} {
		if !strings.Contains(ps, want) {
			t.Errorf("firstlogon.ps1 lacks %q", want)
		}
	}
	lines := strings.Split(strings.TrimSpace(ps), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last != "& shutdown.exe /s /f /t 0" {
		t.Errorf("firstlogon.ps1 ends with %q, want a full shutdown", last)
	}
	for _, l := range lines {
		if l = strings.TrimSpace(l); !strings.HasPrefix(l, "#") && strings.Contains(l, "/hybrid") {
			t.Errorf("firstlogon.ps1 makes a hybrid shutdown: %s", l)
		}
	}
}
