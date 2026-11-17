package evals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// CandidatesFile lists the models an evaluation run may launch. It is not the
// reviewer pool: a candidate is measured here before anyone decides to admit
// it, and nothing in an evaluation reads or writes the pool.
const CandidatesFile = "harness/evals/candidates.json"

// candidate is one entry as the file spells it. Only the fields a launch needs
// are accepted: a pool field such as `signs` would claim a role the file does
// not grant.
type candidate struct {
	ID       string       `json:"id"`
	Runner   string       `json:"runner"`
	Provider string       `json:"provider,omitempty"`
	Model    string       `json:"model"`
	SecretID string       `json:"secret_id,omitempty"`
	Auth     string       `json:"auth,omitempty"`
	Vendor   string       `json:"vendor"`
	Why      string       `json:"why,omitempty"`
	Credits  *CreditRates `json:"credits_per_mtok,omitempty"`
}

// CreditRates is a NaN model's credit rate per million tokens, by token type,
// as ai/nan/README.md records it.
type CreditRates struct {
	Input  float64 `json:"input"`
	Cached float64 `json:"cached"`
	Output float64 `json:"output"`
}

// Candidates is the loaded file, each entry in the launcher's shape.
type Candidates struct {
	Entries []spec.ReviewerEntry
	// Credits holds each NaN candidate's rates, by id. A candidate absent here
	// cannot be priced in credits, and a plan refuses it rather than print 0.
	Credits map[string]CreditRates
	// Envelopes is the token envelope of one run of a case kind that has never
	// run, by kind.
	Envelopes map[string]int
}

// caseKinds are the case kinds an envelope may be declared for.
var caseKinds = map[string]bool{"replay": true, "planted": true}

// LoadCandidates reads and checks repoRoot's candidates file. A missing or
// malformed file fails loudly.
func LoadCandidates(repoRoot string) (Candidates, error) {
	path := filepath.Join(repoRoot, filepath.FromSlash(CandidatesFile))
	raw, err := os.ReadFile(path)
	if err != nil {
		return Candidates{}, fmt.Errorf("read %s: %w", CandidatesFile, err)
	}
	var doc struct {
		Comment    []string       `json:"$comment,omitempty"`
		Envelopes  map[string]int `json:"default_envelope_tokens,omitempty"`
		Candidates []candidate    `json:"candidates"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return Candidates{}, fmt.Errorf("%s: %w", CandidatesFile, err)
	}
	if len(doc.Candidates) == 0 {
		return Candidates{}, fmt.Errorf("%s lists no candidates", CandidatesFile)
	}
	for kind, tokens := range doc.Envelopes {
		if !caseKinds[kind] {
			return Candidates{}, fmt.Errorf("%s: default_envelope_tokens names %q, which is not a case kind", CandidatesFile, kind)
		}
		if tokens <= 0 {
			return Candidates{}, fmt.Errorf("%s: default_envelope_tokens for %s must be positive", CandidatesFile, kind)
		}
	}
	seen := map[string]bool{}
	out := Candidates{Credits: map[string]CreditRates{}, Envelopes: doc.Envelopes}
	for _, c := range doc.Candidates {
		if err := checkCandidate(c); err != nil {
			return Candidates{}, fmt.Errorf("%s: %w", CandidatesFile, err)
		}
		if seen[c.ID] {
			return Candidates{}, fmt.Errorf("%s lists %q twice", CandidatesFile, c.ID)
		}
		seen[c.ID] = true
		if c.Credits != nil {
			out.Credits[c.ID] = *c.Credits
		}
		out.Entries = append(out.Entries, spec.ReviewerEntry{
			ID: c.ID, Runner: c.Runner, Provider: c.Provider, Model: c.Model,
			SecretID: c.SecretID, Auth: c.Auth, Vendor: c.Vendor,
		})
	}
	return out, nil
}

// checkCandidate refuses an entry the launcher would run unsafely or not at
// all. Exactly one of secret_id and `auth: login` is required: an entry with
// neither would be handed every credential in the registry.
func checkCandidate(c candidate) error {
	id := strings.TrimSpace(c.ID)
	switch {
	case id == "":
		return fmt.Errorf("a candidate has no id")
	case spec.IsEvalReviewer(id):
		return fmt.Errorf("candidate %q: an id never starts with %q; the launcher adds it", id, spec.EvalReviewerPrefix)
	case strings.TrimSpace(c.Model) == "":
		return fmt.Errorf("candidate %q has no model to pin", id)
	case strings.TrimSpace(c.Vendor) == "":
		return fmt.Errorf("candidate %q has no vendor", id)
	}
	switch c.Runner {
	case "pi":
		if strings.TrimSpace(c.Provider) == "" {
			return fmt.Errorf("candidate %q uses runner pi, which needs an explicit provider", id)
		}
	case "agy":
	default:
		return fmt.Errorf("candidate %q names runner %q; known runners: pi, agy", id, c.Runner)
	}
	hasSecret, login := strings.TrimSpace(c.SecretID) != "", strings.TrimSpace(c.Auth) == "login"
	if hasSecret == login || (c.Auth != "" && !login) {
		return fmt.Errorf("candidate %q must declare exactly one of secret_id and `auth: login`", id)
	}
	return checkCredits(id, c)
}

// checkCredits accepts rates only where they price something: a NaN model.
func checkCredits(id string, c candidate) error {
	if c.Credits == nil {
		return nil
	}
	if c.Runner != "pi" || c.Provider != "nan" {
		return fmt.Errorf("candidate %q: credits_per_mtok prices NaN models only", id)
	}
	if r := c.Credits; r.Input < 0 || r.Cached < 0 || r.Output < 0 {
		return fmt.Errorf("candidate %q: credits_per_mtok has a negative rate", id)
	}
	return nil
}

// Resolve returns the candidate with the given id.
func (cs Candidates) Resolve(id string) (spec.ReviewerEntry, error) {
	ids := make([]string, 0, len(cs.Entries))
	for _, e := range cs.Entries {
		if e.ID == id {
			return e, nil
		}
		ids = append(ids, e.ID)
	}
	return spec.ReviewerEntry{}, fmt.Errorf("%q is not in %s; the candidates are: %s", id, CandidatesFile, strings.Join(ids, ", "))
}
