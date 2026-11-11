package secrets

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// WorkflowSource reads a repository's workflow files. List returns the paths
// under .github/workflows at the default branch; Read returns one file at ref,
// or at the default branch when ref is empty. The seam keeps ci-check testable
// with no gh and no network.
type WorkflowSource interface {
	List(repo string) ([]string, error)
	Read(repo, path, ref string) ([]byte, error)
}

// CIMissing is a secret a workflow reads and the registry does not declare for
// the repository. OptionalIn names the reusable workflow that declares it
// `required: false` when that is the only place it is read: the run does not
// fail without it, it degrades, which is what the check exists to show.
type CIMissing struct {
	Name       string
	OptionalIn string
}

// CIUnknown is a read whose secret name cannot be known statically.
type CIUnknown struct {
	File string
	Expr string
}

// CIReport compares one repository's workflow reads with its declared secrets.
type CIReport struct {
	Repo    string
	Missing []CIMissing
	Excess  []string
	Unknown []CIUnknown
}

// Failing reports whether the declaration disagrees with the reads. Unknown
// reads are listed, never passed and never failed: nothing can be said of them.
func (r CIReport) Failing() bool { return len(r.Missing) > 0 || len(r.Excess) > 0 }

var (
	exprRe = regexp.MustCompile(`\$\{\{(.*?)\}\}`)
	// secrets.NAME, secrets['NAME'] or secrets["NAME"]; anything else after
	// `secrets` is a whole-context or computed read.
	secretsRe  = regexp.MustCompile(`\bsecrets\b`)
	literalRe  = regexp.MustCompile(`^(?:\s*\.\s*([A-Za-z_][A-Za-z0-9_]*)|\s*\[\s*'([A-Za-z_][A-Za-z0-9_]*)'\s*\]|\s*\[\s*"([A-Za-z_][A-Za-z0-9_]*)"\s*\])`)
	reusableRe = regexp.MustCompile(`^([^/\s]+/[^/\s]+)/(.+)@([^@\s]+)$`)
)

// workflowReads returns the secret names a workflow reads, sorted and unique,
// excluding GITHUB_TOKEN (Actions provides it), and the expressions whose name
// cannot be known. An explicit mapping to a reusable workflow
// (`THEIR: ${{ secrets.MINE }}`) yields MINE, the name the caller must hold.
func workflowReads(wf []byte) (names, unknown []string) {
	for _, m := range exprRe.FindAllSubmatch(wf, -1) {
		expr := string(m[1])
		for _, loc := range secretsRe.FindAllStringIndex(expr, -1) {
			if loc[0] > 0 && expr[loc[0]-1] == '.' {
				continue // inputs.secrets, github.secrets: another context's field
			}
			lit := literalRe.FindStringSubmatch(expr[loc[1]:])
			if lit == nil {
				unknown = append(unknown, strings.TrimSpace(expr))
				continue
			}
			name := lit[1] + lit[2] + lit[3]
			if name != "GITHUB_TOKEN" {
				names = append(names, name)
			}
		}
	}
	return sortedUnique(names), sortedUnique(unknown)
}

// workflowRef addresses a reusable workflow in another repository.
type workflowRef struct {
	Repo, Path, Ref string
}

func (w workflowRef) String() string { return w.Repo + "/" + w.Path + "@" + w.Ref }

