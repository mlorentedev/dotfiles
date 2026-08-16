package secrets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// rawBodyCap bounds what --raw will echo. A non-2xx body is unbounded input from
// a component that is, by definition, misbehaving: an HTML error page or a Node
// stacktrace pasted whole buries the one line that matters.
const rawBodyCap = 512

// ValueFact is what may safely be said about a string found in a response: where
// it was, how long it is, and a fingerprint. Never the value.
//
// Path is the dotted JSON location ("data.fields[0].value"), which is the fact
// that actually diagnoses a mapping bug -- #990 existed because a token lived at
// a different field than the registry declared, and no amount of looking at the
// value would have said so.
type ValueFact struct {
	Path        string
	Length      int
	Fingerprint string
}

// ProbeReport is the printable, secret-free summary of one probe.
//
// Every field here is either transport metadata or derived from content by a
// non-reversible function. There is deliberately no field that can hold a value:
// the safety property is structural, not a matter of remembering to redact at
// print time. That distinction is the entire reason this type exists -- the rule
// it replaces was written down, correct, and violated three times in one day
// (CLI-038, #1012).
type ProbeReport struct {
	Status      int
	ContentType string
	Size        int
	ValidJSON   bool
	Success     bool   // the envelope's own success flag, when there is an envelope
	Message     string // the envelope's message; daemon-authored, not user content
	Values      []ValueFact
	RawBody     string // non-2xx only, capped; empty otherwise
	Truncated   bool
}

// ShapeProbe converts a raw response into the report.
//
// raw admits the body ONLY for a non-2xx status. A 2xx from /object/item/<id> is
// the credential itself, so there is no flag, no verbosity level and no debug
// mode that prints it -- the check lives here rather than at the call site so no
// future caller can opt out of it.
func ShapeProbe(res ProbeResult, raw bool) ProbeReport {
	rep := ProbeReport{
		Status:      res.Status,
		ContentType: res.ContentType,
		Size:        res.Size,
		ValidJSON:   json.Valid(res.Body),
	}

	if raw && !Is2xx(res.Status) {
		body := string(res.Body)
		if len(body) > rawBodyCap {
			body, rep.Truncated = body[:rawBodyCap], true
		}
		rep.RawBody = body
	}

	if !rep.ValidJSON {
		return rep
	}

	var env struct {
		Success bool            `json:"success"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		return rep
	}
	rep.Success, rep.Message = env.Success, env.Message

	var data any
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return rep
	}
	collectValues("data", data, &rep.Values)
	sort.Slice(rep.Values, func(i, j int) bool { return rep.Values[i].Path < rep.Values[j].Path })
	return rep
}

// ProbeItemID resolves a vault item NAME to the id the item endpoint takes,
// using the same search the read path uses so a probe addresses exactly the item
// the resolver would.
//
// It goes through Probe rather than call for one reason: the search step can fail
// the same way the item read does, and routing it through the safe path means a
// failure there is reported as a status rather than as a parser error. The list
// reply carries full items — values included — so it is parsed in memory and
// never rendered; only the id leaves this function.
func (c BWServeClient) ProbeItemID(item string) (string, error) {
	res, err := c.Probe(http.MethodGet, "/list/object/items?search="+url.QueryEscape(item))
	if err != nil {
		return "", err
	}
	if !Is2xx(res.Status) {
		return "", fmt.Errorf("listing items for %q: HTTP %d (%d bytes)", item, res.Status, res.Size)
	}
	var env struct {
		Data bwServeListData `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		return "", fmt.Errorf("listing items for %q: HTTP %d returned no parseable list (%d bytes)",
			item, res.Status, res.Size)
	}
	var ids []string
	for _, it := range env.Data.Data {
		if it.Name == item || it.ID == item {
			ids = append(ids, it.ID)
		}
	}
	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		return "", fmt.Errorf("%w: bw serve item %q: not found", ErrBWItemNotFound, item)
	default:
		// Same ambiguity the read path refuses, refused here for the same reason:
		// probing an arbitrary one of them would report on an item the resolver
		// might not pick.
		return "", fmt.Errorf("bw serve item %q: More than one result was found", item)
	}
}

