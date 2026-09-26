package forge

import (
	"encoding/json"
	"fmt"
)

// unreportedContexts returns the contexts d would newly require that did not
// report on any of the branch's last reportWindow merged pull requests.
//
// With enforce_admins, a required context that never reports — an app outage,
// a job a path filter skips, a renamed job — makes the branch unmergeable by
// the owner too (#1451 note 2). Only NEW contexts are checked: the preflight
// guards the change, not the status quo, and a context already required that
// stopped reporting is drift for `check`, not a reason to refuse a sync.
func unreportedContexts(repo string, d RepoDecl, live Live, run Runner) ([]Check, error) {
	added := newChecks(d.Protection, live.Protection)
	if len(added) == 0 {
		return nil, nil
	}
	heads, err := mergedHeads(repo, d.Branch, run)
	if err != nil {
		return nil, err
	}
	var missing []Check
	for _, c := range added {
		ok, err := reportedOnAny(repo, heads, c, run)
		if err != nil {
			return nil, err
		}
		if !ok {
			missing = append(missing, c)
		}
	}
	return missing, nil
}

// newChecks are the declared checks live does not already require, compared
// by context and source app, as the diff compares them.
func newChecks(declared *Protection, live Protection) []Check {
	if declared.RequiredStatusChecks == nil {
		return nil
	}
	have := map[string]bool{}
	if live.RequiredStatusChecks != nil {
		for _, c := range live.RequiredStatusChecks.Checks {
			have[checkList([]Check{c})] = true
		}
	}
	var added []Check
	for _, c := range declared.RequiredStatusChecks.Checks {
		if !have[checkList([]Check{c})] {
			added = append(added, c)
		}
	}
	return added
}

// mergedHeads returns the head SHAs of the branch's most recent merged pull
// requests, newest first. A closed pull request that never merged is skipped:
// what reported on an abandoned change says nothing about what reports now.
func mergedHeads(repo, branch string, run Runner) ([]string, error) {
	out, errOut, err := run("api", fmt.Sprintf("repos/%s/pulls?state=closed&base=%s&sort=updated&direction=desc&per_page=30", repo, branch))
	if err != nil {
		return nil, fmt.Errorf("list merged pull requests: %s", unanswerable(errOut, err))
	}
	var pulls []struct {
		MergedAt *string `json:"merged_at"`
		Head     struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := json.Unmarshal([]byte(out), &pulls); err != nil {
		return nil, fmt.Errorf("unexpected pull request list: %w", err)
	}
	var heads []string
	for _, p := range pulls {
		if p.MergedAt != nil && len(heads) < reportWindow {
			heads = append(heads, p.Head.SHA)
		}
	}
	return heads, nil
}

// reportedOnAny reports whether c was posted on any of heads: as a check run
// from its pinned app (any app when none is pinned), or as a commit status.
// The statuses API names no app, so a status counts by context alone.
func reportedOnAny(repo string, heads []string, c Check, run Runner) (bool, error) {
	for _, sha := range heads {
		ok, err := reportedOn(repo, sha, c, run)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

func reportedOn(repo, sha string, c Check, run Runner) (bool, error) {
	var runs struct {
		CheckRuns []struct {
			Name string `json:"name"`
			App  struct {
				ID int `json:"id"`
			} `json:"app"`
		} `json:"check_runs"`
	}
	if err := getJSON(run, fmt.Sprintf("repos/%s/commits/%s/check-runs?per_page=100", repo, sha), &runs); err != nil {
		return false, err
	}
	for _, r := range runs.CheckRuns {
		if r.Name == c.Context && (c.AppID == nil || r.App.ID == *c.AppID) {
			return true, nil
		}
	}
	var status struct {
		Statuses []struct {
			Context string `json:"context"`
		} `json:"statuses"`
	}
	if err := getJSON(run, fmt.Sprintf("repos/%s/commits/%s/status", repo, sha), &status); err != nil {
		return false, err
	}
	for _, s := range status.Statuses {
		if s.Context == c.Context {
			return true, nil
		}
	}
	return false, nil
}

func getJSON(run Runner, path string, v any) error {
	out, errOut, err := run("api", path)
	if err != nil {
		return fmt.Errorf("%s: %s", path, unanswerable(errOut, err))
	}
	if err := json.Unmarshal([]byte(out), v); err != nil {
		return fmt.Errorf("%s: unexpected response: %w", path, err)
	}
	return nil
}
