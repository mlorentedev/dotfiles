// Package gate holds the pull-request gates CI runs as dotf commands
// (CI-018): the knowledge gate and, later, the spec gate. They read the pull
// request from an event file and make no API call, so the same binary judges a
// pull request on GitHub and on Gitea (mlorentedev/ci-workflows CI-008).
package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// PR is what the gates read from a pull-request event.
type PR struct {
	Number int
	Body   string
	Labels []string
	Author string
}

// ErrNoPR is an event that carries no pull request: the gates refuse to judge
// it, since passing would certify a change nobody described.
var ErrNoPR = errors.New("the event carries no pull_request")

type eventFile struct {
	PullRequest *struct {
		Number int     `json:"number"`
		Body   *string `json:"body"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
}

// ReadEvent reads a pull-request event file. A null body is an empty one, as
// GitHub sends a pull request opened without a description.
func ReadEvent(file string) (PR, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return PR{}, fmt.Errorf("read event: %w", err)
	}
	var ev eventFile
	if err := json.Unmarshal(raw, &ev); err != nil {
		return PR{}, fmt.Errorf("parse event %s: %w", file, err)
	}
	if ev.PullRequest == nil {
		return PR{}, ErrNoPR
	}
	pr := PR{Number: ev.PullRequest.Number, Author: ev.PullRequest.User.Login}
	if ev.PullRequest.Body != nil {
		pr.Body = *ev.PullRequest.Body
	}
	for _, l := range ev.PullRequest.Labels {
		pr.Labels = append(pr.Labels, l.Name)
	}
	return pr, nil
}

// HasLabel reports whether the pull request carries the label, exactly.
func (p PR) HasLabel(name string) bool {
	for _, l := range p.Labels {
		if l == name {
			return true
		}
	}
	return false
}

// dependencyBots are the authors whose "dependencies"-labelled pull requests
// skip the gates. tests/check-knowledge-gate.bats holds this list equal to the
// shell gates' while both exist.
var dependencyBots = []string{"dependabot[bot]", "dependabot-preview[bot]", "renovate[bot]"}

// IsDependencyBotPR is a dependency bot's pull request with the label. A human
// who adds the label is still judged.
func (p PR) IsDependencyBotPR() bool {
	if !p.HasLabel("dependencies") {
		return false
	}
	for _, b := range dependencyBots {
		if p.Author == b {
			return true
		}
	}
	return false
}
