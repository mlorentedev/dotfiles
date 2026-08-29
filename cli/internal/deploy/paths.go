package deploy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Path-rendering forms an entry may declare (AI-042, #1334). A config that
// carries filesystem paths for a tool — Copilot's trustedFolders, agy's
// trustedWorkspaces — cannot hardcode a home directory and still be the same
// file on every machine; it carries {HOME}/Projects/* and the deploy renders
// it. Which separator the rendered path uses is DECLARED per entry, because it
// is a fact about the tool that reads the file, measured, not a property of the
// OS: Copilot on Windows wrote backslash entries into its own config when the
// user trusted folders there; agy has read forward-slash entries on the same
// box daily. Declaring it keeps the evidence next to the choice.
const (
	PathsNative = "native" // filepath.FromSlash — the OS's own separator
	PathsSlash  = "slash"  // filepath.ToSlash — forward slashes everywhere
)

// expandPaths resolves the manifest's {VAR} tokens inside every JSON string
// value of src, renders each string that carried a token in the declared form,
// and re-encodes the document. JSON-aware on purpose: a native Windows path
// contains backslashes, and the encoder escapes them; a textual substitution
// would have to, by hand, and get it wrong once. Strings without a token are
// returned as they were. Key order follows the encoder (sorted); the deploy
// compares rendered content against the destination, so the order is stable
// run to run.
//
// Two rules the review of AI-042 added (round 2, agy/gemini-3.1-pro-high):
//
//   - Numbers are decoded with UseNumber and re-encoded verbatim. A plain
//     Unmarshal into `any` makes every number a float64, and an integer above
//     2^53 — an ID, a millisecond timestamp — comes back rounded, silently,
//     in a file that declared nothing about numbers.
//   - The separator form applies only to a string that BEGINS with a token.
//     That is what "this string is a path" looks like in a manifest source
//     ({HOME}/Projects/*); a token inside a URL (https://host/{VAR}/x) is not a
//     path, and converting its slashes under `native` would corrupt it.
func expandPaths(src []byte, form, home string, resolve func(string) string) ([]byte, error) {
	if form != PathsNative && form != PathsSlash {
		return nil, fmt.Errorf("unknown paths form %q (want %s or %s)", form, PathsNative, PathsSlash)
	}
	doc, err := decodeJSONNumbers(src)
	if err != nil {
		return nil, fmt.Errorf("paths rendering needs a JSON source: %w", err)
	}
	var bad []string
	render := func(s string) string {
		if !tokenRe.MatchString(s) {
			return s
		}
		isPath := tokenRe.FindStringIndex(s)[0] == 0
		expanded := tokenRe.ReplaceAllStringFunc(s, func(tok string) string {
			name := tok[1 : len(tok)-1]
			if name == "HOME" {
				return home
			}
			if v := resolve(name); v != "" {
				return v
			}
			bad = append(bad, name)
			return tok
		})
		if !isPath {
			return expanded
		}
		if form == PathsNative {
			return filepath.FromSlash(expanded)
		}
		return filepath.ToSlash(expanded)
	}
	doc = walkStrings(doc, render)
	if len(bad) > 0 {
		return nil, fmt.Errorf("unresolvable path variable(s) %s", strings.Join(bad, ", "))
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// decodeJSONNumbers decodes one JSON document into `any` keeping every number
// as json.Number, so re-encoding writes the digits that were read. The
// default decoding into float64 rounds integers above 2^53; a rendered config
// must not change a value it never declared an interest in.
func decodeJSONNumbers(src []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// decodeJSONObject is decodeJSONNumbers for a document that must be an object
// (or `null`, which yields a nil map the caller checks, as Unmarshal did).
func decodeJSONObject(src []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	var obj map[string]any
	if err := dec.Decode(&obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// walkStrings applies fn to every string value in a decoded JSON document.
func walkStrings(v any, fn func(string) string) any {
	switch x := v.(type) {
	case string:
		return fn(x)
	case []any:
		for i := range x {
			x[i] = walkStrings(x[i], fn)
		}
		return x
	case map[string]any:
		for k := range x {
			x[k] = walkStrings(x[k], fn)
		}
		return x
	default:
		return v
	}
}
