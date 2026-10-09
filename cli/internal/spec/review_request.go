package spec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ReviewRequestFile records what THIS repository asked a reviewer to review,
// written before the reviewer starts and never by the reviewer.
//
// The gate it feeds answers a question `review.md` alone cannot: is the verdict
// on disk the one this launch produced? Two live failures make that question
// load-bearing, and neither is hypothetical:
//
//   - Measured 2026-08-21 (#1157): a review ran 25 turns, resolved the right
//     tree, and was ended by the runner's turn cap before writing anything.
//     `dotf spec review` reported success — correctly, since detached launch
//     means it only ever reports that the runner STARTED — and the previous
//     round's verdict was still sitting in review.md, stamped with a sha that
//     was no longer any branch. A stale verdict left in place is indistinguishable
//     from a fresh one that reached the same conclusion.
//   - The same round's reviewer stamped `date: 2026-08-20` on a review run on the
//     21st. The model authors its own frontmatter, so `reviewed_sha` is a claim,
//     not a measurement, and the staleness gate downstream trusts it completely.
//
// So the launcher writes the sha it is actually reviewing, and the digest of
// whatever review.md held beforehand. Both are facts about the launch rather
// than assertions by the reviewed party, which is the entire point.
const ReviewRequestFile = "review-request.json"

// ReviewRequest is the sidecar's on-disk shape. Field names are snake_case to
// match every other registry in this repo.
type ReviewRequest struct {
	// ReviewedSHA is the repository HEAD at launch — what the reviewer was
	// pointed at, independent of what it later says it looked at.
	ReviewedSHA string `json:"reviewed_sha"`
	// Reviewer is the pool id the launcher resolved, so a verdict signed by a
	// different model is visible even when that model is itself pool-admitted.
	Reviewer string `json:"reviewer"`
	// RequestedAt is for humans reading the folder; nothing gates on it,
	// because a timestamp the launcher writes proves only when it ran.
	RequestedAt string `json:"requested_at"`
	// ReviewDigestBefore is the SHA-256 of review.md at launch, or "" when no
	// review.md existed. A digest that has not moved means the reviewer wrote
	// no verdict — the one case the launcher provably cannot observe itself.
	ReviewDigestBefore string `json:"review_digest_before"`
	// BaseSHA is the commit the spec's work starts FROM, so the reviewer has a
	// diff to read instead of a folder to browse.
	//
	// The skill specifies the review scope as `git diff <base>...HEAD`, and
	// until this field existed the launcher recorded no base at all: the
	// reviewer had to guess it, guessed `main`, and a review launched on `main`
	// after the work merged therefore diffed nothing. Measured on BUG-093 round
	// 4 — it declared `git diff main...HEAD` as its source, that diff was empty,
	// and both of its findings say "Code read" with nothing executed. One of the
	// two was factually wrong, having read a mutation payload out of the spec
	// folder as if it were the implementation.
	//
	// omitempty because specs archived before this field existed carry the old
	// shape and must still validate.
	BaseSHA string `json:"base_sha,omitempty"`
	// ContractDigests is the normalised SHA-256 of each contract file as the
	// reviewer found it on disk (SDD-042). Archive recomputes them and decides
	// freshness by content, so a squash-merge, rebase or fresh clone that
	// discards ReviewedSHA's commit no longer decides the question (#1566).
	// Absent on requests written before SDD-042, which keep the SHA check.
	ContractDigests map[string]string `json:"contract_digests,omitempty"`
	// FallbackReason is why a `signs: fallback` pool member was launched as
	// the first signer (AI-045, #1923): the classified failure of the first
	// signers it stands in for. The launcher refuses a fallback member without
	// one, and the archive gate refuses a fallback signature whose request
	// carries none, so a fallback never signs by habit or by draw.
	FallbackReason string `json:"fallback_reason,omitempty"`
}

