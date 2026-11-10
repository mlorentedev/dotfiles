package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testDescriptor(profile string) LaunchDescriptor {
	return LaunchDescriptor{
		Profile:    profile,
		Socks5:     "127.0.0.1:1081",
		HealthAddr: "127.0.0.1:9081",
	}
}

func TestLaunchDescriptorFingerprintIsStableAndDiscriminating(t *testing.T) {
	base := testDescriptor("office")
	if base.Fingerprint() != testDescriptor("office").Fingerprint() {
		t.Fatalf("identical descriptors produced %q and %q", base.Fingerprint(), testDescriptor("office").Fingerprint())
	}
	if !strings.HasPrefix(base.Fingerprint(), "sha256:") {
		t.Fatalf("fingerprint %q is not a sha256 digest", base.Fingerprint())
	}
	for _, tc := range []struct {
		name   string
		change func(LaunchDescriptor) LaunchDescriptor
	}{
		{"profile", func(d LaunchDescriptor) LaunchDescriptor { d.Profile = "branch"; return d }},
		{"config", func(d LaunchDescriptor) LaunchDescriptor { d.Config = filepath.Join("/tmp", "office.yml"); return d }},
		{"socks5", func(d LaunchDescriptor) LaunchDescriptor { d.Socks5 = "127.0.0.1:1082"; return d }},
		{"health", func(d LaunchDescriptor) LaunchDescriptor { d.HealthAddr = "127.0.0.1:9082"; return d }},
	} {
		if got := tc.change(base).Fingerprint(); got == base.Fingerprint() {
			t.Fatalf("%s change did not alter the fingerprint", tc.name)
		}
	}
}

func TestStoreRoundTripsLaunchAndProcessState(t *testing.T) {
	store := NewStore(t.TempDir())
	launch := testDescriptor("office")
	process := ProcessRecord{
		PID:           4242,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "platform:12345",
		Fingerprint:   "sha256:launch",
	}

	if err := store.SaveLaunch(launch); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveProcess(launch.Profile, process); err != nil {
		t.Fatal(err)
	}
	gotLaunch, err := store.LoadLaunch(launch.Profile)
	if err != nil || gotLaunch != launch {
		t.Fatalf("LoadLaunch() = %+v, %v; want %+v", gotLaunch, err, launch)
	}
	gotProcess, err := store.LoadProcess(launch.Profile)
	if err != nil || gotProcess != process {
		t.Fatalf("LoadProcess() = %+v, %v; want %+v", gotProcess, err, process)
	}
	if got, err := store.ResolveProfile(""); err != nil || got != "office" {
		t.Fatalf("ResolveProfile() = %q, %v; want office", got, err)
	}
}

func TestStoreResolveProfileRefusesAmbiguity(t *testing.T) {
	store := NewStore(t.TempDir())
	for _, profile := range []string{"office", "lab"} {
		if err := store.SaveLaunch(testDescriptor(profile)); err != nil {
			t.Fatal(err)
		}
	}

	_, err := store.ResolveProfile("")
	if err == nil || !strings.Contains(err.Error(), "lab, office") {
		t.Fatalf("ResolveProfile() error = %v; want sorted choices", err)
	}
}

func TestStoreReplacesExistingLaunchState(t *testing.T) {
	store := NewStore(t.TempDir())
	first := testDescriptor("office")
	if err := store.SaveLaunch(first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Config = filepath.Join(t.TempDir(), "office.yml")
	if err := store.SaveLaunch(second); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadLaunch("office")
	if err != nil || got != second {
		t.Fatalf("LoadLaunch() = %+v, %v; want %+v", got, err, second)
	}
}

func TestStoreRefusesRelativeStateRoot(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := NewStore("").SaveLaunch(testDescriptor("office")); err == nil ||
		!strings.Contains(err.Error(), "absolute") {
		t.Fatalf("SaveLaunch() error = %v; want absolute state root refusal", err)
	}
	if _, err := os.Stat(filepath.Join("bridge", "office", "launch.json")); !os.IsNotExist(err) {
		t.Fatalf("relative state escaped into the cwd: %v", err)
	}
}

func TestLaunchDescriptorRejectsUnsafeValues(t *testing.T) {
	cases := []struct {
		name   string
		change func(*LaunchDescriptor)
		want   string
	}{
		{"empty profile", func(d *LaunchDescriptor) { d.Profile = "" }, "profile"},
		{"traversal profile", func(d *LaunchDescriptor) { d.Profile = "../office" }, "profile"},
		{"non-loopback socks", func(d *LaunchDescriptor) { d.Socks5 = "10.0.0.2:1081" }, "loopback"},
		{"non-loopback health", func(d *LaunchDescriptor) { d.HealthAddr = "0.0.0.0:9081" }, "loopback"},
		{"missing socks", func(d *LaunchDescriptor) { d.Socks5 = "" }, "SOCKS"},
		{"missing health", func(d *LaunchDescriptor) { d.HealthAddr = "" }, "health"},
		{"relative config", func(d *LaunchDescriptor) { d.Config = "relative.yml" }, "absolute"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := testDescriptor("office")
			tc.change(&d)
			if err := d.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v; want %q", err, tc.want)
			}
		})
	}
}

func TestProcessRecordRejectsInvalidIdentity(t *testing.T) {
	valid := ProcessRecord{
		PID:           42,
		Executable:    filepath.Join(t.TempDir(), "ts-bridge"),
		StartIdentity: "platform:12345",
		Fingerprint:   "sha256:launch",
	}
	cases := []struct {
		name   string
		change func(*ProcessRecord)
	}{
		{"pid", func(p *ProcessRecord) { p.PID = 0 }},
		{"executable", func(p *ProcessRecord) { p.Executable = "" }},
		{"start", func(p *ProcessRecord) { p.StartIdentity = "" }},
		{"fingerprint", func(p *ProcessRecord) { p.Fingerprint = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.change(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("Validate() accepted incomplete process identity")
			}
		})
	}
}
