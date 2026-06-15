package initrepo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StackResult reports what StackInit did.
type StackResult struct {
	Stack   string
	Actions []string // steps performed
	Skipped []string // steps skipped (tool absent, already present, or non-fatal failure)
}

// StackInit performs a lightweight, offline project init for the chosen stack:
// the project manifest plus (for go) a Makefile. It deliberately does NOT install
// opinionated dependencies over the network — a scaffold must be fast and
// offline-safe, and the project declares its own deps. Missing tools degrade to a
// skip note, never an error (mirroring init-project.sh's `|| true`).
func StackInit(root, stack string) (StackResult, error) {
	res := StackResult{Stack: stack}

	switch stack {
	case "go":
		res.goInit(root)
	case "python":
		switch {
		case hasTool("uv"):
			res.run(root, "uv", "uv", "init")
		case hasTool("poetry"):
			res.run(root, "poetry init", "poetry", "init", "-n")
		default:
			res.Skipped = append(res.Skipped, "python init (neither uv nor poetry found)")
		}
	case "node", "ts":
		if hasTool("npm") {
			res.run(root, "npm init", "npm", "init", "-y")
		} else {
			res.Skipped = append(res.Skipped, "npm init (npm not found)")
		}
	case "none", "":
		// nothing to do
	default:
		res.Skipped = append(res.Skipped, "unknown stack: "+stack)
	}
	return res, nil
}

// goInit runs `go mod init` (skip-if-present / skip-if-absent / tolerate failure)
// and writes the embedded Makefile (skip-if-present). The Makefile is useful with
// or without go on PATH.
func (res *StackResult) goInit(root string) {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		res.Skipped = append(res.Skipped, "go mod init (go.mod present)")
	case !hasTool("go"):
		res.Skipped = append(res.Skipped, "go mod init (go not found)")
	default:
		cmd := exec.Command("go", "mod", "init", filepath.Base(root))
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			res.Skipped = append(res.Skipped, "go mod init failed: "+strings.TrimSpace(string(out)))
		} else {
			res.Actions = append(res.Actions, "go mod init")
		}
	}

	dest := filepath.Join(root, "Makefile")
	if fileExists(dest) {
		res.Skipped = append(res.Skipped, "Makefile (present)")
		return
	}
	if raw, err := ReadTemplate("Makefile"); err == nil {
		if err := os.WriteFile(dest, raw, 0o644); err == nil {
			res.Actions = append(res.Actions, "Makefile")
		}
	}
}

// run executes name+args in dir, recording label under Actions on success or a
// skip note on failure (non-fatal — the scaffold continues).
func (res *StackResult) run(dir, label, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		res.Skipped = append(res.Skipped, fmt.Sprintf("%s failed: %s", label, strings.TrimSpace(string(out))))
		return
	}
	res.Actions = append(res.Actions, label)
}

func hasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
