package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ReviewSlot names the files one signature of a spec lives in.
//
// A spec has up to two signatures (AI-045, #1923). The first is the review every
// spec gets. The second is a review by a different vendor, required when
// proposal.md declares `risk: high`. Each slot keeps its own verdict, launch
// request and transcript, so one signer can never overwrite the other's evidence
// and the gate checks each file pair separately.
type ReviewSlot struct {
	Name       string
	Review     string
	Request    string
	Transcript string
}

var (
	// FirstSigner is the slot every reviewed spec has. Its file names predate
	// the second slot and stay as they were, so existing specs keep validating.
	FirstSigner = ReviewSlot{Name: "first", Review: ReviewFile, Request: ReviewRequestFile, Transcript: TranscriptFile}
	// SecondSigner is the cross-vendor signature a `risk: high` spec also needs.
	SecondSigner = ReviewSlot{
		Name:       "second",
		Review:     "review-second.md",
		Request:    "review-second-request.json",
		Transcript: "review-second-transcript.jsonl",
	}
)

// Session is the tmux session a launch of this slot runs under. The second slot
// gets its own name, so it can run while the first is still going.
func (s ReviewSlot) Session(specID string) string {
	if s.Name == SecondSigner.Name {
		return TmuxSession(specID) + "-second"
	}
	return TmuxSession(specID)
}

// Flag is the `dotf spec review` flag that launches this slot.
func (s ReviewSlot) Flag() string {
	if s.Name == SecondSigner.Name {
		return " --second"
	}
	return ""
}

// The `signs` values of a pool entry: which signature a member may give.
//
//   - first: a first signer, drawn by default (an absent `signs` means this).
//   - second: only a second signature, beside a first signer of another vendor.
//   - fallback: a first signature only when the first signers failed for a
//     classified reason, recorded in the launch request.
//
// Amendment B to the pool rule (#1923): an Anthropic model never signs alone.
// Pool loading refuses an Anthropic entry that declares `first`, so an implementer
// from that vendor is never graded by a same-vendor model unless a first signer
// of another vendor has signed too, or every first signer has failed.
const (
	SignsFirst    = "first"
	SignsSecond   = "second"
	SignsFallback = "fallback"
)

// FallbackReasons are the classified first-signer failures that let a
// `signs: fallback` member sign first. Every one is a failure of the provider,
// never a verdict anyone disliked: a first signer that answered FAIL is a
// review, and re-drawing until something passes is exactly what this list
// exists to rule out.
var FallbackReasons = []string{"rate-limit", "quota", "provider-error", "timeout", "content-filter", "empty-answer"}

// SignatureRole is the entry's declared signature role; an absent one means first.
func (e ReviewerEntry) SignatureRole() string {
	if s := strings.TrimSpace(e.Signs); s != "" {
		return s
	}
	return SignsFirst
}

// IsAnthropic reports whether the entry runs an Anthropic model. A declared
// `vendor` answers it. Without one, any id, provider or model naming the vendor
// or its model family answers it too, so an entry cannot slip past
// amendment B by leaving the field out.
func (e ReviewerEntry) IsAnthropic() bool {
	if strings.EqualFold(strings.TrimSpace(e.Vendor), "anthropic") {
		return true
	}
	for _, f := range []string{e.ID, e.Provider, e.Model} {
		f = strings.ToLower(f)
		if strings.Contains(f, "anthropic") || strings.Contains(f, "claude") {
			return true
		}
	}
	return false
}

// vendor is who trains the entry's model, the unit the second signature must
// differ in. A declared `vendor` wins, then the Anthropic inference, then the
// provider, and last the runner.
func (e ReviewerEntry) vendor() string {
	if v := strings.ToLower(strings.TrimSpace(e.Vendor)); v != "" {
		return v
	}
	if e.IsAnthropic() {
		return "anthropic"
	}
	if p := strings.ToLower(strings.TrimSpace(e.Provider)); p != "" {
		return p
	}
	return strings.ToLower(strings.TrimSpace(e.Runner))
}

