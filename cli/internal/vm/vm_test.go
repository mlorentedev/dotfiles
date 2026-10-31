package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/mlorentedev/dotfiles/cli/internal/converge"
)

// The guest must see nothing of the host: no shared directory (the host's ~
// carries credentials and the working tree), no forwarded agent, no host keys.
func TestLinuxTemplate_SharesNothingWithTheHost(t *testing.T) {
	var tmpl struct {
		MinimumLimaVersion string           `yaml:"minimumLimaVersion"`
		Base               []string         `yaml:"base"`
		Plain              bool             `yaml:"plain"`
		Mounts             []map[string]any `yaml:"mounts"`
		SSH                map[string]bool  `yaml:"ssh"`
	}
	if err := yaml.Unmarshal(LinuxTemplate, &tmpl); err != nil {
		t.Fatal(err)
	}
	if !tmpl.Plain {
		t.Error("plain mode must be on: it disables mounts, port forwarding and the guest agent")
	}
	if !strings.Contains(string(LinuxTemplate), "\nmounts: []\n") || len(tmpl.Mounts) != 0 {
		t.Error("mounts must be stated as an empty list")
	}
	if len(tmpl.Base) != 1 || tmpl.Base[0] != "template:_images/ubuntu-24.04" {
		t.Errorf("base must be the Ubuntu 24.04 image alone, not a template that adds mounts: %v", tmpl.Base)
	}
	for _, k := range []string{"loadDotSSHPubKeys", "forwardAgent", "forwardX11"} {
		if v, ok := tmpl.SSH[k]; !ok || v {
			t.Errorf("ssh.%s must be stated false", k)
		}
	}
	if tmpl.MinimumLimaVersion == "" {
		t.Error("minimumLimaVersion must be set: an older Lima reads plain mode differently")
	}
}

