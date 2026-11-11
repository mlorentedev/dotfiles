package secrets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	exprRe = regexp.MustCompile(`(?s)\$\{\{(.*?)\}\}`)
	// secrets.NAME, secrets['NAME'] or secrets["NAME"]; anything else after
	// `secrets` is a whole-context or computed read.
	secretsRe  = regexp.MustCompile(`\bsecrets\b`)
	literalRe  = regexp.MustCompile(`^(?:\s*\.\s*([A-Za-z_][A-Za-z0-9_]*)|\s*\[\s*'([A-Za-z_][A-Za-z0-9_]*)'\s*\]|\s*\[\s*"([A-Za-z_][A-Za-z0-9_]*)"\s*\])`)
	reusableRe = regexp.MustCompile(`^([^/\s]+/[^/\s]+)/(.+)@([^@\s]+)$`)
)

// workflowReads returns the secret names a workflow reads, sorted, unique and
// upper-cased (Actions secret names are case-insensitive), excluding
// GITHUB_TOKEN (Actions provides it), and the expressions whose name cannot be
// known. It reads the decoded YAML's scalar values, so an expression in a
// comment is never a read. An explicit mapping to a reusable workflow
// (`THEIR: ${{ secrets.MINE }}`) yields MINE, the name the caller must hold.
func workflowReads(wf []byte) (names, unknown []string, err error) {
	docs, err := decodeDocs(wf)
	if err != nil {
		return nil, nil, err
	}
	for _, d := range docs {
		eachScalar(d, func(v string) {
			for _, m := range exprRe.FindAllStringSubmatch(v, -1) {
				names, unknown = exprReads(m[1], names, unknown)
			}
		})
	}
	return sortedUnique(names), sortedUnique(unknown), nil
}

// exprReads appends the secret reads of one ${{ }} expression's body.
func exprReads(expr string, names, unknown []string) ([]string, []string) {
	for _, loc := range secretsRe.FindAllStringIndex(expr, -1) {
		if loc[0] > 0 && expr[loc[0]-1] == '.' {
			continue // inputs.secrets, github.secrets: another context's field
		}
		lit := literalRe.FindStringSubmatch(expr[loc[1]:])
		if lit == nil {
			unknown = append(unknown, strings.TrimSpace(expr))
			continue
		}
		if name := strings.ToUpper(lit[1] + lit[2] + lit[3]); name != "GITHUB_TOKEN" {
			names = append(names, name)
		}
	}
	return names, unknown
}

// decodeDocs decodes every YAML document in wf: a file may hold more than one,
// and yaml.Unmarshal would silently read only the first.
func decodeDocs(wf []byte) ([]*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(wf))
	var docs []*yaml.Node
	for {
		var n yaml.Node
		err := dec.Decode(&n)
		if errors.Is(err, io.EOF) {
			return docs, nil
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, &n)
	}
}

// eachScalar calls fn with every scalar value under n. Comments live in the
// nodes' comment fields, never in Value.
func eachScalar(n *yaml.Node, fn func(string)) {
	if n.Kind == yaml.ScalarNode {
		fn(n.Value)
	}
	for _, c := range n.Content {
		eachScalar(c, fn)
	}
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
	docs, err := decodeDocs(wf)
	if err != nil {
		return nil, err
	}
	var out []workflowRef
	for _, d := range docs {
		var doc struct {
			Jobs map[string]struct {
				Uses    string    `yaml:"uses"`
				Secrets yaml.Node `yaml:"secrets"`
			} `yaml:"jobs"`
		}
		if err := d.Decode(&doc); err != nil {
			return nil, err
		}
		for _, j := range doc.Jobs {
			m := reusableRe.FindStringSubmatch(j.Uses)
			if m == nil || j.Secrets.Kind != yaml.ScalarNode || j.Secrets.Value != "inherit" {
				continue
			}
			out = append(out, workflowRef{Repo: m[1], Path: m[2], Ref: m[3]})
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].String() < out[b].String() })
	return slices.Compact(out), nil
}

