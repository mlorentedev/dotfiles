//go:build darwin

package worktree

import (
	"bufio"
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// darwinProc is one process as the darwin probes see it. Cwd is empty when
// lsof could not read it: another user's process, or one that exited between
// the two listings.
type darwinProc struct {
	PPID int
	Comm string
	Cwd  string
}

// readProcTable lists every process with its parent and, where readable, its
// working directory. Darwin has no /proc, and libproc needs cgo, so it asks ps
// and lsof, each once: ~0.1s for the whole machine, cheap enough for done.
//
// lsof's exit status is not the verdict. It exits 1 when any process could not
// be listed, which an ordinary user's scan always hits, and still prints the
// rest. What fails the read is an empty listing from either tool: with no
// answer, a caller must not conclude that nobody is inside.
func readProcTable() (map[int]darwinProc, error) {
	ps, err := exec.Command("ps", "-axo", "pid=,ppid=,comm=").Output()
	if err != nil {
		return nil, err
	}
	procs := parsePs(ps)
	if len(procs) == 0 {
		return nil, errors.New("ps listed no process")
	}
	lsof, _ := exec.Command("lsof", "-n", "-P", "-w", "-d", "cwd", "-F", "pn").Output()
	cwds := parseLsofCwds(lsof)
	if len(cwds) == 0 {
		return nil, errors.New("lsof listed no working directory")
	}
	for pid, cwd := range cwds {
		if p, ok := procs[pid]; ok {
			p.Cwd = cwd
			procs[pid] = p
		}
	}
	return procs, nil
}

// parsePs reads `ps -axo pid=,ppid=,comm=`: two numbers, then the command,
// which may hold spaces. Comm keeps the base name, as /proc's comm does.
func parsePs(out []byte) map[int]darwinProc {
	procs := map[int]darwinProc{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		procs[pid] = darwinProc{PPID: ppid, Comm: filepath.Base(strings.Join(f[2:], " "))}
	}
	return procs
}

// parseLsofCwds reads `lsof -F pn -d cwd`: a "p<pid>" line opens each
// process, and its "n<path>" line is the working directory, physical (macOS
// reports /private/var, never the /var symlink).
func parseLsofCwds(out []byte) map[int]string {
	cwds := map[int]string{}
	pid := 0
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			n, err := strconv.Atoi(line[1:])
			if err != nil {
				n = 0
			}
			pid = n
		case 'n':
			if pid > 0 && strings.HasPrefix(line[1:], "/") {
				cwds[pid] = line[1:]
			}
		}
	}
	return cwds
}
