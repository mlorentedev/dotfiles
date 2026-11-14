// Package evals holds the model evaluation corpus and its deterministic grader
// (ADR-063). A case is one JSON file validated by case.schema.json; the grader
// matches a reviewer's findings table against the case's anchor-encoded key.
//
// This package may import internal/spec; internal/spec must never import it.
// Slice 2 adds the eval slot to spec's review launcher, and the dependency has
// to point one way for that slot to stay an internal API.
package evals

import (
	"bytes"
	_ "embed"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// CaseSchemaFile is the repo-relative path of the case schema.
const CaseSchemaFile = "harness/evals/case.schema.json"

// The embedded schema is a build-time copy of harness/evals/case.schema.json:
// an embed directive cannot reach harness/ from here, and `dotf eval check` must validate
// cases in any repository at the released version without vendoring a copy.
// TestEmbeddedCaseSchemaEqualsTheRepoSchema holds the two equal.
//
//go:embed case.schema.json
var embeddedCaseSchema []byte

var (
	caseSchemaOnce sync.Once
	caseSchema     *jsonschema.Schema
	caseSchemaErr  error
)

func compiledCaseSchema() (*jsonschema.Schema, error) {
	caseSchemaOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(embeddedCaseSchema))
		if err != nil {
			caseSchemaErr = fmt.Errorf("parse %s: %w", CaseSchemaFile, err)
			return
		}
		// A stable resource id, so errors never carry a machine-specific path.
		const url = "https://mlorentedev.github.io/dotfiles/evals/case.schema.json"
		c := jsonschema.NewCompiler()
		if err := c.AddResource(url, doc); err != nil {
			caseSchemaErr = fmt.Errorf("load %s: %w", CaseSchemaFile, err)
			return
		}
		caseSchema, caseSchemaErr = c.Compile(url)
	})
	return caseSchema, caseSchemaErr
}

// ValidateCase checks one case document against the case schema.
func ValidateCase(doc []byte) error {
	schema, err := compiledCaseSchema()
	if err != nil {
		return err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(doc))
	if err != nil {
		return fmt.Errorf("parse case: %w", err)
	}
	if err := schema.Validate(inst); err != nil {
		return fmt.Errorf("case does not match %s: %w", CaseSchemaFile, err)
	}
	return nil
}
