package secrets

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// FlipRegistryToBW reads the registry at path, activates secret `id`'s bw backend via
// SetBackendBW, and writes it back atomically — the file-level cutover `migrate` performs
// AFTER the parity gate, so a crash mid-write never corrupts the registry and a pre-flip
// failure leaves it untouched (still resolving via age).
func FlipRegistryToBW(path, id string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}
	out, err := SetBackendBW(data, id)
	if err != nil {
		return err
	}
	return AtomicWrite(path, out)
}

// SetBackendBW activates the pre-declared Bitwarden backend of secret `id`: it flips the
// secret's `backend:` value to bw and drops the now-dead `age:` source line, leaving the
// already-declared `bw: { item, field }` block — and every comment, blank line, alignment
// space, and other secret — byte-for-byte in place.
//
// The env-var-per-entry registry pre-declares `bw:` up front (dormant while backend is
// age — ADR-028 §2 addendum), so the flip neither computes nor relocates the target. That
// is deliberate: the declared block is the SINGLE source for both `migrate`'s parity write
// and the post-flip resolution, so the value parity verified and the value resolved after
// the cutover are pinned to the same item/field, with no chance of drift. That is also why
// this writer takes no item/field — they would be a second, divergeable copy of the SSOT.
//
// A yaml.v3 Node round-trip was rejected empirically: re-encoding the parsed document
// collapses blank lines between secrets and the alignment padding of trailing comments.
// So this is deliberate line surgery, re-validated through ParseRegistry before return.
//
// Scope: the single, scalar env-var case (`expose: { env: VAR }`) the bulk of the registry
// uses, in block form (`- id: x` then indented keys). The secret MUST carry a `bw:` target
// to activate; multi-var / per-var (dockerhub, x-twitter) and file secrets are rejected
// with a clear error — they need the multi-field migration path (#612 M3/M6). Idempotent:
// an already-bw secret (backend bw, no age line) returns the input unchanged.
func SetBackendBW(data []byte, id string) ([]byte, error) {
	if err := assertMigratable(data, id); err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	start, end, dashIndent, err := secretBlock(lines, id)
	if err != nil {
		return nil, err
	}
	keyIndent := dashIndent + "  " // keys sit two columns past the "- "

	out := make([]string, 0, len(lines))
	out = append(out, lines[:start+1]...) // up to and including the `- id:` line
	backendSeen := false
	for _, ln := range lines[start+1 : end] {
		trimmed := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(trimmed, "backend:"):
			out = append(out, keyIndent+"backend: bw")
			backendSeen = true
		case strings.HasPrefix(trimmed, "age:"):
			// drop the dead age source — the value now lives in Bitwarden
		default:
			out = append(out, ln) // keep everything else verbatim — crucially the declared bw:
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

// assertMigratable parses the registry and confirms `id` is the shape SetBackendBW can
// flip: exactly one env var in the simple scalar form (no per-var source override, not a
// file) WITH a declared `bw:` target to activate. The `migrate` command guards these too,
// but the writer fails loud on its own so a stray caller can never half-flip a secret —
// leaving it `backend: bw` with no bw target, hence unresolvable.
func assertMigratable(data []byte, id string) error {
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
	if s.BW == nil || s.BW.Item == "" {
		return fmt.Errorf("secret %q has no bw: target to activate — declare it in the registry first", id)
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
