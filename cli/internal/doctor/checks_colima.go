package doctor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// colimaCPU and colimaMemoryGiB are the Colima VM the repo declares (owner,
// #2013 P5b): kubelab's compose stack does not fit the 2 CPU / 2 GiB default.
//
// They live here, not in a deploy entry, because colima.yaml is colima's own
// file: colima treats one without `runtime` as empty, so a merged fragment is
// ignored on first start and then overwritten with defaults. Doctor lets colima
// write the file and corrects these two keys in it afterwards.
const (
	colimaCPU       = 4
	colimaMemoryGiB = 8
)

// checkColimaSize compares the declared size with colima.yaml and with the
// running VM. A size change applies only at the VM's next start, so --fix
// edits the file and then restarts the Homebrew service, saying so: the
// restart stops every container the engine runs. A WARN, never a FAIL, like
// the engine check it follows.
func checkColimaSize(sys *System, rep *Report, fix bool) {
	if sys.GOOS != "darwin" || !sys.has("colima") {
		return
	}
	rep.Section("Colima VM size")
	want := fmt.Sprintf("%d CPU, %d GiB", colimaCPU, colimaMemoryGiB)

	file := filepath.Join(colimaHome(sys), "default", "colima.yaml")
	raw, err := os.ReadFile(file) //nolint:gosec // colima's own config under $HOME
	if errors.Is(err, os.ErrNotExist) {
		rep.Skip("no Colima instance yet; the engine check starts one (dotf doctor --fix)")
		return
	}
	if err != nil {
		rep.Warn("cannot read " + file + ": " + err.Error())
		return
	}
	cpu, mem, err := colimaFileSize(raw)
	if err != nil {
		rep.Warn(file + " does not parse (" + err.Error() + "); colima rewrites it on `colima start --edit`")
		return
	}
	fileOK := cpu == colimaCPU && mem == colimaMemoryGiB
	running, runCPU, runMem := colimaRunningSize(sys)
	vmOK := !running || (runCPU == colimaCPU && runMem == colimaMemoryGiB<<30)

	if fileOK && vmOK {
		rep.Pass(want)
		return
	}
	drift := fmt.Sprintf("colima.yaml has %d CPU, %g GiB", cpu, mem)
	if fileOK {
		drift = fmt.Sprintf("the VM runs with %d CPU, %d GiB", runCPU, runMem>>30)
	}
	if !fix {
		rep.Warn(drift + "; the repo declares " + want + " (run: dotf doctor --fix, which restarts Colima and stops its containers)")
		return
	}
	if !fileOK {
		if err := setColimaSize(file, raw); err != nil {
			rep.Warn("cannot set the size in " + file + ": " + err.Error())
			return
		}
		if !running {
			rep.Fix("set " + want + " in " + file + "; it applies when Colima next starts")
			return
		}
	}
	restartColima(sys, rep, want)
}

// colimaHome is where colima keeps its profiles: $COLIMA_HOME, else ~/.colima.
func colimaHome(sys *System) string {
	if h := sys.Getenv("COLIMA_HOME"); h != "" {
		return h
	}
	return filepath.Join(sys.home(), ".colima")
}

// colimaFileSize reads cpu and memory (GiB, possibly fractional) from
// colima.yaml.
func colimaFileSize(raw []byte) (cpu int, memGiB float64, err error) {
	var c struct {
		CPU    int     `yaml:"cpu"`
		Memory float64 `yaml:"memory"`
	}
	err = yaml.Unmarshal(raw, &c)
	return c.CPU, c.Memory, err
}

// colimaRunningSize asks colima for the default profile's running VM. `colima
// list --json` prints one object per profile, memory in bytes. A stopped VM,
// or no answer, reports not running: the file is then the whole story.
func colimaRunningSize(sys *System) (running bool, cpu int, memBytes int64) {
	out, _, err := sys.CommandOutputBounded(dockerEngineTimeout, "colima", "list", "--json")
	if err != nil {
		return false, 0, 0
	}
	dec := json.NewDecoder(strings.NewReader(out))
	for {
		var p struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			CPUs   int    `json:"cpus"`
			Memory int64  `json:"memory"`
		}
		if dec.Decode(&p) != nil {
			return false, 0, 0
		}
		if p.Name == "default" {
			return strings.EqualFold(p.Status, "Running"), p.CPUs, p.Memory
		}
	}
}

// setColimaSize rewrites the top-level cpu and memory lines of colima.yaml in
// place, so the rest of colima's file, comments and blank lines included,
// stays byte for byte; a YAML round trip drops the blank lines. The result is
// parsed back before it is written, and the write is a rename, so colima never
// reads a half-written or wrong file.
func setColimaSize(file string, raw []byte) error {
	out := setYAMLLine(raw, "cpu", colimaCPU)
	out = setYAMLLine(out, "memory", colimaMemoryGiB)
	if cpu, mem, err := colimaFileSize(out); err != nil || cpu != colimaCPU || mem != colimaMemoryGiB {
		return fmt.Errorf("the edited file does not read back as %d CPU, %d GiB", colimaCPU, colimaMemoryGiB)
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	tmp := file + ".dotf-tmp"
	if err := os.WriteFile(tmp, out, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// setYAMLLine sets a top-level `key: value` line, keeping an inline comment,
// or appends the line when the key is absent.
func setYAMLLine(raw []byte, key string, v int) []byte {
	re := regexp.MustCompile(`(?m)^` + key + `:[ \t]*[^\n#]*?([ \t]*#[^\n]*)?$`)
	line := key + ": " + strconv.Itoa(v)
	if re.Match(raw) {
		return re.ReplaceAll(raw, []byte(line+"$1"))
	}
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		raw = append(raw, '\n')
	}
	return append(raw, line+"\n"...)
}

// restartColima applies a size through the Homebrew service, the only owner of
// Colima's lifecycle doctor uses: its plist runs `colima start -f` and
// relaunches it on a clean exit, so a direct `colima stop` or `start` races it.
// A VM running outside the service is left alone with the command to run.
func restartColima(sys *System, rep *Report, want string) {
	if !colimaServiceRunning(sys) {
		rep.Warn("Colima runs outside brew services; apply " + want + " with: colima stop && brew services start colima")
		return
	}
	if _, errOut, err := sys.CommandOutputBounded(2*time.Minute, "brew", "services", "restart", "colima"); err != nil {
		rep.Warn("brew services restart colima failed: " + strings.TrimSpace(errOut+" "+err.Error()))
		return
	}
	// The repair is the VM running at the size, not the engine answering: the
	// stopping VM can still answer `docker info` on the first probe.
	for i := 0; i < engineProbeAttempts; i++ {
		if running, cpu, mem := colimaRunningSize(sys); running && cpu == colimaCPU && mem == colimaMemoryGiB<<30 {
			rep.Fix("restarted Colima at " + want + " (brew services restart colima); its containers stopped")
			return
		}
		time.Sleep(engineProbeInterval)
	}
	rep.Warn("restarted Colima (its containers stopped), but the VM has not reported " + want + " yet; re-run dotf doctor in a minute")
}

// colimaServiceRunning reports whether Homebrew's launchd service runs Colima.
func colimaServiceRunning(sys *System) bool {
	out, _, err := sys.CommandOutputBounded(dockerEngineTimeout, "brew", "services", "info", "colima", "--json")
	if err != nil {
		return false
	}
	var services []struct {
		Running bool `json:"running"`
	}
	return json.Unmarshal([]byte(out), &services) == nil && len(services) == 1 && services[0].Running
}
