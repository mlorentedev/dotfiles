package prreview

import (
	_ "embed"
	"fmt"
	"os"

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
