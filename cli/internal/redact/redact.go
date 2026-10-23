// Package redact masks injected secret values in a stream before it reaches a
// transcript (ADR-028: never dump a secrets store to standard output). It is
// shared by every command that hands a secret to a child or a guest, so the
// guarantee is one implementation, not a copy per caller.
package redact

import (
	"bytes"
	"io"
	"sort"
	"strings"
	"sync"
)

// Writer intercepts output emitted by a process and replaces any byte sequence
// corresponding to an injected secret value (len >= 6) with [REDACTED:<KEY>].
// Enforces ADR-028 doctrine: "Never dump a secrets store to standard output".
type Writer struct {
	target       io.Writer
	pairs        [][2][]byte
	tail         []byte
	maxSecretLen int
	mu           sync.Mutex
}

// New returns a Writer over target that replaces the value of every KEY=value
// pair in injected with [REDACTED:KEY].
func New(target io.Writer, injected []string) *Writer {
	var pairs [][2][]byte
	maxLen := 0
	for _, kv := range injected {
		key, val, ok := strings.Cut(kv, "=")
		if ok && len(val) >= 6 {
			valBytes := []byte(val)
			repBytes := []byte("[REDACTED:" + key + "]")
			pairs = append(pairs, [2][]byte{valBytes, repBytes})
			if len(valBytes) > maxLen {
				maxLen = len(valBytes)
			}
		}
	}
	// Longest secret first. If one injected value is a prefix of another
	// (`abc` and `abcdef`), replacing the shorter one first breaks the longer
	// one's match and leaks its suffix. Vanishingly unlikely with random API
	// keys and pre-existing, but it is a hole in redaction and the order is
	// free to fix. Surfaced by the SEC-002 adversarial review.
	sort.SliceStable(pairs, func(i, j int) bool {
		return len(pairs[i][0]) > len(pairs[j][0])
	})

	return &Writer{
		target:       target,
		pairs:        pairs,
		maxSecretLen: maxLen,
	}
}

func (r *Writer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pairs) == 0 {
		return r.target.Write(p)
	}

	data := p
	if len(r.tail) > 0 {
		data = append(r.tail, p...)
		r.tail = nil
	}

	for _, pair := range r.pairs {
		data = bytes.ReplaceAll(data, pair[0], pair[1])
	}

	// Withhold ONLY a trailing run that could still grow into a secret, never a
	// fixed window. The previous rule held back maxSecretLen-1 bytes on every
	// write regardless of content, which is invisible when the consumer is a
	// pipe drained at Flush() and fatal when it is a terminal: with a 64-byte
	// key, the last 63 bytes of every frame -- typically the cursor
	// positioning -- stayed dark until more output happened to arrive. A TUI
	// driven that way renders late and misplaced (SEC-002).
	if hold, _ := r.holdBack(data, anyPrefix); hold > 0 {
		split := len(data) - hold
		r.tail = make([]byte, hold)
		copy(r.tail, data[split:])
		_, err := r.target.Write(data[:split])
		return len(p), err
	}

	r.tail = nil
	_, err := r.target.Write(data)
	return len(p), err
}

// holdBack returns how many trailing bytes of data must be withheld because
// they are a PROPER prefix of some secret -- i.e. bytes that are not a leak
// yet but would be if the next write completed them. Zero in the common case,
// so a frame ending in ordinary output goes out whole and immediately.
//
// Only a proper prefix counts: a suffix that already contains a whole secret
// was replaced by the ReplaceAll pass above and is no longer present to match.
// A prefix counts only when it is at least floor(len(secret)) bytes long.
func (r *Writer) holdBack(data []byte, floor func(secretLen int) int) (int, []byte) {
	maxHold := r.maxSecretLen - 1
	if maxHold > len(data) {
		maxHold = len(data)
	}
	// Longest first: the held run must cover every secret it could complete,
	// and a shorter match would release bytes that a longer one still needs.
	for n := maxHold; n > 0; n-- {
		suffix := data[len(data)-n:]
		for _, pair := range r.pairs {
			if len(suffix) < len(pair[0]) && n >= floor(len(pair[0])) && bytes.HasPrefix(pair[0], suffix) {
				return n, pair[1]
			}
		}
	}
	return 0, nil
}

// anyPrefix is Write's floor: while the stream is open, every proper prefix is
// held, however short, because the next write may complete it.
func anyPrefix(int) int { return 1 }

// materialPrefix is Flush's floor: the stream has ended, so a held prefix can
// no longer complete, and only one long enough to be credential material is
// redacted. That is 6 bytes, the floor below which a value is not treated as a
// secret at all, or half the secret when that is shorter, so a short secret
// never prints most of itself. Below it the bytes are a letter or a public
// format prefix like `sk-`, and redacting them rewrote ordinary output:
// `results` printed as `result[REDACTED:KEY]` for a key starting with `s`.
func materialPrefix(secretLen int) int { return min(6, (secretLen+1)/2) }

// Flush writes out what the hold-back still withholds, redacting a held prefix
// that is credential material. Call it when the stream ends.
func (r *Writer) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.tail) == 0 {
		return nil
	}
	for _, pair := range r.pairs {
		r.tail = bytes.ReplaceAll(r.tail, pair[0], pair[1])
	}

	// What remains may still be a PROPER PREFIX of a secret -- the stream ended
	// before the child could finish writing the value, so the ReplaceAll above
	// found nothing to match. Those bytes are credential material: up to
	// maxSecretLen-1 leading characters of a key. Writing them out raw is the
	// one hole the hold-back leaves, and it leaves it at exactly the moment the
	// transcript is closed. Replace them with the placeholder of the secret they
	// begin, which is the honest report: material from that key was here.
	// Surfaced by the SEC-002 adversarial review. A prefix too short to be
	// material is ordinary output and is written as it is (materialPrefix).
	if n, rep := r.holdBack(r.tail, materialPrefix); n > 0 {
		r.tail = append(r.tail[:len(r.tail)-n], rep...)
	}

	_, err := r.target.Write(r.tail)
	r.tail = nil
	return err
}
