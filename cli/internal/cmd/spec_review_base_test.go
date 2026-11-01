package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// HARNESS-112. A review whose diff is empty does not fail — it silently becomes
// a reading of the spec folder, and reports findings with nothing executed
// behind them. That is worse than no review, because it arrives wearing a
// verdict. The launcher must refuse instead.

func withReviewBase(t *testing.T, base spec.ReviewBase, head string) {
	t.Helper()
	prevBase, prevHead := resolveReviewBase, headSHAOf
	resolveReviewBase = func(string, string) spec.ReviewBase { return base }
	headSHAOf = func(string) string { return head }
	t.Cleanup(func() { resolveReviewBase, headSHAOf = prevBase, prevHead })
}

func seedReviewFixture(t *testing.T) string {
	t.Helper()
	root := makeRepo(t)
	writeFixture(t, root, "harness/reviewer-pool.json", `{"pool":[
		{"id":"nan/deepseek-v4-flash","runner":"pi","provider":"nan","model":"deepseek-v4-flash","role":"primary"}
	]}`)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")
	prev := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("no tmux") }
	t.Cleanup(func() { lookPath = prev })
	return root
}

func TestSpecReviewRefusesWhenNoBaseCanBeResolved(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, spec.ReviewBase{Refusal: spec.BaseNotCommitted}, "headheadhead")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("the launch was allowed with no review base; the reviewer would browse the "+
			"spec folder and call that a review:\n%s", stdout+stderr)
	}
	if !strings.Contains(err.Error(), "cannot resolve a review base") {
		t.Errorf("the refusal does not say what is wrong: %v", err)
	}
}

func TestSpecReviewRefusesWhenTheBaseIsTheHead(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, spec.ReviewBase{SHA: "samesamesame"}, "samesamesame")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("a review with base == HEAD was allowed; its diff is empty by construction:\n%s",
			stdout+stderr)
	}
	if !strings.Contains(err.Error(), "nothing to review") {
		t.Errorf("the refusal does not name the empty diff: %v", err)
	}
}

// The positive direction, so the two refusals above cannot pass vacuously by the
// launcher refusing everything.
func TestSpecReviewLaunchesAndStatesTheBaseWhenOneResolves(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, spec.ReviewBase{SHA: "basebasebase"}, "headheadhead")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err != nil {
		t.Fatalf("a resolvable base was refused: %v\n%s", err, stdout+stderr)
	}
	if out := stdout + stderr; !strings.Contains(out, "basebasebase") {
		t.Errorf("the launch does not hand the reviewer its base, so it will guess `main` — "+
			"which is empty on a post-merge review:\n%s", out)
	}
}

// CLI-132 (#2566). A spec born in its repository's root commit has no base
// commit. --initial reviews it from the empty tree, and is refused for every
// other resolver answer, so the flag can never widen an ordinary, uncommitted or
// untraceable spec into a whole-tree review.

func withEmptyTree(t *testing.T, tree string) {
	t.Helper()
	prev := emptyTreeOf
	emptyTreeOf = func(string) (string, error) { return tree, nil }
	t.Cleanup(func() { emptyTreeOf = prev })
}

// specEvidence is the bytes of the files a refused or dry-run launch must not
// touch: the previous verdict and the previous request.
func specEvidence(t *testing.T, root string) map[string]string {
	t.Helper()
	got := map[string]string{}
	for _, name := range []string{spec.ReviewFile, spec.ReviewRequestFile} {
		data, err := os.ReadFile(filepath.Join(root, "specs", "AI-001-x", name))
		if err != nil {
			t.Fatal(err)
		}
		got[name] = string(data)
	}
	return got
}

func assertEvidenceUnchanged(t *testing.T, root string, before map[string]string) {
	t.Helper()
	for name, after := range specEvidence(t, root) {
		if after != before[name] {
			t.Errorf("%s changed, but nothing may be written before a reviewer starts:\nbefore: %s\nafter:  %s",
				name, before[name], after)
		}
	}
}

func TestSpecReviewInitialLaunchesForARootCommitSpec(t *testing.T) {
	root := seedReviewFixture(t)
	withReviewBase(t, spec.ReviewBase{Refusal: spec.BaseRootCommit, RootSHA: "rootrootroot"}, "headheadhead")
	withEmptyTree(t, "treetreetree")
	before := specEvidence(t, root)

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--initial", "--dry-run")
	if err != nil {
		t.Fatalf("--initial was refused for a root-commit spec: %v\n%s", err, stdout+stderr)
	}
	out := stdout + stderr
	for _, want := range []string{"initial review", "git diff treetreetree HEAD", "rootrootroot"} {
		if !strings.Contains(out, want) {
			t.Errorf("the dry run does not describe the initial scope (%q):\n%s", want, out)
		}
	}
	if strings.Contains(out, "treetreetree...") {
		t.Errorf("the scope is a triple-dot diff from a tree, which git refuses:\n%s", out)
	}
	assertEvidenceUnchanged(t, root, before)
}

