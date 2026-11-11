package spec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// TurnCapHint names the cause when a review failed because the runner stopped
// the reviewer at its turn cap, and returns "" otherwise.
//
// A capped run and a finished one both end with agent_end, so the launcher's
// no-verdict error alone reads like a model that gave up. Measured 2026-08-21
// (#1157): pi ended a working reviewer after 25 turns, with zero output tokens
// on its last message and its final text cut mid-sentence. The operator's right
// move there is to re-run, not to distrust the model, and this line says so.
//
// The signature is per runner and only pi's has been measured. Any other runner
// gets no line: a guessed cause is worse than none.
func TurnCapHint(runner, transcript string) string {
	if runner != "pi" {
		return ""
	}
	f, err := os.Open(transcript)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	turns, capped := piTurnCap(f)
	if !capped {
		return ""
	}
	return fmt.Sprintf("the reviewer ended after %d turns on a message with no output, which is how pi's turn cap stops a run; re-run the review", turns)
}

// withTurnCapHint adds TurnCapHint's line to err for a review launched earlier,
// typically detached, whose missing verdict is only found when `spec archive`
// runs. The runner is the one the review-request sidecar says was launched,
// resolved through the pool, and the transcript is read where the launcher
// writes it (TranscriptPath). Without a sidecar, a pool entry or a transcript
// there is nothing to read, and err is returned as it was.
func withTurnCapHint(err error, repoRoot, specID, specDir string, slot ReviewSlot) error {
	req, found, rerr := ReadReviewRequestIn(specDir, slot)
	if rerr != nil || !found || req.Reviewer == "" {
		return err
	}
	if hint := TurnCapHint(runnerOf(repoRoot, req.Reviewer), TranscriptPathIn(repoRoot, specID, slot)); hint != "" {
		return fmt.Errorf("%w\n%s", err, hint)
	}
	// Not a turn cap: say what the reviewer itself said (#1511). A run that
	// died after the launcher's startup window is first noticed here.
	if words, ok := ReviewerLastWords(TranscriptPathIn(repoRoot, specID, slot)); ok {
		return fmt.Errorf("%w\nthe review launched with %s left no verdict. %s", err, req.Reviewer, words)
	}
	return err
}

// runnerOf is the runner of the pool member with this id, or "" when the pool
// cannot be read or has no such member.
func runnerOf(repoRoot, reviewer string) string {
	entries, err := loadReviewerPoolEntries(repoRoot)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.TrimSpace(e.ID) == reviewer {
			return e.Runner
		}
	}
	return ""
}

// ReviewerLastWords quotes the start and end of what a reviewer wrote, from
// its stderr first and then its transcript, and reports whether there was
// anything to quote.
func ReviewerLastWords(transcript string) (string, bool) {
	for _, src := range []struct{ label, path string }{
		{"stderr", StderrPath(transcript)},
		{"transcript", transcript},
	} {
		raw, err := readEnds(src.path)
		if err != nil {
			continue
		}
		out := strings.TrimSpace(raw)
		if out == "" {
			continue
		}
		return "It wrote, on " + src.label + ":\n" + headAndTail(out, src.label, src.path), true
	}
	return "", false
}

// endWindow is how much of each end of a long file readEnds reads: a
// transcript can be very large, and only quoteHalf bytes of each end are kept.
const (
	quoteHalf = 400
	endWindow = 4096
)

// readEnds reads a file whole when it is small, and otherwise only its first
// and last endWindow bytes, joined: enough for headAndTail to cut from.
func readEnds(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if st.Size() <= 2*endWindow {
		b, err := io.ReadAll(f)
		return string(b), err
	}
	head := make([]byte, endWindow)
	tail := make([]byte, endWindow)
	if _, err := f.ReadAt(head, 0); err != nil {
		return "", err
	}
	if _, err := f.ReadAt(tail, st.Size()-endWindow); err != nil && err != io.EOF {
		return "", err
	}
	return string(head) + string(tail), nil
}

// headAndTail keeps the start of a long output, where a startup failure names
// its cause, and its end, where a run writes how it finished. It cuts on rune
// boundaries and replaces any invalid byte a killed process left behind, so
// the quote is valid UTF-8.
func headAndTail(out, label, path string) string {
	if len(out) > 2*quoteHalf {
		head, tail := quoteHalf, len(out)-quoteHalf
		for head > 0 && !utf8.RuneStart(out[head]) {
			head--
		}
		for tail < len(out) && !utf8.RuneStart(out[tail]) {
			tail++
		}
		out = out[:head] + fmt.Sprintf("\n… (truncated; full %s at %s)\n", label, path) + out[tail:]
	}
	return strings.ToValidUTF8(out, "\uFFFD")
}

// piTurnCap reads a pi --mode json transcript and reports how many turns ran
// and whether the run ended the way a capped one does: agent_end was emitted
// and the last assistant message produced zero output tokens. A reviewer that
// finished on its own ends on a message that said something.
func piTurnCap(r io.Reader) (turns int, capped bool) {
	var ended, sawAssistant bool
	var lastOutput int
	// bufio.Reader rather than Scanner, for the reason SinkTranscript gives: one
	// settled message can exceed any token cap a Scanner would impose.
	br := bufio.NewReader(r)
	for {
		line, readErr := br.ReadBytes('\n')
		var ev struct {
			Type    string `json:"type"`
			Message *struct {
				Role  string `json:"role"`
				Usage *struct {
					Output int `json:"output"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &ev) == nil {
			switch ev.Type {
			case "turn_end":
				turns++
			case "agent_end":
				ended = true
			case "message_end":
				if m := ev.Message; m != nil && m.Role == "assistant" && m.Usage != nil {
					sawAssistant = true
					lastOutput = m.Usage.Output
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	return turns, ended && sawAssistant && lastOutput == 0
}
