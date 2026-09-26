package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ApplyStatus grades one repository after an apply.
type ApplyStatus string

const (
	ApplyUnchanged ApplyStatus = "unchanged" // live already matches: changed=0
	ApplyPlanned   ApplyStatus = "planned"   // --dry-run: Changes is what a real run would write
	ApplyApplied   ApplyStatus = "applied"   // written, and the re-read matches the declaration
	ApplyRefused   ApplyStatus = "refused"   // the preflight refused; nothing was written
	ApplySkipped   ApplyStatus = "skipped"   // declared state, not an object: apply writes nothing
	ApplyFailed    ApplyStatus = "failed"    // a read or write failed, or the write did not take effect
)

// ApplyResult is the outcome for one repository.
type ApplyResult struct {
	Repo    string
	Status  ApplyStatus
	Changes []Change
	Detail  string
}

// reportWindow is how many of a branch's most recent merged pull requests the
// preflight reads for a report of a newly required context (GUARD-017 R-1).
const reportWindow = 5

// ApplyRepo converges one repository's live protection on its declaration.
//
// It reads live, diffs, and writes only on a difference, so a second run
// reports changed=0. The write is the complete object, because PUT replaces
// the whole of it and an omitted field is one it silently clears (#1451 note
// 1). A 200 means the request was accepted, not that it took effect, so the
// result is re-read and diffed again. Declared states are never written:
// removing protection is not something a sync does as a side effect.
func ApplyRepo(repo string, d RepoDecl, run Runner, dryRun bool) ApplyResult {
	res := ApplyResult{Repo: repo}
	if d.Protection == nil {
		res.Status, res.Detail = ApplySkipped, "declared "+d.State+"; apply writes declared protection objects only"
		return res
	}
	path := fmt.Sprintf("repos/%s/branches/%s/protection", repo, d.Branch)
	live, protected, err := readLive(run, path)
	if err != nil {
		res.Status, res.Detail = ApplyFailed, err.Error()
		return res
	}
	res.Changes = Diff(*d.Protection, live)
	if !protected {
		res.Changes = append([]Change{{Field: "protection", Declared: "required", Live: "none"}}, res.Changes...)
	}
	if len(res.Changes) == 0 {
		res.Status = ApplyUnchanged
		return res
	}
	if missing, err := unreportedContexts(repo, d, live, run); err != nil || len(missing) > 0 {
		res.Status, res.Detail = ApplyRefused, refusal(missing, err)
		return res
	}
	if dryRun {
		res.Status = ApplyPlanned
		return res
	}
	res.Status, res.Detail = write(run, path, *d.Protection, live)
	return res
}

// readLive GETs the protection of a branch. A branch with none reads as a live
// object with every block absent, and protected=false.
func readLive(run Runner, path string) (live Live, protected bool, err error) {
	out, errOut, err := run("api", path)
	if err != nil {
		if isNotProtected(out, errOut) {
			return Live{}, false, nil
		}
		return Live{}, false, fmt.Errorf("could not read %s: %s", path, unanswerable(errOut, err))
	}
	live, err = Normalise([]byte(out))
	return live, err == nil, err
}

// write PUTs the complete object, sets required_signatures through its own
// endpoint when it differs, and re-reads to prove the fields took effect.
func write(run Runner, path string, p Protection, live Live) (ApplyStatus, string) {
	if err := putProtection(run, path, p); err != nil {
		return ApplyFailed, err.Error()
	}
	if p.RequiredSignatures != live.Protection.RequiredSignatures {
		method := "DELETE"
		if p.RequiredSignatures {
			method = "POST"
		}
		if _, errOut, err := run("api", "-X", method, path+"/required_signatures"); err != nil {
			return ApplyFailed, "required_signatures: " + unanswerable(errOut, err)
		}
	}
	after, protected, err := readLive(run, path)
	if err != nil || !protected {
		return ApplyFailed, fmt.Sprintf("written, but the re-read does not show it (protected=%v): %v", protected, err)
	}
	if left := Diff(p, after); len(left) > 0 {
		return ApplyFailed, "the forge accepted the write but did not apply: " + fieldNames(left)
	}
	return ApplyApplied, ""
}

func putProtection(run Runner, path string, p Protection) error {
	body, err := PutBody(p)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "dotf-protection-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.Write(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, errOut, err := run("api", "-X", "PUT", path, "--input", f.Name()); err != nil {
		return fmt.Errorf("PUT %s: %s", path, unanswerable(errOut, err))
	}
	return nil
}

// putBody is the PUT shape. Every key is always present — restrictions as an
// explicit null — because the endpoint replaces the whole object.
// required_signatures is absent on purpose: PUT does not accept it.
type putBody struct {
	RequiredStatusChecks           *StatusChecks `json:"required_status_checks"`
	RequiredPullRequestReviews     *Reviews      `json:"required_pull_request_reviews"`
	EnforceAdmins                  bool          `json:"enforce_admins"`
	Restrictions                   *struct{}     `json:"restrictions"`
	RequiredLinearHistory          bool          `json:"required_linear_history"`
	AllowForcePushes               bool          `json:"allow_force_pushes"`
	AllowDeletions                 bool          `json:"allow_deletions"`
	BlockCreations                 bool          `json:"block_creations"`
	RequiredConversationResolution bool          `json:"required_conversation_resolution"`
	LockBranch                     bool          `json:"lock_branch"`
	AllowForkSyncing               bool          `json:"allow_fork_syncing"`
}

// PutBody renders p as the complete body of PUT /branches/{b}/protection.
func PutBody(p Protection) ([]byte, error) {
	return json.Marshal(putBody{
		RequiredStatusChecks:           p.RequiredStatusChecks,
		RequiredPullRequestReviews:     p.RequiredPullRequestReviews,
		EnforceAdmins:                  p.EnforceAdmins,
		RequiredLinearHistory:          p.RequiredLinearHistory,
		AllowForcePushes:               p.AllowForcePushes,
		AllowDeletions:                 p.AllowDeletions,
		BlockCreations:                 p.BlockCreations,
		RequiredConversationResolution: p.RequiredConversationResolution,
		LockBranch:                     p.LockBranch,
		AllowForkSyncing:               p.AllowForkSyncing,
	})
}

func fieldNames(changes []Change) string {
	names := make([]string, 0, len(changes))
	for _, c := range changes {
		names = append(names, c.Field)
	}
	return strings.Join(names, ", ")
}

func refusal(missing []Check, err error) string {
	if err != nil {
		return "could not confirm that every newly required context reports: " + err.Error()
	}
	return fmt.Sprintf("%s would become required but did not report on any of the last %d merged pull requests; "+
		"with enforce_admins that locks the branch, the owner included", checkList(missing), reportWindow)
}