// validateSigns is the pool-load check of each entry's `signs`: a known value,
// and never `first` for an Anthropic model.
func validateSigns(e ReviewerEntry) error {
	switch e.SignatureRole() {
	case SignsFirst:
		if e.IsAnthropic() {
			return fmt.Errorf("pool entry %q is an Anthropic model that would sign first, and an Anthropic model never signs a review alone\n"+
				"declare `\"signs\": \"second\"` or `\"signs\": \"fallback\"` on it", e.ID)
		}
	case SignsSecond, SignsFallback:
	default:
		return fmt.Errorf("pool entry %q declares signs %q (want %s, %s or %s)", e.ID, e.Signs, SignsFirst, SignsSecond, SignsFallback)
	}
	return nil
}

// ReviewerChoice is what the launcher asks of the pool for one launch.
type ReviewerChoice struct {
	Slot ReviewSlot
	// Want names one member; empty draws one.
	Want string
	// FallbackReason, first slot only, launches a `signs: fallback` member in
	// place of the first signers. Must be one of FallbackReasons.
	FallbackReason string
	// FirstVendor, second slot only, is the vendor of the member that signed
	// first. The second signer must be of another vendor.
	FirstVendor string
}

// ChooseReviewer picks the pool member for one launch: Want when it names one,
// otherwise a draw among the members eligible for the slot. draw is the source
// of randomness (rand.IntN in production, a fixed index in tests).
//
// Eligibility is the same whether the member was named or drawn, so --reviewer
// cannot reach a member the draw would never pick: a second signer cannot sign
// first, a fallback cannot sign first without a reason, and a second signer
// cannot share the first signer's vendor.
func ChooseReviewer(entries []ReviewerEntry, c ReviewerChoice, draw func(n int) int) (ReviewerEntry, error) {
	if len(entries) == 0 {
		return ReviewerEntry{}, fmt.Errorf("no %s in this repo — the launcher has no model to run and will not guess one", ReviewerPoolFile)
	}
	reason := strings.TrimSpace(c.FallbackReason)
	if reason != "" {
		if c.Slot.Name == SecondSigner.Name {
			return ReviewerEntry{}, fmt.Errorf("--fallback-reason replaces a failed FIRST signer; a second signature has no fallback")
		}
		if !slices.Contains(FallbackReasons, reason) {
			return ReviewerEntry{}, fmt.Errorf("fallback reason %q is not a classified first-signer failure\nknown: %s",
				reason, strings.Join(FallbackReasons, ", "))
		}
	}

	if strings.TrimSpace(c.Want) != "" {
		e, err := ResolveReviewer(entries, c.Want)
		if err != nil {
			return ReviewerEntry{}, err
		}
		if why := ineligible(e, c.Slot, reason, c.FirstVendor); why != "" {
			return ReviewerEntry{}, fmt.Errorf("reviewer %q cannot sign this review: %s", e.ID, why)
		}
		return e, nil
	}

	var eligible []ReviewerEntry
	for _, e := range entries {
		if ineligible(e, c.Slot, reason, c.FirstVendor) == "" {
			eligible = append(eligible, e)
		}
	}
	if len(eligible) == 0 {
		return ReviewerEntry{}, fmt.Errorf("no member of %s can sign the %s review%s", ReviewerPoolFile, c.Slot.Name, forWhom(c, reason))
	}
	return DrawReviewer(eligible, draw)
}

// ineligible says why e cannot give this signature, or "" when it can. With a
// fallback reason the draw is among the fallbacks only: the reason says the
// first signers already failed.
func ineligible(e ReviewerEntry, slot ReviewSlot, reason, firstVendor string) string {
	signs := e.SignatureRole()
	if slot.Name == SecondSigner.Name {
		if signs != SignsSecond {
			return fmt.Sprintf("it signs %s, and only a `signs: second` member gives the second signature", signs)
		}
		if firstVendor != "" && e.vendor() == firstVendor {
			return fmt.Sprintf("it is of vendor %s, the same as the first signer, and the second signature exists to be of another", firstVendor)
		}
		return ""
	}
	switch {
	case signs == SignsSecond:
		return "it only gives a second signature (launch it with --second, after a first signer)"
	case signs == SignsFallback && reason == "":
		return "it is a fallback, which signs first only when the first signers failed: pass --fallback-reason"
	case signs == SignsFirst && reason != "":
		return "a fallback reason launches a `signs: fallback` member, and this one signs first"
	}
	return ""
}

