// Package forge holds the GitHub-side state this repository declares in git
// and reconciles: today, branch protection (GUARD-017, #1451).
//
// Branch protection was the last per-repo forge setting applied by hand, and
// it leaves no trace in git: a required context that is dropped or renamed is
// invisible until a merge that should have been impossible (web#275). The
// declaration in forge/branch-protection.json is the SSOT; this package reads
// it, normalises the live API's shape into it, and diffs field by field.
package forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// DeclarationFile and SchemaFile are repo-relative. Not embedded: a
// declaration that silently fell back to a build-time copy would report "no
// drift" against a state nobody declared.
const (
	DeclarationFile = "forge/branch-protection.json"
	SchemaFile      = "forge/branch-protection.schema.json"
)

// Repository states a declaration may give instead of a protection object.
const (
	StateUnavailable = "unavailable" // the protection API cannot be used (private repo, free plan: 403)
	StateUnprotected = "unprotected" // deliberately no protection
)

// Declaration is forge/branch-protection.json.
type Declaration struct {
	Policy Policy              `json:"policy"`
	Repos  map[string]RepoDecl `json:"repos"`
}

// Policy is owner-wide (decision D-1 on #1625).
type Policy struct {
	RequiredApprovingReviewCount int    `json:"required_approving_review_count"`
	Rationale                    string `json:"rationale"`
}

// RepoDecl is one repository: a complete protection object, or a state.
type RepoDecl struct {
	Branch                  string      `json:"branch"`
	Protection              *Protection `json:"protection,omitempty"`
	State                   string      `json:"state,omitempty"`
	Reason                  string      `json:"reason,omitempty"`
	ApprovalsOverrideReason string      `json:"approvals_override_reason,omitempty"`
}

// Protection is the complete protection object in its normalised shape: the
// GET response with its {"enabled": bool} wrappers unwrapped, and a block the
// API omits spelled null. Every field is declared because PUT replaces the
// whole object — an undeclared field is one an apply silently clears.
type Protection struct {
	RequiredStatusChecks           *StatusChecks `json:"required_status_checks"`
	RequiredPullRequestReviews     *Reviews      `json:"required_pull_request_reviews"`
	EnforceAdmins                  bool          `json:"enforce_admins"`
	RequiredSignatures             bool          `json:"required_signatures"`
	RequiredLinearHistory          bool          `json:"required_linear_history"`
	AllowForcePushes               bool          `json:"allow_force_pushes"`
	AllowDeletions                 bool          `json:"allow_deletions"`
	BlockCreations                 bool          `json:"block_creations"`
	RequiredConversationResolution bool          `json:"required_conversation_resolution"`
	LockBranch                     bool          `json:"lock_branch"`
	AllowForkSyncing               bool          `json:"allow_fork_syncing"`
}

// StatusChecks are the required contexts, each with the app that must post it.
type StatusChecks struct {
	Strict bool    `json:"strict"`
	Checks []Check `json:"checks"`
}

// Check pins a context to its source. A nil AppID accepts any source, which
// is exactly the looseness pinning exists to remove, so it is declared, never
// defaulted.
type Check struct {
	Context string `json:"context"`
	AppID   *int   `json:"app_id"`
}

// Reviews is the pull-request requirement; nil means no pull request is
// required, so direct pushes are allowed.
type Reviews struct {
	RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
	DismissStaleReviews          bool `json:"dismiss_stale_reviews"`
	RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
	RequireLastPushApproval      bool `json:"require_last_push_approval"`
}

// Live is a normalised GET response. Restricted is reported separately
// because push restrictions are never declared for these single-owner repos:
// a live one is drift by definition.
type Live struct {
	Protection Protection
	Restricted bool
}

// Load reads and validates the declaration under repoRoot.
func Load(repoRoot string) (Declaration, error) {
	doc, err := os.ReadFile(filepath.Join(repoRoot, DeclarationFile))
	if err != nil {
		return Declaration{}, err
	}
	schema, err := os.ReadFile(filepath.Join(repoRoot, SchemaFile))
	if err != nil {
		return Declaration{}, err
	}
	if err := Validate(doc, schema); err != nil {
		return Declaration{}, err
	}
	// $schema and $comment are for editors and readers, not part of the Go
	// shape; every other unknown key is an error, as it is in the schema.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(doc, &raw); err != nil {
		return Declaration{}, err
	}
	delete(raw, "$schema")
	delete(raw, "$comment")
	stripped, err := json.Marshal(raw)
	if err != nil {
		return Declaration{}, err
	}
	var d Declaration
	dec := json.NewDecoder(bytes.NewReader(stripped))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Declaration{}, fmt.Errorf("decode %s: %w", DeclarationFile, err)
	}
	return d, nil
}

// Validate checks doc against the declaration schema (draft 2020-12, the
// same implementation the harness registries use).
func Validate(doc, schema []byte) error {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schema))
	if err != nil {
		return fmt.Errorf("parse %s: %w", SchemaFile, err)
	}
	const url = "https://mlorentedev.github.io/dotfiles/branch-protection.schema.json"
	c := jsonschema.NewCompiler()
	if err := c.AddResource(url, schemaDoc); err != nil {
		return fmt.Errorf("load %s: %w", SchemaFile, err)
	}
	compiled, err := c.Compile(url)
	if err != nil {
		return fmt.Errorf("%s is not a valid schema: %w", SchemaFile, err)
	}
	d, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("parse %s: %w", DeclarationFile, err)
	}
	if err := compiled.Validate(d); err != nil {
		return fmt.Errorf("%s does not satisfy %s: %w", DeclarationFile, SchemaFile, err)
	}
	return nil
}