// calleeSecrets returns a reusable workflow's on.workflow_call.secrets, each
// mapped to its `required` flag (GitHub's default is false). It walks nodes, not
// a map, so the `on` key is found by its text whatever a YAML 1.1 reader would
// make of it.
func calleeSecrets(wf []byte) (map[string]bool, error) {
	docs, err := decodeDocs(wf)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, d := range docs {
		if len(d.Content) == 0 {
			continue
		}
		secrets := mappingValue(mappingValue(mappingValue(d.Content[0], "on"), "workflow_call"), "secrets")
		if secrets == nil || secrets.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(secrets.Content); i += 2 {
			req := mappingValue(secrets.Content[i+1], "required")
			out[strings.ToUpper(secrets.Content[i].Value)] = req != nil && req.Value == "true"
		}
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

// ciReads is what a repository's workflows read: names read unconditionally,
// names read only where a callee marks them optional (mapped to that callee),
// and reads whose name cannot be known.
type ciReads struct {
	required map[string]bool
	optional map[string]string
	unknown  []CIUnknown
}

// CheckRepo reads repo's workflows and diffs what they read with what
// Registry.SelectCI declares for it, the same selection `secrets sync ci`
// uploads, so the check and the sync cannot disagree.
func (c *CIChecker) CheckRepo(repo string) (CIReport, error) {
	reads, err := c.collectReads(repo)
	if err != nil {
		return CIReport{Repo: repo}, err
	}
	declared := map[string]bool{}
	for _, e := range c.reg.SelectCI(repo).Upload {
		declared[strings.ToUpper(e.Var)] = true
	}
	rep := diffReads(reads, declared)
	rep.Repo = repo
	return rep, nil
}

func (c *CIChecker) collectReads(repo string) (ciReads, error) {
	r := ciReads{required: map[string]bool{}, optional: map[string]string{}}
	files, err := c.src.List(repo)
	if err != nil {
		return r, fmt.Errorf("list workflows: %w", err)
	}
	sort.Strings(files)
	for _, f := range files {
		if ext := path.Ext(f); ext != ".yml" && ext != ".yaml" {
			continue
		}
		if err := c.addFile(repo, f, &r); err != nil {
			return r, err
		}
	}
	return r, nil
}

func (c *CIChecker) addFile(repo, f string, r *ciReads) error {
	wf, err := c.src.Read(repo, f, "")
	if err != nil {
		return fmt.Errorf("read %s: %w", f, err)
	}
	names, unknown, err := workflowReads(wf)
	if err != nil {
		return fmt.Errorf("parse %s: %w", f, err)
	}
	for _, n := range names {
		r.required[n] = true
	}
	for _, u := range unknown {
		r.unknown = append(r.unknown, CIUnknown{File: f, Expr: u})
	}
	calls, err := inheritedCalls(wf)
	if err != nil {
		return fmt.Errorf("parse %s: %w", f, err)
	}
	for _, call := range calls {
		sec, err := c.callee(call)
		if err != nil {
			return err
		}
		for n, req := range sec {
			r.addCallee(n, req, call.String())
		}
	}
	return nil
}

func (r *ciReads) addCallee(name string, required bool, callee string) {
	if required {
		r.required[name] = true
		return
	}
	if _, seen := r.optional[name]; !seen {
		r.optional[name] = callee
	}
}

// diffReads compares the reads with the declared names. A name read anywhere
// unconditionally is required even if a callee also marks it optional.
func diffReads(r ciReads, declared map[string]bool) CIReport {
	rep := CIReport{Unknown: r.unknown}
	for n := range r.required {
		if !declared[n] {
			rep.Missing = append(rep.Missing, CIMissing{Name: n})
		}
	}
	for n, in := range r.optional {
		if !r.required[n] && !declared[n] {
			rep.Missing = append(rep.Missing, CIMissing{Name: n, OptionalIn: in})
		}
	}
	for n := range declared {
		if _, opt := r.optional[n]; !r.required[n] && !opt {
			rep.Excess = append(rep.Excess, n)
		}
	}
	sort.Slice(rep.Missing, func(a, b int) bool { return rep.Missing[a].Name < rep.Missing[b].Name })
	sort.Strings(rep.Excess)
	return rep
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
