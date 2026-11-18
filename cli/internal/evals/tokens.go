package evals

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
)

// ReadTokens sums what a runner reported spending in its transcript. It
// returns nil, "not reported", for a runner whose transcript it cannot read,
// a transcript that reports nothing, or one whose every count is zero: a
// record must never claim a run was free. A line that is not JSON, such as
// the last one of a runner that was killed mid-write, is skipped.
func ReadTokens(runner, transcript string) *Tokens {
	var add func([]byte, *Tokens)
	switch runner {
	case "pi":
		add = addPiUsage
	case "agy":
		add = addAgyUsage
	default:
		return nil
	}
	f, err := os.Open(transcript)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	var sum Tokens
	// A bufio.Reader, not a Scanner: a pi line carrying the system prompt is
	// far longer than a Scanner's 64KB token.
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		add(line, &sum)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil
			}
			break
		}
	}
	if sum == (Tokens{}) {
		return nil
	}
	return &sum
}

// addPiUsage counts an assistant's message_end. pi repeats the same usage on
// turn_end and every message again on agent_end, so those are not counted.
func addPiUsage(line []byte, sum *Tokens) {
	var ev struct {
		Type    string `json:"type"`
		Message struct {
			Role  string `json:"role"`
			Usage struct {
				Input      int `json:"input"`
				Output     int `json:"output"`
				CacheRead  int `json:"cacheRead"`
				CacheWrite int `json:"cacheWrite"`
				Reasoning  int `json:"reasoning"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &ev) != nil || ev.Type != "message_end" || ev.Message.Role != "assistant" {
		return
	}
	u := ev.Message.Usage
	sum.Input += u.Input
	sum.Cached += u.CacheRead + u.CacheWrite
	sum.Output += u.Output
	sum.Thinking += u.Reasoning
}

// addAgyUsage counts a finished step: agy reports a step's usage once, on the
// step_update whose state is DONE.
func addAgyUsage(line []byte, sum *Tokens) {
	var ev struct {
		Event string `json:"event"`
		Step  struct {
			State string `json:"state"`
			Usage struct {
				Input     int `json:"input_tokens"`
				Output    int `json:"output_tokens"`
				Thinking  int `json:"thinking_tokens"`
				CacheRead int `json:"cache_read_tokens"`
			} `json:"usage"`
		} `json:"step_update"`
	}
	if json.Unmarshal(line, &ev) != nil || ev.Event != "step_update" || ev.Step.State != "DONE" {
		return
	}
	u := ev.Step.Usage
	sum.Input += u.Input
	sum.Cached += u.CacheRead
	sum.Output += u.Output
	sum.Thinking += u.Thinking
}
