package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
// value of src, renders them in the declared form, and re-encodes the
// document. JSON-aware on purpose: a native Windows path contains backslashes,
// and the encoder escapes them; a textual substitution would have to, by hand,
// and get it wrong once. Strings without a token are returned as they were.
// Key order follows the encoder (sorted); the deploy compares rendered content
// against the destination, so the order is stable run to run.
//
// THE RULE, symmetric in both forms (AI-042 review rounds 2 and 3):
//
//   - every token's expansion is rendered in the declared form, wherever the
//     token sits — {HOME} inside https://host/{HOME}/x becomes C:/Users/u
//     under slash and C:\Users\u under native, and the rest of the string is
//     untouched;
//   - a string that BEGINS with a token is a path, and the whole string is
//     rendered in the declared form as well ({HOME}/Projects/* → one
//     separator throughout). A token elsewhere never triggers that: a URL's
//     own slashes are not ours to convert.
//
// Numbers are decoded with UseNumber and re-encoded verbatim (a plain decode
// into `any` rounds integers above 2^53); a second JSON document after the
// first is refused, as the manifest reader refuses it; and `<`, `>`, `&` are
// written as themselves, so the rendered bytes are what the tool would write.
func expandPaths(src []byte, form, home string, resolve func(string) string) ([]byte, error) {
	if form != PathsNative && form != PathsSlash {
		return nil, fmt.Errorf("unknown paths form %q (want %s or %s)", form, PathsNative, PathsSlash)
	}
	doc, err := decodeJSONNumbers(src)
	if err != nil {
		return nil, fmt.Errorf("paths rendering needs a JSON source: %w", err)
	}
	shape := filepath.ToSlash
	if form == PathsNative {
		shape = filepath.FromSlash
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
				return shape(home)
			}
			if v := resolve(name); v != "" {
				return shape(v)
			}
			bad = append(bad, name)
			return tok
		})
		if isPath {
			return shape(expanded)
		}
		return expanded
	}
	doc = walkStrings(doc, render)
	if len(bad) > 0 {
		return nil, fmt.Errorf("unresolvable path variable(s) %s", strings.Join(bad, ", "))
	}
	return encodeJSON(doc)
}

// decodeJSONNumbers decodes ONE JSON document into `any`, keeping every number
// as json.Number so re-encoding writes the digits that were read. Anything
// after the document is an error, the same refusal decodeManifest makes: a
// decoder that stops at the first closing brace would render `{"a":1}
// {"injected":true}` as `{"a":1}` and call the rest nothing.
func decodeJSONNumbers(src []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON document (dotf reads one document; remove what follows it)")
	}
	return doc, nil
}

// decodeJSONObject is decodeJSONNumbers for a document that must be an object
// (or `null`, which yields a nil map the caller checks, as Unmarshal did).
func decodeJSONObject(src []byte) (map[string]any, error) {
	doc, err := decodeJSONNumbers(src)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}
	obj, ok := doc.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("json: cannot unmarshal %T into Go value of type map[string]interface {}", doc)
	}
	return obj, nil
}

// encodeJSON writes v indented, sorted by key, with `<`, `>` and `&` as
// themselves and a trailing newline — the bytes a tool would write for the
// same content, so an in-sync comparison against a tool-written file is fair.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil // Encode already ends with '\n'
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
