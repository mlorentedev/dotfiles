package spec

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	entries, perr := loadReviewerPoolEntries(repoRoot)
	if perr != nil {
		return err
	}
	for _, e := range entries {
		if strings.TrimSpace(e.ID) != req.Reviewer {
			continue
		}
		if hint := TurnCapHint(e.Runner, TranscriptPathIn(repoRoot, specID, slot)); hint != "" {
			return fmt.Errorf("%w\n%s", err, hint)
		}
		break
	}
	return err
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
