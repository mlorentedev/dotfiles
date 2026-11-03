package prreview

import (
	_ "embed"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/spec"
)

// EmbeddedSource is the --pool value that names the pool built into dotf.
const EmbeddedSource = "embedded"

// embeddedPool is a build-time copy of harness/reviewer-pool.json: go:embed
// cannot reach outside this package's directory. TestEmbeddedPoolEqualsTheRepoPool
// holds the two equal.
//
//go:embed reviewer-pool.json
var embeddedPool []byte

// LoadPool reads the pool the caller names: EmbeddedSource, or a file path.
//
// There is no fallback from a file that cannot be read to the embedded copy,
// for the reason harness/model_map.go gives: a pool that is absent would become
// a build-time default and nothing would say so. The source stays in the
// result's summary, so a reader sees which pool drew.
func LoadPool(source string) (spec.ReviewerPool, error) {
	raw, name := embeddedPool, "the embedded pool"
	if source != EmbeddedSource {
		var err error
		if raw, err = os.ReadFile(source); err != nil {
			return spec.ReviewerPool{}, fmt.Errorf("cannot read a pool from %q: %w", source, err)
		}
		name = source
	}
	p, err := spec.ParseReviewerPool(raw, name)
	if err != nil {
		return spec.ReviewerPool{}, fmt.Errorf("cannot read a pool from %q: %w", source, err)
	}
	return p, nil
}

// NaNChain is the pool's NaN models for PR-Agent, the chain the preflight probes:
// every `openai/` member, by descending weight, pool order breaking a tie. The
// heaviest member is the primary, and a model two members share appears once.
// Document order alone will not do, because the
// pool's own comment says its order is provenance, not precedence.
func NaNChain(p spec.ReviewerPool) []string {
	var members []spec.PRAgentBlock
	for _, e := range p.Pool {
		if e.PRAgent != nil && strings.HasPrefix(e.PRAgent.Model, "openai/") {
			members = append(members, *e.PRAgent)
		}
	}
	slices.SortStableFunc(members, func(a, b spec.PRAgentBlock) int { return b.Weight - a.Weight })
	chain := make([]string, 0, len(members))
	for _, m := range members {
		if !slices.Contains(chain, m.Model) {
			chain = append(chain, m.Model)
		}
	}
	return chain
}
