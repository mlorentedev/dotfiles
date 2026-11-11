package secrets

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GHWorkflows is the production WorkflowSource and fleet lister. It uses the REST
// API through `gh api` only, never GraphQL, whose budget other jobs share
// (#3228). Live-verified with the operator's `gh auth`, not in CI.
type GHWorkflows struct {
	Bin string // gh binary name/path; "" → "gh"
}

func (g GHWorkflows) gh(args ...string) ([]byte, error) {
	bin := g.Bin
	if bin == "" {
		bin = "gh"
	}
	cmd := exec.Command(bin, args...) //nolint:gosec // repo/path/ref come from the operator's flags and the repositories' own workflows
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// List returns the workflow file paths at the default branch. A repository
// with no .github/workflows directory has none, not an error.
func (g GHWorkflows) List(repo string) ([]string, error) {
	out, err := g.gh("api", "repos/"+repo+"/contents/.github/workflows", "-q", `.[] | select(.type == "file") | .path`)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 404") {
			return nil, nil
		}
		return nil, err
	}
	return lines(out), nil
}

// Read returns one file's raw bytes at ref (the default branch when empty).
func (g GHWorkflows) Read(repo, path, ref string) ([]byte, error) {
	endpoint := "repos/" + repo + "/contents/" + path
	if ref != "" {
		endpoint += "?ref=" + ref
	}
	return g.gh("api", "-H", "Accept: application/vnd.github.raw", endpoint)
}

// Fleet returns every non-archived, non-fork repository the authenticated user
// owns, as owner/name.
func (g GHWorkflows) Fleet() ([]string, error) {
	out, err := g.gh("api", "--paginate", "user/repos?affiliation=owner&per_page=100",
		"-q", `.[] | select(.archived | not) | select(.fork | not) | .full_name`)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func lines(b []byte) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
