package spec

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// SDD-042 (#1566, #970). These fixtures build a real history: a spec on main,
// work on a branch, a review launched at the branch head, the review's own
// "tick the boxes" finding applied, and then a landing onto main. Squash,
// rebase and merge must all be accepted — freshness is a property of the
// CONTENT the reviewer read, not of whether its commit survived — and a
// reworded criterion must still be refused.

const csID = "AI-001-content"

const (
	csProposal = "---\nid: \"" + csID + "\"\nstatus: verifying\n---\n\n## Acceptance criteria\n\n- [ ] **AC1** — refuses a closed issue\n"
	csTasks    = "- [ ] write the failing test\n- [ ] make it pass\n"
	csFeatures = `[{"id":"f1","behavior":"refuses","verification":"go test","state":"pending","evidence":""}]` + "\n"
	csReviewer = "nan/deepseek-v4-flash"
)

func csWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reviewedBranch builds the history up to "review applied, not yet landed" and
// returns the repo root and the reviewed sha. afterReview edits the spec after
// the review, before the landing; nil applies only the reviewer's own finding.
func reviewedBranch(t *testing.T, afterReview func(root string)) (root, reviewed string) {
	t.Helper()
	root = t.TempDir()
	gitRun(t, root, "init", "-q", "-b", "main")
	writePool(t, root, testPool)
	csWrite(t, root, "specs/"+csID+"/proposal.md", csProposal)
	csWrite(t, root, "specs/"+csID+"/tasks.md", csTasks)
	csWrite(t, root, "specs/"+csID+"/features.json", csFeatures)
	csWrite(t, root, "cli/feature.go", "package cli\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "spec and base")
	base := gitRun(t, root, "rev-parse", "HEAD")

	gitRun(t, root, "checkout", "-qb", "feat")
	csWrite(t, root, "cli/feature.go", "package cli\n\nfunc Refuse() bool { return true }\n")
	gitRun(t, root, "commit", "-qam", "work")
	reviewed = gitRun(t, root, "rev-parse", "HEAD")

	specDir := filepath.Join(root, "specs", csID)
	if err := WriteReviewRequest(specDir, reviewed, csReviewer, base); err != nil {
		t.Fatal(err)
	}
	csWrite(t, root, "specs/"+csID+"/review.md",
		"---\nspec: \""+csID+"\"\nverdict: \"PASS\"\nreviewed_sha: \""+reviewed+"\"\nreviewer: \""+csReviewer+"\"\n---\nMinor: tick the boxes.\n")

	// The reviewer's own finding, applied: progress, not a contract change.
	csWrite(t, root, "specs/"+csID+"/tasks.md", strings.ReplaceAll(csTasks, "- [ ]", "- [x]"))
	csWrite(t, root, "specs/"+csID+"/features.json",
		`[{"id":"f1","behavior":"refuses","verification":"go test","state":"passing","evidence":"ok"}]`+"\n")
	if afterReview != nil {
		afterReview(root)
	}
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "review applied")
	return root, reviewed
}

// prune makes the reviewed commit unreachable AND absent, the state a fresh
// clone or CI sees — the machine that did the work keeps it until gc, which is
// why #1566's gate passed there and nowhere else.
func prune(t *testing.T, root string) {
	t.Helper()
	gitRun(t, root, "branch", "-D", "feat")
	gitRun(t, root, "reflog", "expire", "--expire=now", "--all")
	gitRun(t, root, "gc", "-q", "--prune=now")
}

func land(t *testing.T, root, how string) {
	t.Helper()
	gitRun(t, root, "checkout", "-q", "main")
	csWrite(t, root, "README.md", "unrelated work on main\n")
	gitRun(t, root, "add", "-A")
	gitRun(t, root, "commit", "-qm", "unrelated")
	switch how {
	case "squash":
		gitRun(t, root, "merge", "-q", "--squash", "feat")
		gitRun(t, root, "commit", "-qm", "feat (squashed)")
	case "rebase":
		gitRun(t, root, "checkout", "-q", "feat")
		gitRun(t, root, "rebase", "-q", "main")
		gitRun(t, root, "checkout", "-q", "main")
		gitRun(t, root, "merge", "-q", "--ff-only", "feat")
	case "merge":
		gitRun(t, root, "merge", "-q", "--no-ff", "feat", "-m", "merge feat")
	default:
		t.Fatalf("unknown landing %q", how)
	}
}