// HeadSHA returns the repository HEAD, or "" when repoRoot is not a checkout.
//
// Empty rather than an error on purpose: a missing HEAD must not stop a review
// from launching. It degrades the provenance gate to "not asserted", which is
// honest, where refusing to launch would make the guard a liability.
func HeadSHA(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RequestIgnored reports whether git ignores the review request in specDir.
// An ignored request is written, read by a local archive, and lost on every
// other checkout, so the review it records cannot archive anywhere else
// (#1908). A tracked request is not ignored, whatever the rules say: a
// `git add -f` already makes it travel. Any answer but "ignored" (not a
// repository, git missing) is false:
// the archive gate still refuses a request that did not travel, so failing
// open here only moves the refusal later.
func RequestIgnored(repoRoot, specDir string) bool {
	return RequestIgnoredIn(repoRoot, specDir, FirstSigner)
}

// RequestIgnoredIn is RequestIgnored for the request of one signer's slot.
func RequestIgnoredIn(repoRoot, specDir string, slot ReviewSlot) bool {
	path := filepath.Join(specDir, slot.Request)
	return exec.Command("git", "-C", repoRoot, "check-ignore", "-q", "--", path).Run() == nil
}

// ResolveReviewBase returns the commit the spec's work starts from: the PARENT
// of the commit that first added specDir.
//
// Why this and not the PR's base branch. The PR route needs the network, `gh`
// auth and PR metadata a human can edit, and it answers differently depending
// on which of a spec's several PRs you ask about. This is local, offline and
// deterministic, and it is correct in BOTH of the situations that matter:
//
//   - review on the work branch — the adding commit is on the branch, so the
//     parent is where the branch left main.
//   - review on main after a squash merge — the adding commit IS the squash
//     commit, so the parent is main immediately before the work landed.
//
// That second case is the whole point: it is the one the old code got wrong by
// having no answer at all.
//
// A folder that was renamed -- by hand, or by `spec archive` moving it under
// specs/archive/ -- appears ADDED under its new path, so the search follows each
// rename back to the folder's first name (BUG-108, #1829). Without that, the base
// was the parent of the rename, after the implementation, and the review diff
// held the rename and nothing else.
//
// Returns "" when there is no such commit (a spec folder not yet committed),
// which the caller must treat as "cannot review", not as "review everything".
// It also returns "" when the folder, under any of its names, was added in the
// repository's root commit: there is no "before" to compare against, and
// reviewing the entire repository history is not what was asked for.
func ResolveReviewBase(repoRoot, specDir string) string {
	rel, err := filepath.Rel(repoRoot, specDir)
	if err != nil {
		rel = specDir
	}
	rel = filepath.ToSlash(rel)
	// Bounded so a pathological history cannot loop; a spec renamed more than
	// a handful of times is refused, not reviewed against a guessed base.
	for hop := 0; hop < 8; hop++ {
		adding := earliestAdding(repoRoot, rel)
		if adding == "" {
			return ""
		}
		out, err := exec.Command("git", "-C", repoRoot, "rev-parse", adding+"^").Output()
		if err != nil {
			return "" // the adding commit is the root commit
		}
		parent := strings.TrimSpace(string(out))
		from, renamed := renamedFrom(repoRoot, parent, adding, rel)
		if !renamed {
			return parent
		}
		if from == "" {
			// Something was renamed into the folder but its source cannot be
			// traced. The parent would be a base after the work, which is the
			// partial diff this function exists to refuse.
			return ""
		}
		rel = from
	}
	return ""
}

// earliestAdding is the first commit that added a file under rel. --diff-filter=A finds the commits that ADDED it; the last line of a
// reverse-chronological log is the earliest. `--` guards a path that could be
// read as a revision.
func earliestAdding(repoRoot, rel string) string {
	out, err := exec.Command("git", "-C", repoRoot,
		"log", "--diff-filter=A", "--format=%H", "--", rel).Output()
	if err != nil {
		return ""
	}
	lines := strings.Fields(strings.TrimSpace(string(out)))
	if len(lines) == 0 {
		return ""
	}
	return lines[len(lines)-1]
}

// renamedFrom reports whether commit renamed anything into rel and, if so, the
// folder it came from. renamed with an empty from means the source could not be
// traced, which the caller refuses.
//
// A move that also edits a file past git's similarity threshold is not paired:
// it shows as a delete and an add. A file deleted elsewhere in the same commit
// whose place in its folder matches a file added under rel is read as that
// unpaired move, and refused rather than reported as "genuinely new here".
//
// Both readings are confined to the specs root, rel's first component: a spec
// folder lives there under every name it has had. A file moved in from outside
// it (docs/notes.md into the spec) is content joining the folder, not the
// folder moving, and a delete outside it is unrelated work in the same commit.
func renamedFrom(repoRoot, parent, commit, rel string) (from string, renamed bool) {
	changes, err := treeChanges(repoRoot, parent, commit)
	if err != nil {
		return "", true
	}
	root, _, _ := strings.Cut(rel, "/")
	root += "/"
	var added, deleted []string
	for _, c := range changes {
		inRel := strings.HasPrefix(c.dst, rel+"/")
		fromRoot := strings.HasPrefix(c.src, root) && !strings.HasPrefix(c.src, rel+"/")
		switch {
		case c.status == 'R' && inRel && fromRoot:
			renamed = true
			if dir := renamedDir(c.src, c.dst, rel); dir != "" {
				return dir, true
			}
		case c.status == 'A' && inRel:
			added = append(added, strings.TrimPrefix(c.dst, rel+"/"))
		case c.status == 'D' && fromRoot:
			deleted = append(deleted, c.src)
		}
	}
	for _, d := range deleted {
		for _, a := range added {
			if strings.HasSuffix(d, "/"+a) {
				return "", true
			}
		}
	}
	return "", renamed
}

// treeChange is one record of `git diff-tree --name-status`: src is the path
// before, dst the path after, the same path for anything but a rename or copy.
type treeChange struct {
	status   byte
	src, dst string
}

// treeChanges lists what commit changed against parent, with rename detection.
// -z keeps paths unquoted whatever core.quotePath says.
func treeChanges(repoRoot, parent, commit string) ([]treeChange, error) {
	out, err := exec.Command("git", "-C", repoRoot,
		"diff-tree", "-r", "-M", "--name-status", "-z", parent, commit).Output()
	if err != nil {
		return nil, err
	}
	// Records are status NUL path, or for a rename or copy status NUL src NUL dst.
	f := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var changes []treeChange
	for i := 0; i+1 < len(f); {
		status := f[i]
		if status == "" {
			break
		}
		if status[0] == 'R' || status[0] == 'C' {
			if i+2 >= len(f) {
				break
			}
			changes = append(changes, treeChange{status[0], f[i+1], f[i+2]})
			i += 3
			continue
		}
		changes = append(changes, treeChange{status[0], f[i+1], f[i+1]})
		i += 2
	}
	return changes, nil
}

// renamedDir is the folder oldPath sat in when its rename to newPath moved it
// into rel: oldPath less as many trailing components as newPath has below rel.
// Counting components rather than matching the suffix keeps a file renamed in
// the same commit as the move (proposal.md to spec.md) traceable. "" when
// oldPath is too shallow to have held it.
func renamedDir(oldPath, newPath, rel string) string {
	suffix, ok := strings.CutPrefix(newPath, rel+"/")
	if !ok {
		return ""
	}
	parts := strings.Split(oldPath, "/")
	depth := strings.Count(suffix, "/") + 1
	if len(parts) <= depth {
		return ""
	}
	return strings.Join(parts[:len(parts)-depth], "/")
}

// fileDigest returns the SHA-256 of path, or "" when it does not exist.
//
// "" is also what an unreadable file yields, and that is deliberate: it makes
// the later comparison fail OPEN into "the digest moved", never into a silent
// pass. A guard that cannot read the file must not conclude the file is fine.
func fileDigest(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// WriteReviewRequest records the launch in specDir, replacing any previous one.
//
// Replacing rather than appending: the sidecar describes the CURRENT outstanding
// request, and a history of requests is what the transcript is for.
func WriteReviewRequest(specDir, reviewedSHA, reviewer, baseSHA string) error {
	return WriteReviewRequestIn(specDir, FirstSigner, reviewedSHA, reviewer, baseSHA, "")
}

// WriteReviewRequestIn records the launch of one signer's slot. fallbackReason
// is empty unless a `signs: fallback` member was launched as the first signer.
func WriteReviewRequestIn(specDir string, slot ReviewSlot, reviewedSHA, reviewer, baseSHA, fallbackReason string) error {
	req := ReviewRequest{
		ReviewedSHA:        reviewedSHA,
		Reviewer:           reviewer,
		RequestedAt:        time.Now().UTC().Format(time.RFC3339),
		ReviewDigestBefore: fileDigest(filepath.Join(specDir, slot.Review)),
		BaseSHA:            baseSHA,
		ContractDigests:    ContractDigests(specDir),
		FallbackReason:     fallbackReason,
	}
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding the review request: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(specDir, slot.Request), data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", slot.Request, err)
	}
	return nil
}

// ReadReviewRequest loads the sidecar. found is false when there is none, which
// is not an error: reviews predating this file, and hand-written ones, are still
// governed by the verdict and staleness checks.
func ReadReviewRequest(specDir string) (ReviewRequest, bool, error) {
	return ReadReviewRequestIn(specDir, FirstSigner)
}

// ReadReviewRequestIn loads the request of one signer's slot.
func ReadReviewRequestIn(specDir string, slot ReviewSlot) (ReviewRequest, bool, error) {
	data, err := os.ReadFile(filepath.Join(specDir, slot.Request))
	if os.IsNotExist(err) {
		return ReviewRequest{}, false, nil
	}
	if err != nil {
		return ReviewRequest{}, false, fmt.Errorf("reading %s: %w", slot.Request, err)
	}
	var req ReviewRequest
	if err := json.Unmarshal(data, &req); err != nil {
		// Loud, not skipped. An unparseable sidecar is the shape C15 forbids:
		// treating it as absent would silently drop the guard exactly when the
		// file that carries it is damaged.
		return ReviewRequest{}, true, fmt.Errorf("%s is not valid JSON: %w", slot.Request, err)
	}
	return req, true, nil
}

// VerifyReviewProduced reports whether the run that has just finished left a
// verdict, and is the launcher's own half of the guard checkReviewProvenance
// enforces later.
//
// Later is the problem it fixes. The archive gate answers the same question, but
// only when somebody next tries to archive -- which can be days on, in another
// session, with the transcript no longer in hand. A FOREGROUND run knows the
// answer the moment the runner exits, and every Windows run is one, because tmux
// is Linux-only.
//
// Measured 2026-08-29 (#1383): AI-042 round 4 ran 40 minutes, made 248 bash calls,
// wrote no review.md, and `dotf spec review` exited 0. A review that produced no
// file is a failed review, not a green one.
//
// Detached launches are out of scope by construction: the command returns while
// the reviewer is still running, so there is nothing yet to verify.
func VerifyReviewProduced(specDir, transcript string) error {
	return VerifyReviewProducedIn(specDir, FirstSigner, transcript)
}

// VerifyReviewProducedIn is VerifyReviewProduced for one signer's slot.
func VerifyReviewProducedIn(specDir string, slot ReviewSlot, transcript string) error {
	digest := fileDigest(filepath.Join(specDir, slot.Review))
	if digest == "" {
		return fmt.Errorf("the reviewer exited without writing %s -- that is a failed review, not a passing one\n"+
			"what it did instead is in the transcript: %s\n"+
			"re-run the review (a run ended by a turn cap, a rate limit, or a reviewer that talked itself out of the job leaves exactly this state)",
			slot.Review, transcript)
	}

	req, found, err := ReadReviewRequestIn(specDir, slot)
	if err != nil {
		return err
	}
	if found && req.ReviewDigestBefore != "" && req.ReviewDigestBefore == digest {
		return fmt.Errorf("%s is byte-identical to what it held before this run -- the reviewer wrote no verdict\n"+
			"what is on disk is the PREVIOUS round's, which is not a review of this change\n"+
			"the transcript of the run that wrote nothing: %s",
			slot.Review, transcript)
	}
	if _, _, err := FindReviewIn(specDir, slot); err != nil {
		return fmt.Errorf("the reviewer wrote a malformed %s: %w\nreview transcript: %s",
			slot.Review, err, transcript)
	}
	return nil
}
