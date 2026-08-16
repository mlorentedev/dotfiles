package secrets

import (
	"encoding/json"
	"fmt"
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

	if raw && !is2xx(res.Status) {
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

func is2xx(status int) bool { return status >= 200 && status < 300 }

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