type enabled struct {
	Enabled bool `json:"enabled"`
}

// liveShape is the subset of the GET response the declaration covers.
type liveShape struct {
	RequiredStatusChecks *struct {
		Strict bool    `json:"strict"`
		Checks []Check `json:"checks"`
	} `json:"required_status_checks"`
	RequiredPullRequestReviews     *Reviews        `json:"required_pull_request_reviews"`
	Restrictions                   json.RawMessage `json:"restrictions"`
	EnforceAdmins                  enabled         `json:"enforce_admins"`
	RequiredSignatures             enabled         `json:"required_signatures"`
	RequiredLinearHistory          enabled         `json:"required_linear_history"`
	AllowForcePushes               enabled         `json:"allow_force_pushes"`
	AllowDeletions                 enabled         `json:"allow_deletions"`
	BlockCreations                 enabled         `json:"block_creations"`
	RequiredConversationResolution enabled         `json:"required_conversation_resolution"`
	LockBranch                     enabled         `json:"lock_branch"`
	AllowForkSyncing               enabled         `json:"allow_fork_syncing"`
}

// Normalise turns a GET /branches/{b}/protection body into the declared shape.
func Normalise(body []byte) (Live, error) {
	var s liveShape
	if err := json.Unmarshal(body, &s); err != nil {
		return Live{}, fmt.Errorf("unexpected protection response: %w", err)
	}
	p := Protection{
		RequiredPullRequestReviews:     s.RequiredPullRequestReviews,
		EnforceAdmins:                  s.EnforceAdmins.Enabled,
		RequiredSignatures:             s.RequiredSignatures.Enabled,
		RequiredLinearHistory:          s.RequiredLinearHistory.Enabled,
		AllowForcePushes:               s.AllowForcePushes.Enabled,
		AllowDeletions:                 s.AllowDeletions.Enabled,
		BlockCreations:                 s.BlockCreations.Enabled,
		RequiredConversationResolution: s.RequiredConversationResolution.Enabled,
		LockBranch:                     s.LockBranch.Enabled,
		AllowForkSyncing:               s.AllowForkSyncing.Enabled,
	}
	if s.RequiredStatusChecks != nil {
		p.RequiredStatusChecks = &StatusChecks{Strict: s.RequiredStatusChecks.Strict, Checks: s.RequiredStatusChecks.Checks}
	}
	restricted := len(s.Restrictions) > 0 && string(s.Restrictions) != "null"
	return Live{Protection: p, Restricted: restricted}, nil
}

// Change is one field whose declared and live values differ.
type Change struct {
	Field, Declared, Live string
}

// Diff compares every declared field with live, and reports each difference
// by its dotted field path, sorted.
func Diff(declared Protection, live Live) []Change {
	want := flatten(declared)
	want["restrictions"] = "none"
	got := flatten(live.Protection)
	got["restrictions"] = "none"
	if live.Restricted {
		got["restrictions"] = "set"
	}
	fields := map[string]bool{}
	for k := range want {
		fields[k] = true
	}
	for k := range got {
		fields[k] = true
	}
	var changes []Change
	for f := range fields {
		d, l := orAbsent(want, f), orAbsent(got, f)
		if d != l {
			changes = append(changes, Change{Field: f, Declared: d, Live: l})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Field < changes[j].Field })
	return changes
}

func orAbsent(m map[string]string, k string) string {
	if v, ok := m[k]; ok {
		return v
	}
	return "(absent)"
}

// flatten renders p as dotted field → value. A nil block contributes only its
// own key ("none"), so declaring "not required" against a live block reports
// the block once rather than every sub-field.
func flatten(p Protection) map[string]string {
	b := strconv.FormatBool
	m := map[string]string{
		"enforce_admins":                   b(p.EnforceAdmins),
		"required_signatures":              b(p.RequiredSignatures),
		"required_linear_history":          b(p.RequiredLinearHistory),
		"allow_force_pushes":               b(p.AllowForcePushes),
		"allow_deletions":                  b(p.AllowDeletions),
		"block_creations":                  b(p.BlockCreations),
		"required_conversation_resolution": b(p.RequiredConversationResolution),
		"lock_branch":                      b(p.LockBranch),
		"allow_fork_syncing":               b(p.AllowForkSyncing),
		"required_status_checks":           "none",
		"required_pull_request_reviews":    "none",
	}
	if sc := p.RequiredStatusChecks; sc != nil {
		m["required_status_checks"] = "required"
		m["required_status_checks.strict"] = b(sc.Strict)
		m["required_status_checks.checks"] = checkList(sc.Checks)
	}
	if r := p.RequiredPullRequestReviews; r != nil {
		m["required_pull_request_reviews"] = "required"
		m["required_pull_request_reviews.required_approving_review_count"] = strconv.Itoa(r.RequiredApprovingReviewCount)
		m["required_pull_request_reviews.dismiss_stale_reviews"] = b(r.DismissStaleReviews)
		m["required_pull_request_reviews.require_code_owner_reviews"] = b(r.RequireCodeOwnerReviews)
		m["required_pull_request_reviews.require_last_push_approval"] = b(r.RequireLastPushApproval)
	}
	return m
}

// checkList renders checks as a sorted "context@app" list; order is not a
// requirement, and "any" marks a context no app is pinned to.
func checkList(checks []Check) string {
	parts := make([]string, 0, len(checks))
	for _, c := range checks {
		app := "any"
		if c.AppID != nil {
			app = strconv.Itoa(*c.AppID)
		}
		parts = append(parts, c.Context+"@"+app)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// marshalProtection renders p in the declared JSON shape.
func marshalProtection(p Protection) ([]byte, error) { return json.Marshal(p) }