// inheritedCalls returns the remote reusable workflows a workflow calls with
// `secrets: inherit`, sorted. A local call (./...) is skipped: its file is one of
// the repository's own workflows, whose reads are scanned directly.
func inheritedCalls(wf []byte) ([]workflowRef, error) {
	var doc struct {
		Jobs map[string]struct {
			Uses    string    `yaml:"uses"`
			Secrets yaml.Node `yaml:"secrets"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(wf, &doc); err != nil {
		return nil, err
	}
	var out []workflowRef
	for _, j := range doc.Jobs {
		if j.Secrets.Kind != yaml.ScalarNode || j.Secrets.Value != "inherit" {
			continue
		}
		m := reusableRe.FindStringSubmatch(j.Uses)
		if m == nil {
			continue
		}
		out = append(out, workflowRef{Repo: m[1], Path: m[2], Ref: m[3]})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].String() < out[b].String() })
	return slices.Compact(out), nil
}

// calleeSecrets returns a reusable workflow's on.workflow_call.secrets, each
// mapped to its `required` flag (GitHub's default is false). It walks nodes, not
// a map, so the `on` key is found by its text whatever a YAML 1.1 reader would
// make of it.
func calleeSecrets(wf []byte) (map[string]bool, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(wf, &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 {
		return map[string]bool{}, nil
	}
	secrets := mappingValue(mappingValue(mappingValue(root.Content[0], "on"), "workflow_call"), "secrets")
	out := map[string]bool{}
	if secrets == nil || secrets.Kind != yaml.MappingNode {
		return out, nil
	}
	for i := 0; i+1 < len(secrets.Content); i += 2 {
		req := mappingValue(secrets.Content[i+1], "required")
		out[secrets.Content[i].Value] = req != nil && req.Value == "true"
	}
	return out, nil
}

func mappingValue(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

// CIChecker compares workflow reads with the registry, caching each callee.
type CIChecker struct {
	src     WorkflowSource
	reg     *Registry
	callees map[workflowRef]map[string]bool
}

// NewCIChecker builds a checker over src and reg.
func NewCIChecker(src WorkflowSource, reg *Registry) *CIChecker {
	return &CIChecker{src: src, reg: reg, callees: map[workflowRef]map[string]bool{}}
}

// CheckRepo reads repo's workflows and diffs what they read with what
// Registry.SelectCI declares for it, the same selection `secrets sync ci`
// uploads, so the check and the sync cannot disagree.
func (c *CIChecker) CheckRepo(repo string) (CIReport, error) {
	rep := CIReport{Repo: repo}
	files, err := c.src.List(repo)
	if err != nil {
		return rep, fmt.Errorf("%s: list workflows: %w", repo, err)
	}
	sort.Strings(files)
	required := map[string]bool{}   // read unconditionally
	optional := map[string]string{} // read only where a callee marks it optional
	for _, f := range files {
		if ext := path.Ext(f); ext != ".yml" && ext != ".yaml" {
			continue
		}
		wf, err := c.src.Read(repo, f, "")
		if err != nil {
			return rep, fmt.Errorf("%s: read %s: %w", repo, f, err)
		}
		names, unknown := workflowReads(wf)
		for _, n := range names {
			required[n] = true
		}
		for _, u := range unknown {
			rep.Unknown = append(rep.Unknown, CIUnknown{File: f, Expr: u})
		}
		calls, err := inheritedCalls(wf)
		if err != nil {
			return rep, fmt.Errorf("%s: parse %s: %w", repo, f, err)
		}
		for _, call := range calls {
			sec, err := c.callee(call)
			if err != nil {
				return rep, err
			}
			for n, req := range sec {
				if req {
					required[n] = true
				} else if _, seen := optional[n]; !seen {
					optional[n] = call.String()
				}
			}
		}
	}
	declared := map[string]bool{}
	for _, e := range c.reg.SelectCI(repo).Upload {
		declared[e.Var] = true
	}
	for n := range required {
		if !declared[n] {
			rep.Missing = append(rep.Missing, CIMissing{Name: n})
		}
	}
	for n, in := range optional {
		if !required[n] && !declared[n] {
			rep.Missing = append(rep.Missing, CIMissing{Name: n, OptionalIn: in})
		}
	}
	for n := range declared {
		if _, opt := optional[n]; !required[n] && !opt {
			rep.Excess = append(rep.Excess, n)
		}
	}
	sort.Slice(rep.Missing, func(a, b int) bool { return rep.Missing[a].Name < rep.Missing[b].Name })
	sort.Strings(rep.Excess)
	return rep, nil
}

func (c *CIChecker) callee(w workflowRef) (map[string]bool, error) {
	if sec, ok := c.callees[w]; ok {
		return sec, nil
	}
	wf, err := c.src.Read(w.Repo, w.Path, w.Ref)
	if err != nil {
		return nil, fmt.Errorf("read reusable workflow %s: %w", w, err)
	}
	sec, err := calleeSecrets(wf)
	if err != nil {
		return nil, fmt.Errorf("parse reusable workflow %s: %w", w, err)
	}
	c.callees[w] = sec
	return sec, nil
}

func sortedUnique(s []string) []string {
	sort.Strings(s)
	return slices.Compact(s)
}