func forWhom(c ReviewerChoice, reason string) string {
	switch {
	case reason != "":
		return " as a fallback (no member declares `signs: fallback`)"
	case c.Slot.Name == SecondSigner.Name && c.FirstVendor != "":
		return " (a `signs: second` member of a vendor other than " + c.FirstVendor + ")"
	case c.Slot.Name == SecondSigner.Name:
		return " (no member declares `signs: second`)"
	}
	return ""
}

// FirstSignerVendor is the vendor of the member the first signature was
// launched on, read from the launcher's own request rather than from review.md,
// which the reviewer writes. A second signature is launched against it, so it
// refuses when there is no first launch to be second to.
func FirstSignerVendor(specDir string, entries []ReviewerEntry) (string, error) {
	req, found, err := ReadReviewRequestIn(specDir, FirstSigner)
	if err != nil {
		return "", err
	}
	if !found || req.Reviewer == "" {
		return "", fmt.Errorf("no first review was launched for %s (%s is missing): the second signature is given after a first signer's\n"+
			"run `dotf spec review %s` first", filepath.Base(specDir), ReviewRequestFile, filepath.Base(specDir))
	}
	e, ok := poolEntry(entries, req.Reviewer)
	if !ok {
		return "", fmt.Errorf("the first review was launched on %q, which is no longer in %s; re-run the first review", req.Reviewer, ReviewerPoolFile)
	}
	return e.vendor(), nil
}

// RiskHigh reports whether proposal.md declares `risk: high`, which makes the
// second signature a requirement of the archive rather than an option.
func RiskHigh(specDir string) bool {
	data, err := os.ReadFile(filepath.Join(specDir, "proposal.md"))
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(frontmatterFields(string(data))["risk"]), "high")
}

// checkSigners is the last archive check: who signed, in which role. It runs
// after both signatures have passed provenance, verdict, staleness and pool
// membership, so it only asks the questions those cannot.
//
// A repo without a pool has no roles to check, so it passes as the pool check
// does. second is nil when no second signature is required or present.
func checkSigners(repoRoot, specDir string, first Review, second *Review) error {
	entries, err := loadReviewerPoolEntries(repoRoot)
	if err != nil {
		return fmt.Errorf("%w\nfix the file", err)
	}
	if entries == nil {
		return nil // no pool, no roles
	}
	firstEntry, ok := poolEntry(entries, first.Reviewer)
	if !ok {
		return nil
	}
	switch firstEntry.SignatureRole() {
	case SignsSecond:
		return fmt.Errorf("%s is signed by %q, which only gives a second signature: a first signer of another vendor must review first\n"+
			"run `dotf spec review %s` and let it draw a first signer",
			ReviewFile, first.Reviewer, filepath.Base(specDir))
	case SignsFallback:
		req, _, rerr := ReadReviewRequestIn(specDir, FirstSigner)
		if rerr != nil || !slices.Contains(FallbackReasons, req.FallbackReason) {
			return fmt.Errorf("%s is signed by the fallback %q, but %s records no classified reason the first signers failed\n"+
				"a fallback signs first only through `dotf spec review %s --fallback-reason <%s>`",
				ReviewFile, first.Reviewer, ReviewRequestFile, filepath.Base(specDir), strings.Join(FallbackReasons, "|"))
		}
	}
	if second == nil {
		return nil
	}
	secondEntry, ok := poolEntry(entries, second.Reviewer)
	if !ok {
		return nil
	}
	if secondEntry.SignatureRole() != SignsSecond {
		return fmt.Errorf("%s is signed by %q, which does not declare `signs: second`", SecondSigner.Review, second.Reviewer)
	}
	if secondEntry.vendor() == firstEntry.vendor() {
		return fmt.Errorf("%s and %s are both signed by vendor %s (%q, %q): the second signature has to come from another vendor",
			ReviewFile, SecondSigner.Review, firstEntry.vendor(), first.Reviewer, second.Reviewer)
	}
	return nil
}

func poolEntry(entries []ReviewerEntry, id string) (ReviewerEntry, bool) {
	for _, e := range entries {
		if strings.TrimSpace(e.ID) == id {
			return e, true
		}
	}
	return ReviewerEntry{}, false
}
