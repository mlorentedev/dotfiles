package spec

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// The reviewer streams its work as jsonl. Two consumers want different things
// from that stream and cannot share one file:
//
//   - the tmux pane, watched live, wants every frame — the incremental deltas
//     ARE the visible progress;
//   - the transcript on disk wants the settled events, because its purpose is
//     that a finished review can be audited afterwards.
//
// Writing the raw stream to disk served the first and defeated the second.
// Measured on a real 25-minute review: 527 MB, of which 525 MB was
// `message_update` — the same message re-emitted at every growing length.
// Keeping only settled events left 1.95 MB (0.37%) and 664 lines, which is a
// file a human opens and a script greps (#995).
//
// So the sink passes everything through and filters only what it stores.

// isIncrementalFrame reports whether a line is a partial frame on the way to an
// event rather than an event. Anything unrecognised is KEPT: a line we cannot
// parse is more likely a diagnostic worth having than noise worth dropping, and
// this filter must never be the reason a failure left no trace.
func isIncrementalFrame(line []byte) bool {
	var ev struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &ev); err != nil {
		return false
	}
	return ev.Type == "message_update" || strings.HasSuffix(ev.Type, "_delta")
}

// SinkTranscript copies in to out verbatim while writing only the auditable
// events to path. It is the `tee` of the reviewer pipeline, with the storage
// side filtered.
//
// Errors writing the transcript do not stop the passthrough: the review itself
// is the valuable thing in flight, and losing its live output because a file
// could not be written would trade a large problem for a larger one.
func SinkTranscript(in io.Reader, out io.Writer, path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("opening the transcript for writing: %w", err)
	}
	// Close is checked, not deferred blindly: on a buffered file it is where a
	// short write finally surfaces, and swallowing it would mean reporting a
	// complete transcript we did not finish writing — the failure mode this
	// whole file exists to stop.
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing the transcript: %w", cerr)
		}
	}()

	// bufio.Reader, not Scanner: a single settled message can exceed any token
	// cap we would pick, and Scanner turns that into an error that truncates
	// the stream. ReadString has no such ceiling.
	r := bufio.NewReader(in)
	w := bufio.NewWriter(f)

	var storeErr error
	for {
		line, readErr := r.ReadString('\n')
		if line != "" {
			if _, err := io.WriteString(out, line); err != nil {
				return fmt.Errorf("passing the reviewer stream through: %w", err)
			}
			if storeErr == nil && !isIncrementalFrame([]byte(strings.TrimRight(line, "\n"))) {
				if _, err := w.WriteString(line); err != nil {
					storeErr = err
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return fmt.Errorf("reading the reviewer stream: %w", readErr)
		}
	}
	if ferr := w.Flush(); ferr != nil && storeErr == nil {
		storeErr = ferr
	}
	if storeErr != nil {
		return fmt.Errorf("writing the transcript: %w", storeErr)
	}
	return nil
}
