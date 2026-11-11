package fleetlint

import (
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The rule ids are a contract: exceptions name them, so a rename is a
// breaking change for every repository that declares one.
const (
	RuleTimeout            = "fleet/timeout"
	RuleSHAPin             = "fleet/sha-pin"
	RulePersistCredentials = "fleet/persist-credentials"
	RuleNoSecretsInherit   = "fleet/no-secrets-inherit"
)

var shaRef = regexp.MustCompile(`@[0-9a-f]{40}$`)

// lintWorkflow applies every rule to one workflow file.
func lintWorkflow(path string, data []byte) ([]Finding, error) {
	docs, err := decodeDocs(data)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, d := range docs {
		if len(d.Content) == 0 {
			continue
		}
		jobs := mapValue(d.Content[0], "jobs")
		if jobs == nil || jobs.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(jobs.Content); i += 2 {
			out = append(out, lintJob(path, jobs.Content[i], jobs.Content[i+1])...)
		}
	}
	return out, nil
}

func lintJob(path string, key, job *yaml.Node) []Finding {
	var out []Finding
	add := func(rule string, line int, msg string) {
		out = append(out, Finding{Rule: rule, Path: path, Line: line, Message: msg})
	}
	uses := mapValue(job, "uses")
	// A job that calls a reusable workflow cannot set timeout-minutes: the
	// called workflow's jobs carry their own.
	if uses == nil && mapValue(job, "timeout-minutes") == nil {
		add(RuleTimeout, key.Line, "job "+key.Value+" sets no timeout-minutes")
	}
	if uses != nil && !pinned(uses.Value) {
		add(RuleSHAPin, uses.Line, uses.Value+" is not pinned by a full commit SHA")
	}
	if s := mapValue(job, "secrets"); s != nil && s.Kind == yaml.ScalarNode && s.Value == "inherit" {
		add(RuleNoSecretsInherit, s.Line, "job "+key.Value+" passes every secret with secrets: inherit; name the ones it needs")
	}
	steps := mapValue(job, "steps")
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return out
	}
	for _, st := range steps.Content {
		su := mapValue(st, "uses")
		if su == nil {
			continue
		}
		if !pinned(su.Value) {
			add(RuleSHAPin, su.Line, su.Value+" is not pinned by a full commit SHA")
		}
		if strings.HasPrefix(su.Value, "actions/checkout@") {
			pc := mapValue(mapValue(st, "with"), "persist-credentials")
			if pc == nil || pc.Value != "false" {
				add(RulePersistCredentials, su.Line, "checkout keeps the token in .git/config; set persist-credentials: false")
			}
		}
	}
	return out
}

// pinned reports whether a uses: reference is immutable. A local action or
// workflow (./) is part of the commit itself; a docker:// image is out of
// scope here, its digest pin being a registry rule, not a git one.
func pinned(ref string) bool {
	if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
		return true
	}
	return shaRef.MatchString(ref)
}
