package tools

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const pluginConf = `# mise: cli
AGE_VERSION=1.3.1
# mise: cli
ZSH_AUTOSUGGESTIONS_VERSION=0.7.1
`

func TestParseMisePins_AcceptsAFileToolAtAPinnedDigest(t *testing.T) {
	p, err := ParseMisePins([]byte(pluginConf))
	if err != nil {
		t.Fatal(err)
	}
	if !hasTool(p.Tools, "zsh-autosuggestions") {
		t.Fatalf("tools = %v", p.Tools)
	}
}

func TestParseMisePins_RejectsAFileToolWhoseDigestIsNotPinned(t *testing.T) {
	_, err := ParseMisePins([]byte("# mise: cli\nZSH_AUTOSUGGESTIONS_VERSION=9.9.9\n"))
	if err == nil || !strings.Contains(err.Error(), "no pinned sha256") || !strings.Contains(err.Error(), "0.7.1") {
		t.Fatalf("err = %v, want the unpinned version and the known ones named", err)
	}
}

func TestParseMisePins_RejectsAFileToolThatTracksLatest(t *testing.T) {
	_, err := ParseMisePins([]byte("# mise: latest\nZSH_AUTOSUGGESTIONS_VERSION=0.7.1\n"))
	if err == nil || !strings.Contains(err.Error(), "cannot track latest") {
		t.Fatalf("err = %v", err)
	}
}

func TestRenderMiseConfig_PinsAFileToolByUrlAndDigest(t *testing.T) {
	got := string(RenderMiseConfig([]MiseTool{{"age", "1.3.1"}, {"zsh-autosuggestions", "0.7.1"}}, ""))
	want := `"http:zsh-autosuggestions" = { version = "0.7.1", url = "https://github.com/zsh-users/zsh-autosuggestions/archive/refs/tags/v0.7.1.tar.gz", checksum = "sha256:0df7affff21cd87ed298e6a3970ed08a1dd66a6efa676454ee5b091ad503badf" }`
	if runtime.GOOS == "windows" {
		if strings.Contains(got, "zsh-autosuggestions") {
			t.Errorf("windows rendered a zsh plugin:\n%s", got)
		}
		return
	}
	if !strings.Contains(got, want+"\n") {
		t.Errorf("config lacks the http entry:\n%s", got)
	}
	if !strings.Contains(got, `age = "1.3.1"`) {
		t.Errorf("a binary pin is gone:\n%s", got)
	}
}

func TestBinaryPins_DropsTheFileTools(t *testing.T) {
	got := BinaryPins([]MiseTool{{"age", "1.3.1"}, {"zsh-autosuggestions", "0.7.1"}})
	if want := []MiseTool{{"age", "1.3.1"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("BinaryPins = %v, want %v", got, want)
	}
}

// fakeFileMise stands in for mise's http backend: `mise install` unpacks the
// plugin into <root>/<version>/, `mise where` answers only for an installed
// version, and `mise ls --prunable` lists what the test sets.
type fakeFileMise struct {
	root      string
	version   string // what `mise install` unpacks
	installs  int
	broken    bool                // install runs but leaves no file behind
	prunable  map[string][]string // as mise lists them: keyed http:<name>
	unpacked  bool
	prunes    [][]string
	failWhere bool
	forced    int
}

func (f *fakeFileMise) run(name string, args ...string) ([]byte, error) {
	switch {
	case name == "mise" && len(args) == 2 && args[0] == "where" && args[1] == "http:zsh-autosuggestions":
		if f.failWhere || !f.unpacked {
			return nil, errors.New("not installed")
		}
		return []byte(filepath.Join(f.root, f.version) + "\n"), nil
	case name == "mise" && len(args) >= 1 && args[0] == "install":
		force := len(args) == 3 && args[1] == "--force" && args[2] == "http:zsh-autosuggestions@"+f.version
		if len(args) != 1 && !force {
			break
		}
		dir := filepath.Join(f.root, f.version)
		if force {
			f.forced++
		} else {
			f.installs++
			// mise counts an existing install directory as the install.
			if _, err := os.Stat(dir); err == nil {
				return nil, nil
			}
		}
		if f.broken {
			return nil, nil
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		f.unpacked = true
		return nil, os.WriteFile(filepath.Join(dir, "zsh-autosuggestions.zsh"), []byte("# plugin\n"), 0o644)
	case name == "mise" && len(args) == 3 && args[0] == "ls" && args[1] == "--prunable":
		out := "{"
		first := true
		for tool, versions := range f.prunable {
			for _, v := range versions {
				if !first {
					out += ","
				}
				first = false
				out += `"` + tool + `":[{"version":"` + v + `"}]`
			}
		}
		return []byte(out + "}"), nil
	case name == "mise" && len(args) >= 3 && args[0] == "prune":
		f.prunes = append(f.prunes, args[3:])
		for _, tool := range args[3:] {
			delete(f.prunable, tool)
		}
		return nil, nil
	}
	return nil, errors.New("unexpected call: " + name + " " + strings.Join(args, " "))
}

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("zsh plugins are not installed on Windows")
	}
}