func TestIsolated(t *testing.T) {
	clean := "/ ext4\n/boot/efi vfat\n/run tmpfs"
	cases := []struct {
		name, pid1, mounts, want string
	}{
		{"a clean systemd guest", "systemd", clean, ""},
		{"PID 1 is not systemd", "bash", clean, "not systemd"},
		{"the host's home over virtiofs", "systemd", clean + "\n/Users/manu virtiofs", "/Users/manu (virtiofs)"},
		{"a reverse-sshfs mount", "systemd", "/tmp/lima fuse.sshfs", "fuse.sshfs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Isolated(c.pid1, c.mounts)
			if c.want == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestIdempotent_NamesTheStepsThatChanged(t *testing.T) {
	r := converge.RunReport{Result: "ok", Changed: 3, Entries: []converge.RunEntry{{Name: "tools", Status: "change", Changes: 3}, {Name: "configs", Status: "ok"}}}
	if err := Converged(r); err != nil {
		t.Fatalf("a first run with changes and no failure converged: %v", err)
	}
	err := Idempotent(r)
	if err == nil || !strings.Contains(err.Error(), "tools (3)") || strings.Contains(err.Error(), "configs") {
		t.Fatalf("want the changing step named, got %v", err)
	}
	r.Changed, r.Entries[0].Changes, r.Entries[0].Status = 0, 0, "ok"
	if err := Idempotent(r); err != nil {
		t.Fatalf("a no-op second run: %v", err)
	}
	r.Entries[1].Status = "failed"
	if err := Idempotent(r); err == nil || !strings.Contains(err.Error(), "configs") {
		t.Fatalf("a failed step must fail the run, naming it: %v", err)
	}
}

// fakeLima answers like a clean guest whose converges report the given
// changes, and records each call with what it read on stdin.
type fakeLima struct {
	changed   []int // the report's change count after each converge
	runs      int
	calls     []string
	stdins    []string
	failNext  string // a call starting with this fails
	silentRun int    // this converge (1-based) writes no report
	report    bool   // a report exists in the guest
}

func (f *fakeLima) run(_ context.Context, stdin io.Reader, stdout, _ io.Writer, args ...string) error {
	call := strings.Join(args, " ")
	f.calls = append(f.calls, call)
	in := ""
	if stdin != nil {
		b, _ := io.ReadAll(stdin)
		in = string(b)
	}
	f.stdins = append(f.stdins, in)
	if f.failNext != "" && strings.HasPrefix(call, f.failNext) {
		return errors.New("boom")
	}
	switch {
	case strings.HasSuffix(call, "printenv HOME"):
		_, _ = io.WriteString(stdout, "/home/manu.linux\n")
	case strings.HasSuffix(call, "comm="):
		_, _ = io.WriteString(stdout, "systemd\n")
	case strings.HasSuffix(call, "TARGET,FSTYPE"):
		_, _ = io.WriteString(stdout, "/ ext4\n")
	case strings.Contains(call, " bash -c "):
		f.runs++
		if f.runs != f.silentRun {
			f.report = true // a silent run leaves the previous report in place
		}
	case strings.Contains(call, " rm -f "):
		f.report = false
	case strings.Contains(call, " cat "):
		if !f.report {
			return errors.New("cat: no such file")
		}
		_, _ = fmt.Fprintf(stdout, `{"result":"ok","changed":%d,"entries":[{"name":"tools","status":"ok","changes":%d}]}`, f.changed[f.runs-1], f.changed[f.runs-1])
	}
	return nil
}

func TestVerify_ConvergesTwiceAndDeletesTheGuest(t *testing.T) {
	f := &fakeLima{changed: []int{12, 0}}
	var out strings.Builder
	if err := Verify(context.Background(), f.run, Options{Ref: "feat/x", Source: "/tmp/src.tar.gz", Name: "dotf-linux-1", Token: "s3cr3t", Out: &out}); err != nil {
		t.Fatalf("Verify: %v\n%s", err, out.String())
	}
	if !strings.HasPrefix(f.calls[0], "start --tty=false --name dotf-linux-1 ") {
		t.Errorf("first call: %q", f.calls[0])
	}
	if last := f.calls[len(f.calls)-1]; last != "delete --force dotf-linux-1" {
		t.Errorf("the guest must be deleted last, got %q", last)
	}
	converges := 0
	for i, c := range f.calls {
		if strings.Contains(c, "s3cr3t") {
			t.Errorf("the token reached an argument list: %q", c)
		}
		if strings.Contains(c, " bash -c ") {
			converges++
			if f.stdins[i] != "s3cr3t\n" {
				t.Errorf("a converge read %q on stdin, want the token", f.stdins[i])
			}
			if !strings.Contains(c, "--workdir /home/manu.linux/Projects/dotfiles ") {
				t.Errorf("a converge ran outside the clone: %q", c)
			}
		} else if f.stdins[i] != "" {
			t.Errorf("%q read %q on stdin: only a converge may", c, f.stdins[i])
		}
	}
	if converges != 2 {
		t.Errorf("want 2 converges, got %d", converges)
	}
	// The ref reaches the guest as the host's git archive, not as a clone:
	// any committed ref works, pushed or not, and the guest needs no network
	// for its checkout.
	copied := false
	for _, c := range f.calls {
		if strings.Contains(c, "git ") {
			t.Errorf("the guest ran git: %q", c)
		}
		if c == "copy /tmp/src.tar.gz dotf-linux-1:/home/manu.linux/dotf-src.tar.gz" {
			copied = true
		}
	}
	if !copied {
		t.Errorf("the archive was not copied into the guest's home: %q", f.calls)
	}
}

func TestVerify_ASecondRunWithChangesFailsAndStillDeletes(t *testing.T) {
	f := &fakeLima{changed: []int{12, 1}}
	err := Verify(context.Background(), f.run, Options{Ref: "main", Source: "/tmp/src.tar.gz", Name: "n", Out: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "second converge changed 1") {
		t.Fatalf("want the second run's change reported, got %v", err)
	}
	if last := f.calls[len(f.calls)-1]; last != "delete --force n" {
		t.Errorf("a failed run must still delete the guest, got %q", last)
	}
}

func TestVerify_KeepLeavesTheGuest(t *testing.T) {
	f := &fakeLima{changed: []int{1, 0}, failNext: "copy "}
	if err := Verify(context.Background(), f.run, Options{Ref: "main", Source: "/tmp/src.tar.gz", Name: "n", Keep: true, Out: io.Discard}); err == nil {
		t.Fatal("a failed copy must fail the run")
	}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "delete") {
			t.Errorf("--keep deleted the guest: %q", c)
		}
	}
}

func TestVerify_RefusesARunWithNoSource(t *testing.T) {
	f := &fakeLima{}
	if err := Verify(context.Background(), f.run, Options{Ref: "main", Name: "n", Out: io.Discard}); err == nil || len(f.calls) != 0 {
		t.Errorf("no source: err %v after %d call(s)", err, len(f.calls))
	}
}

// A start that fails partway (an image download, provisioning) can leave an
// instance and its disk behind: the delete must run anyway.
func TestVerify_AFailedStartStillDeletes(t *testing.T) {
	f := &fakeLima{failNext: "start "}
	if err := Verify(context.Background(), f.run, Options{Ref: "main", Source: "/tmp/src.tar.gz", Name: "n", Out: io.Discard}); err == nil {
		t.Fatal("a failed start must fail the run")
	}
	if last := f.calls[len(f.calls)-1]; last != "delete --force n" {
		t.Errorf("a failed start must still delete the guest, got %q", last)
	}
}

// Each converge is judged on the report it wrote: the previous run's report
// is removed first, so a run that writes none cannot pass on stale data.
func TestVerify_AConvergeThatWritesNoReportFails(t *testing.T) {
	f := &fakeLima{changed: []int{0, 0}, silentRun: 2}
	err := Verify(context.Background(), f.run, Options{Ref: "main", Source: "/tmp/src.tar.gz", Name: "n", Out: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "left no report") {
		t.Fatalf("a second converge that wrote no report must fail, got %v", err)
	}
}