func objectExists(t *testing.T, root, sha string) bool {
	t.Helper()
	return exec.Command("git", "-C", root, "cat-file", "-e", sha+"^{commit}").Run() == nil
}

func archiveOnMain(t *testing.T, root string) error {
	t.Helper()
	_, err := Archive(root, csID, ArchiveOptions{Date: "2026-09-23"})
	return err
}

func TestStaleSquashLandingIsAccepted(t *testing.T) {
	root, reviewed := reviewedBranch(t, nil)
	land(t, root, "squash")
	prune(t, root)
	if objectExists(t, root, reviewed) {
		t.Fatal("fixture: the squash-orphaned reviewed commit must be gone, or this proves nothing about #1566")
	}
	if err := archiveOnMain(t, root); err != nil {
		t.Fatalf("a squash landing of the reviewed content must archive: %v", err)
	}
}

func TestStaleRebaseLandingIsAccepted(t *testing.T) {
	root, reviewed := reviewedBranch(t, nil)
	land(t, root, "rebase")
	prune(t, root)
	if objectExists(t, root, reviewed) {
		t.Fatal("fixture: the pre-rebase reviewed commit must be gone")
	}
	if err := archiveOnMain(t, root); err != nil {
		t.Fatalf("a rebase landing of the reviewed content must archive: %v", err)
	}
}

func TestStaleMergeLandingIsAccepted(t *testing.T) {
	root, _ := reviewedBranch(t, nil)
	land(t, root, "merge")
	if err := archiveOnMain(t, root); err != nil {
		t.Fatalf("a merge-commit landing of the reviewed content must archive: %v", err)
	}
}

func TestStaleContractEditRefused(t *testing.T) {
	root, _ := reviewedBranch(t, func(root string) {
		csWrite(t, root, "specs/"+csID+"/proposal.md",
			strings.Replace(csProposal, "refuses a closed issue", "warns on a closed issue", 1))
	})
	land(t, root, "squash")
	prune(t, root)
	err := archiveOnMain(t, root)
	if err == nil || !strings.Contains(err.Error(), "proposal.md") {
		t.Fatalf("a reworded criterion after the review must refuse, naming proposal.md: %v", err)
	}
	for _, unmoved := range []string{"tasks.md", "features.json"} {
		if strings.Contains(err.Error(), unmoved) {
			t.Errorf("the refusal names %s, whose change was bookkeeping only: %v", unmoved, err)
		}
	}
}

// A review launched before SDD-042 carries no contract_digests. It keeps the
// SHA-based check, and that check now says why it cannot answer instead of
// guessing "rewritten by a rebase?" — it still refuses, because it cannot prove
// the content is the reviewed content.
func TestStaleLegacyAbsentObjectNamesTheCause(t *testing.T) {
	root, reviewed := reviewedBranch(t, nil)
	reqPath := filepath.Join(root, "specs", csID, ReviewRequestFile)
	var req map[string]any
	data, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatal(err)
	}
	delete(req, "contract_digests")
	legacy, _ := json.Marshal(req)
	if err := os.WriteFile(reqPath, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "commit", "-qam", "a pre-SDD-042 request")
	land(t, root, "squash")
	prune(t, root)
	if objectExists(t, root, reviewed) {
		t.Fatal("fixture: the reviewed commit must be gone")
	}

	err = archiveOnMain(t, root)
	if err == nil {
		t.Fatal("a legacy review whose reviewed commit is gone cannot be proven fresh and must refuse")
	}
	for _, want := range []string{"not in this clone", "contract digests"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal should say %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "changed after") {
		t.Errorf("an absent object is not a changed file — the two causes must stay distinct: %v", err)
	}
}

func TestReviewRequestRecordsContractDigests(t *testing.T) {
	dir := t.TempDir()
	csWrite(t, dir, "proposal.md", csProposal)
	if err := WriteReviewRequest(dir, "sha", csReviewer, "base"); err != nil {
		t.Fatal(err)
	}
	req, found, err := ReadReviewRequest(dir)
	if err != nil || !found {
		t.Fatalf("read back: %v found=%v", err, found)
	}
	want := ContractDigests(dir)
	if len(req.ContractDigests) != len(want) {
		t.Fatalf("contract_digests = %v, want %v", req.ContractDigests, want)
	}
	for name, d := range want {
		if req.ContractDigests[name] != d {
			t.Errorf("%s: recorded %q, want %q", name, req.ContractDigests[name], d)
		}
	}
}