func TestSync_InstallsAFileToolVerifiesTheFileAndIsIdempotent(t *testing.T) {
	skipOnWindows(t)
	pins := []MiseTool{{"zsh-autosuggestions", "0.7.1"}}
	f := &fakeFileMise{root: t.TempDir(), version: "0.7.1"}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}

	plan, err := s.Plan(pins)
	if err != nil || !reflect.DeepEqual(plan.Missing, []string{"zsh-autosuggestions"}) {
		t.Fatalf("fresh plan: %+v, %v", plan, err)
	}
	if _, err := s.Apply(pins); err != nil {
		t.Fatal(err)
	}
	again, err := s.Apply(pins)
	if err != nil || again.Pending() || f.installs != 1 {
		t.Errorf("second run: pending=%v err=%v after %d installs, want changed=0", again.Pending(), err, f.installs)
	}
}

func TestSync_AFileToolWhoseInstallLeftNoFileFailsNamingIt(t *testing.T) {
	skipOnWindows(t)
	pins := []MiseTool{{"zsh-autosuggestions", "0.7.1"}}
	f := &fakeFileMise{root: t.TempDir(), version: "0.7.1", broken: true, unpacked: true}
	if err := os.MkdirAll(filepath.Join(f.root, "0.7.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run}
	_, err := s.Apply(pins)
	if err == nil || !strings.Contains(err.Error(), "zsh-autosuggestions") {
		t.Fatalf("err = %v, want the tool named: an empty install directory is not an install", err)
	}
}

func TestSync_AnOlderInstalledVersionOfAFileToolIsMissing(t *testing.T) {
	skipOnWindows(t)
	f := &fakeFileMise{root: t.TempDir(), version: "0.7.0", unpacked: true}
	if err := os.MkdirAll(filepath.Join(f.root, "0.7.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, "0.7.0", "zsh-autosuggestions.zsh"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run}
	if s.runsAtPin(MiseTool{"zsh-autosuggestions", "0.7.1"}) {
		t.Error("0.7.0 on disk counted as the 0.7.1 pin")
	}
}

// mise lists the plugin as http:zsh-autosuggestions. Matching it only by the
// pin's short name would prune it on every run and reinstall it on the next:
// never changed=0.
func TestSync_DoesNotPruneAPinnedFileToolAndPrunesAnUnpinnedOne(t *testing.T) {
	skipOnWindows(t)
	pins := []MiseTool{{"zsh-autosuggestions", "0.7.1"}}
	f := &fakeFileMise{root: t.TempDir(), version: "0.7.1", prunable: map[string][]string{"http:zsh-autosuggestions": {"0.7.1"}}}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run, Prune: true}
	orphans, err := s.Orphans(pins)
	if err != nil || len(orphans) != 0 {
		t.Fatalf("pinned plugin reported as orphan: %v, %v", orphans, err)
	}
	orphans, err = s.Orphans(nil)
	if want := []string{"http:zsh-autosuggestions@0.7.1"}; err != nil || !reflect.DeepEqual(orphans, want) {
		t.Errorf("unpinned plugin: orphans = %v, %v; want %v", orphans, err, want)
	}
}

// `mise install` treats a directory without the file as installed, so only
// --force repairs it; without that the sync failed on every run.
func TestSync_RepairsAFileToolWhoseFileWasDeleted(t *testing.T) {
	skipOnWindows(t)
	pins := []MiseTool{{"zsh-autosuggestions", "0.7.1"}}
	f := &fakeFileMise{root: t.TempDir(), version: "0.7.1", unpacked: true}
	if err := os.MkdirAll(filepath.Join(f.root, "0.7.1"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := MiseSync{ConfigDir: t.TempDir(), Run: f.run}
	if _, err := s.Apply(pins); err != nil {
		t.Fatal(err)
	}
	if f.forced != 1 {
		t.Errorf("forced installs = %d, want 1", f.forced)
	}
	again, err := s.Apply(pins)
	if err != nil || again.Pending() || f.forced != 1 {
		t.Errorf("second run: pending=%v err=%v forced=%d, want changed=0", again.Pending(), err, f.forced)
	}
}
