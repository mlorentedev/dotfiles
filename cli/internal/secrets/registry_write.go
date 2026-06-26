package secrets

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FlipRegistryToBW reads the registry at path, rewrites secret `id` to the bw backend
// (item/field) via SetBackendBW, and writes it back atomically — the file-level cutover
// `migrate` performs AFTER the parity gate, so a crash mid-write never corrupts the
// registry and a pre-flip failure leaves it untouched (still resolving via age).
func FlipRegistryToBW(path, id, item, field string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	out, err := SetBackendBW(data, id, item, field)
	if err != nil {
		return err
	}
	return AtomicWrite(path, out)
}

// SetBackendBW rewrites the registry block of secret `id` to the bw backend with the
// given item/field: it flips `backend:` to bw, drops the `age:` source line, and
// inserts `bw: { item: <item>, field: <field> }` — touching ONLY those lines, so every
// comment, blank line, alignment space, and other secret stays byte-for-byte.
//
// A yaml.v3 Node round-trip was rejected empirically: re-encoding the parsed document
// collapses blank lines between secrets and the alignment padding of trailing
// comments. So this is deliberate line surgery, re-validated through ParseRegistry
// before return.
//
// Scope: the single, scalar env-var case (`expose: { env: VAR }`) the bulk of the
// registry uses, in block form (`- id: x` then indented keys). Multi-var / per-var
// (dockerhub, x-twitter) and file secrets are rejected with a clear error — they need
// the multi-field migration path (#612 M3/M6). Idempotent: a secret already bw with
// the same item+field returns the input unchanged.
func SetBackendBW(data []byte, id, item, field string) ([]byte, error) {
	if item == "" || field == "" {
		return nil, fmt.Errorf("bw item and field are required")
	}
	if err := assertSingleScalarEnv(data, id); err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	start, end, dashIndent, err := secretBlock(lines, id)
	if err != nil {
		return nil, err
	}
	keyIndent := dashIndent + "  " // keys sit two columns past the "- "

	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:start+1]...) // up to and including the `- id:` line
	backendSeen := false
	for _, ln := range lines[start+1 : end] {
		trimmed := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(trimmed, "backend:"):
			out = append(out, keyIndent+"backend: bw")
			out = append(out, keyIndent+fmt.Sprintf("bw: { item: %s, field: %s }", item, field))
			backendSeen = true
		case strings.HasPrefix(trimmed, "age:"):
			// drop the age source
		case strings.HasPrefix(trimmed, "bw:"):
			if !isInlineMapping(trimmed) {
				return nil, fmt.Errorf("secret %q has a block-form bw: block this writer cannot edit", id)
			}
			// drop the old inline bw line; it is re-emitted next to backend
		default:
			out = append(out, ln)
		}
	}
	if !backendSeen {
		return nil, fmt.Errorf("secret %q has no backend: line", id)
	}
	out = append(out, lines[end:]...)

	result := []byte(strings.Join(out, "\n"))
	if _, err := ParseRegistry(result); err != nil {
		return nil, fmt.Errorf("post-edit registry invalid: %w", err)
	}
	return result, nil
}

// assertSingleScalarEnv parses the registry and confirms `id` exposes exactly one env
// var in the simple scalar form (no per-var source override, not a file) — the only
// shape SetBackendBW can flip without leaving a dead per-var age reference or
// silently collapsing many vars onto one bw field.
func assertSingleScalarEnv(data []byte, id string) error {
	reg, err := ParseRegistry(data)
	if err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}
	s := reg.Lookup(id)
	if s == nil {
		return fmt.Errorf("secret %q not found in registry", id)
	}
	if s.Expose.File != nil {
		return fmt.Errorf("secret %q is a file secret; SetBackendBW handles env secrets only", id)
	}
	if len(s.Expose.Env.Vars) != 1 || s.Expose.Env.Vars[0].Age != "" {
		return fmt.Errorf("secret %q is multi-var or per-var; use the multi-field migration path", id)
	}
	return nil
}

// secretBlock returns the [start,end) line range of secret `id`'s block and the indent
// before its `-`. The block runs from the `- id:` line through every line indented
// deeper than the dash (its keys, including multi-line expose), ending at the next line
// dedented to the dash level or shallower (the next item, a section comment, or EOF).
func secretBlock(lines []string, id string) (start, end int, dashIndent string, err error) {
	re := regexp.MustCompile(`^(\s*)-\s+id:\s*` + regexp.QuoteMeta(id) + `\s*(#.*)?$`)
	start = -1
	for i, ln := range lines {
		if m := re.FindStringSubmatch(ln); m != nil {
			start, dashIndent = i, m[1]
			break
		}
	}
	if start == -1 {
		return 0, 0, "", fmt.Errorf("secret %q not found in block form (inline-mapping secrets are unsupported)", id)
	}
	dashLevel := len(dashIndent)
	end = len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if leadingSpaces(lines[i]) <= dashLevel {
			end = i
			break
		}
	}
	return start, end, dashIndent, nil
}

// leadingSpaces counts the leading ASCII spaces of a line.
func leadingSpaces(s string) int {
	return len(s) - len(strings.TrimLeft(s, " "))
}

// isInlineMapping reports whether a `bw:` line carries its value inline (`bw: { … }`)
// rather than opening a block mapping (`bw:` with the keys on following lines).
func isInlineMapping(trimmed string) bool {
	return strings.Contains(strings.TrimPrefix(trimmed, "bw:"), "{")
}
