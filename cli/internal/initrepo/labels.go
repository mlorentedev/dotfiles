package initrepo

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The color and description a created Dependabot label gets: dependabot.yml
// carries neither, and these match the label first created by hand (#1222).
const (
	dependabotLabelColor       = "0366d6"
	dependabotLabelDescription = "Dependency updates opened by Dependabot"
)

// LabelsResult reports the outcome of ApplyDependabotLabels.
type LabelsResult struct {
	Repo   string
	Action string   // "skipped" | "none" | "present" | "dry-run" | "created"
	Labels []string // the labels created, or that a dry run would create
	Reason string   // populated for "skipped"
}

// Summary is a one-line account of the result, for the init report and
// `dotf init github`.
func (r LabelsResult) Summary() string {
	switch r.Action {
	case "skipped":
		return "dependabot labels skipped: " + r.Reason
	case "none":
		return "no dependabot labels declared on " + r.Repo
	case "present":
		return "dependabot labels present on " + r.Repo
	case "dry-run":
		return "would create dependabot labels on " + r.Repo + ": " + strings.Join(r.Labels, ", ")
	default:
		return "created dependabot labels on " + r.Repo + ": " + strings.Join(r.Labels, ", ")
	}
}

// ApplyDependabotLabels creates every label repo's own .github/dependabot.yml
// declares and the repository lacks. Dependabot applies a custom label only if
// it exists and never creates one, so an undeclared-in-reality label silently
// drops off every Dependabot PR (#1222).
//
// It reads before it writes, so a converged repository gets no write, and it
// leaves an existing label's color and description alone. Like
// ApplyDeleteBranchOnMerge it degrades to "skipped" when gh is absent or the
// repository cannot be read; a create the server rejects is an error.
func ApplyDependabotLabels(repo string, dryRun bool) (LabelsResult, error) {
	res := LabelsResult{Repo: repo}
	if _, err := exec.LookPath("gh"); err != nil {
		res.Action, res.Reason = "skipped", "gh CLI not found (install https://cli.github.com/, then re-run)"
		return res, nil
	}

	declared, err := dependabotLabels(repo)
	if err != nil {
		res.Action, res.Reason = "skipped", err.Error()
		return res, nil
	}
	if len(declared) == 0 {
		res.Action = "none"
		return res, nil
	}

	out, err := exec.Command("gh", "label", "list", "--repo", repo, "--limit", "1000",
		"--json", "name", "--jq", ".[].name").Output()
	if err != nil {
		res.Action, res.Reason = "skipped", "could not list the labels of "+repo+" via gh"
		return res, nil
	}
	have := map[string]bool{}
	for _, name := range strings.Split(string(out), "\n") {
		have[strings.ToLower(strings.TrimSpace(name))] = true
	}
	for _, name := range declared {
		if !have[strings.ToLower(name)] {
			res.Labels = append(res.Labels, name)
		}
	}

	switch {
	case len(res.Labels) == 0:
		res.Action = "present"
	case dryRun:
		res.Action = "dry-run"
	default:
		for _, name := range res.Labels {
			if out, err := exec.Command("gh", "label", "create", name, "--repo", repo,
				"--color", dependabotLabelColor, "--description", dependabotLabelDescription).CombinedOutput(); err != nil {
				return res, fmt.Errorf("create label %q on %s, which .github/dependabot.yml declares: %w: %s",
					name, repo, err, strings.TrimSpace(string(out)))
			}
		}
		res.Action = "created"
	}
	return res, nil
}

// dependabotLabels returns the sorted, de-duplicated labels repo's
// .github/dependabot.yml (or .yaml) declares across its updates. A repository
// with neither file declares none; any other read failure is an error.
func dependabotLabels(repo string) ([]string, error) {
	for _, name := range []string{"dependabot.yml", "dependabot.yaml"} {
		cmd := exec.Command("gh", "api", "/repos/"+repo+"/contents/.github/"+name, "--jq", ".content")
		out, err := cmd.Output()
		if err != nil {
			var exit *exec.ExitError
			if errors.As(err, &exit) && strings.Contains(string(exit.Stderr), "404") {
				continue
			}
			return nil, fmt.Errorf("could not read .github/%s of %s via gh (not authenticated, offline, or repo not found)", name, repo)
		}
		raw, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(out)), ""))
		if err != nil {
			return nil, fmt.Errorf("decode .github/%s of %s: %w", name, repo, err)
		}
		var cfg struct {
			Updates []struct {
				Labels []string `yaml:"labels"`
			} `yaml:"updates"`
		}
		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse .github/%s of %s: %w", name, repo, err)
		}
		seen := map[string]bool{}
		var labels []string
		for _, u := range cfg.Updates {
			for _, l := range u.Labels {
				if l = strings.TrimSpace(l); l != "" && !seen[strings.ToLower(l)] {
					seen[strings.ToLower(l)] = true
					labels = append(labels, l)
				}
			}
		}
		sort.Strings(labels)
		return labels, nil
	}
	return nil, nil
}