// collectValues walks the decoded payload and records every string leaf.
//
// It walks generically rather than reading Bitwarden's item schema on purpose.
// A schema-aware version has to enumerate where secrets live -- fields[].value,
// login.password, notes, and whatever a future item type adds -- and the failure
// mode of missing one is that a credential is treated as safe metadata. Treating
// EVERY string as value-bearing cannot fail that way, and costs only a few extra
// lines of harmless output (an item's name and id are strings too).
func collectValues(path string, v any, out *[]ValueFact) {
	switch t := v.(type) {
	case string:
		*out = append(*out, ValueFact{Path: path, Length: len(t), Fingerprint: Fingerprint(t)})
	case map[string]any:
		// One narrow exception to "every string is value-bearing": Bitwarden's
		// custom fields are {"name": label, "value": secret} pairs, and the label
		// is the single most diagnostic fact in the whole payload. #990 existed
		// because a token lived in a different field than the registry declared —
		// findable by reading field names, invisible from paths alone, since
		// `data.fields[0].value` says nothing about WHICH field that is.
		//
		// Scoped as tightly as it can be: only a key literally named "name", only
		// when the same object also carries a "value", and only to LABEL that
		// object. The label is a user-chosen identifier, not a credential; the
		// value beside it is still fingerprinted like everything else.
		//
		// The assumption, stated because it is domain knowledge rather than
		// something this code verifies: a Bitwarden custom-field NAME is a
		// user-chosen label, not credential material. It is printed verbatim, so
		// an operator who names a field "gitlab-token-2026-08" sees that string.
		// That is a label leak at worst, never a value leak — and validating
		// labels heuristically would trade a certain loss of diagnostic power for
		// a guess about what looks secret.
		if label, ok := fieldLabel(t); ok {
			collectValues(fmt.Sprintf("%s[%s]", path, label), t["value"], out)
			for k, child := range t {
				if k == "name" || k == "value" {
					continue
				}
				collectValues(path+"."+k, child, out)
			}
			return
		}
		for k, child := range t {
			collectValues(path+"."+k, child, out)
		}
	case []any:
		for i, child := range t {
			collectValues(fmt.Sprintf("%s[%d]", path, i), child, out)
		}
	case nil:
		// An explicit null is a real state — a cleared field — and reads
		// differently from an absent key. Named so it is not mistaken for either.
		*out = append(*out, ValueFact{Path: path, Length: 0, Fingerprint: "(null)"})
	default:
		// Numbers and booleans. They cannot be credentials, but silently dropping
		// them makes the field INVISIBLE, which is a diagnostic gap rather than a
		// safety one: a custom field {"name":"enabled","value":true} would vanish
		// from a report whose whole job is to describe the payload's shape. The
		// type is reported, never the value, so the rule "no content leaves here"
		// still holds without exception.
		*out = append(*out, ValueFact{
			Path: path, Length: 0, Fingerprint: fmt.Sprintf("(%T)", t),
		})
	}
}

// fieldLabel reports the label of a Bitwarden {"name": ..., "value": ...} pair.
// Both keys must be present and the name must be a non-empty string, so an
// ordinary object that merely happens to have a "name" (the item itself has one)
// is not mistaken for a custom field.
func fieldLabel(m map[string]any) (string, bool) {
	if _, hasValue := m["value"]; !hasValue {
		return "", false
	}
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

// Is2xx reports whether a status is a success. Exported so the command layer
// asks the SAME question the reporting layer answers with — a second private
// copy is how two callers drift into disagreeing about which bodies are safe.
func Is2xx(status int) bool { return status >= 200 && status < 300 }

// String renders the report for a terminal.
func (r ProbeReport) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "HTTP %d  %s  %d bytes\n", r.Status, orDash(r.ContentType), r.Size)
	if !r.ValidJSON {
		// Named plainly, because the absence of this sentence is what made #988
		// unreadable: the client reported `invalid character 'I'`, which describes
		// a parser's disappointment rather than what the daemon did.
		b.WriteString("body: not JSON\n")
	} else {
		fmt.Fprintf(&b, "envelope: success=%v", r.Success)
		if r.Message != "" {
			fmt.Fprintf(&b, " message=%q", r.Message)
		}
		b.WriteString("\n")
	}

	if len(r.Values) > 0 {
		b.WriteString("values (never printed — length and fingerprint only):\n")
		for _, v := range r.Values {
			fmt.Fprintf(&b, "  %-32s len=%-5d %s\n", v.Path, v.Length, v.Fingerprint)
		}
	}

	if r.RawBody != "" {
		b.WriteString("body:\n  " + strings.ReplaceAll(r.RawBody, "\n", "\n  ") + "\n")
		if r.Truncated {
			fmt.Fprintf(&b, "  ... truncated at %d bytes\n", rawBodyCap)
		}
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
