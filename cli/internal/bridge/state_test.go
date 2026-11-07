package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDescriptor(profile string) LaunchDescriptor {
	return LaunchDescriptor{
		Profile:    profile,
		Socks5:     "127.0.0.1:1081",
		HealthAddr: "127.0.0.1:9081",
	}
}

func TestStoreRoundTripsLaunchAndProcessState(t *testing.T) {
	store := NewStore(t.TempDir())
	launch := testDescriptor("office")
	process := ProcessRecord{
		PID:         4242,
		Executable:  filepath.Join(t.TempDir(), "ts-bridge"),
		StartedAt:   time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
		Fingerprint: "sha256:launch",
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
		PID:         42,
		Executable:  filepath.Join(t.TempDir(), "ts-bridge"),
		StartedAt:   time.Now().UTC(),
		Fingerprint: "sha256:launch",
	}
	cases := []struct {
		name   string
		change func(*ProcessRecord)
	}{
		{"pid", func(p *ProcessRecord) { p.PID = 0 }},
		{"executable", func(p *ProcessRecord) { p.Executable = "" }},
		{"start", func(p *ProcessRecord) { p.StartedAt = time.Time{} }},
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