func TestSpecReviewInitialRefusesEveryOtherBaseState(t *testing.T) {
	for name, base := range map[string]spec.ReviewBase{
		"ordinary spec":    {SHA: "basebasebase"},
		"uncommitted spec": {Refusal: spec.BaseNotCommitted},
		"untraceable move": {Refusal: spec.BaseUntraceable},
	} {
		t.Run(name, func(t *testing.T) {
			root := seedReviewFixture(t)
			withReviewBase(t, base, "headheadhead")
			withEmptyTree(t, "treetreetree")
			before := specEvidence(t, root)

			stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--initial", "--dry-run")
			if err == nil {
				t.Fatalf("--initial was accepted for a spec not born in the root commit:\n%s", stdout+stderr)
			}
			if !strings.Contains(err.Error(), "--initial") {
				t.Errorf("the refusal does not say why --initial does not apply: %v", err)
			}
			assertEvidenceUnchanged(t, root, before)
		})
	}
}

// The default refusal stays, and points at the supported path when it applies.
func TestSpecReviewWithoutInitialRefusesARootCommitSpecAndNamesTheFlag(t *testing.T) {
	seedReviewFixture(t)
	withReviewBase(t, spec.ReviewBase{Refusal: spec.BaseRootCommit, RootSHA: "rootrootroot"}, "headheadhead")

	stdout, stderr, err := execute(t, "spec", "review", "AI-001-x", "--dry-run")
	if err == nil {
		t.Fatalf("a root-commit spec was launched without --initial:\n%s", stdout+stderr)
	}
	for _, want := range []string{"cannot resolve a review base", "--initial", "rootrootroot"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not carry %q: %v", want, err)
		}
	}
}

// The request is what the archive checks, so the command must hand the
// initial scope to it rather than a blank base.
func TestRecordReviewRequestWritesTheInitialScope(t *testing.T) {
	root := seedReviewFixture(t)
	specDir := filepath.Join(root, "specs", "AI-001-x")
	scope := spec.ReviewScope{BaseTree: "treetreetree", RootSHA: "rootrootroot"}

	if err := recordReviewRequest(&cobra.Command{}, root, specDir, "AI-001-x", spec.FirstSigner,
		"headheadhead", "nan/deepseek-v4-flash", scope, ""); err != nil {
		t.Fatal(err)
	}
	req, found, err := spec.ReadReviewRequest(specDir)
	if err != nil || !found {
		t.Fatalf("no request was written: found=%v err=%v", found, err)
	}
	if req.Scope != spec.ScopeInitial || req.RootSHA != "rootrootroot" || req.BaseTree != "treetreetree" || req.BaseSHA != "" {
		t.Errorf("the request does not record the initial scope: %+v", req)
	}
}

// launchInitialForeground runs `spec review --initial` with the runner stubbed
// out and returns the request the given slot ended up with.
func launchInitialForeground(t *testing.T, root string, slot spec.ReviewSlot, args ...string) spec.ReviewRequest {
	t.Helper()
	withReviewBase(t, spec.ReviewBase{Refusal: spec.BaseRootCommit, RootSHA: "rootrootroot"}, "headheadhead")
	withEmptyTree(t, "treetreetree")
	prev := runForeground
	runForeground = func(string, []string, string) error { return nil }
	t.Cleanup(func() { runForeground = prev })

	// The stub writes no verdict, so the launch ends refused for that and only
	// that; any earlier refusal would mean the request was never written.
	argv := append([]string{"spec", "review", "AI-001-x", "--initial", "--foreground"}, args...)
	if stdout, stderr, err := execute(t, argv...); err != nil && !strings.Contains(err.Error(), "exited without writing "+slot.Review) {
		t.Fatalf("%v: %v\n%s", argv, err, stdout+stderr)
	}
	req, found, err := spec.ReadReviewRequestIn(filepath.Join(root, "specs", "AI-001-x"), slot)
	if err != nil || !found {
		t.Fatalf("no %s was written: found=%v err=%v", slot.Request, found, err)
	}
	if req.Scope != spec.ScopeInitial || req.RootSHA != "rootrootroot" || req.BaseTree != "treetreetree" || req.BaseSHA != "" {
		t.Errorf("%s does not record the initial scope: %+v", slot.Request, req)
	}
	return req
}

// #3019: the second slot writes its request through the same scoped writer, so
// an initial second signature must record the initial scope in its own file.
func TestSpecReviewInitialSecondRecordsTheInitialScope(t *testing.T) {
	root := makeRepo(t)
	seedSignerPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\nrisk: high\n---\n# AI-001-x\n")
	specDir := filepath.Join(root, "specs", "AI-001-x")
	first := spec.ReviewScope{BaseTree: "treetreetree", RootSHA: "rootrootroot"}
	if err := spec.WriteReviewRequestScoped(specDir, spec.FirstSigner, "headheadhead", "nan/deepseek-v4-flash", first, ""); err != nil {
		t.Fatal(err)
	}

	req := launchInitialForeground(t, root, spec.SecondSigner, "--second")
	if req.Reviewer != "anthropic-review/claude-sonnet-5-5" {
		t.Errorf("the second slot did not launch the second signer: %+v", req)
	}
}

// #3019: a fallback launch records its reason and the initial scope together.
func TestSpecReviewInitialFallbackRecordsTheInitialScopeAndItsReason(t *testing.T) {
	root := makeRepo(t)
	seedSignerPool(t, root)
	seedSpec(t, root, "AI-001-x", "---\nstatus: implementing\n---\n# AI-001-x\n")

	req := launchInitialForeground(t, root, spec.FirstSigner, "--fallback-reason", "rate-limit")
	if req.Reviewer != "anthropic-review/claude-haiku-5-5" || req.FallbackReason != "rate-limit" {
		t.Errorf("want the fallback and its reason recorded with the initial scope, got %+v", req)
	}
}
