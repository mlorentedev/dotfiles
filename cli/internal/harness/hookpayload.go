package harness

import (
	"encoding/json"
	"strings"
)

// promptFieldCandidates are the spellings PromptFromHookPayload will accept for
// the field carrying the user's prompt text, most specific first.
//
// Order is the determinism contract: a payload carrying several candidates must
// always resolve to the same one, or the same prompt renders two different
// suggestions on two runs.
var promptFieldCandidates = []string{
	"prompt",
	"user_prompt",
	"userPrompt",
	"prompt_text",
	"promptText",
	"text",
	"message",
	"input",
}

// PromptFromHookPayload extracts the user's prompt from a UserPromptSubmit hook
// payload, returning the text, the field name it arrived under, and whether it
// was found at all.
//
// THE FIELD NAME IS NOT DOCUMENTED. The published hooks reference names
// `session_id`, `prompt_id`, `transcript_path`, `cwd`, `permission_mode`,
// `effort` and `hook_event_name` — and does not name the field carrying the
// prompt. Assuming a spelling is exactly how #1434 happened: a payload shape was
// taken on faith, the assumption was wrong, and the feature silently did nothing.
//
// So: accept every plausible spelling, and REPORT WHICH ONE ARRIVED, in the
// defensive shape of OutcomePayloadUnrecognised. The caller records the field so
// the guess becomes a measurement the next session can read, rather than a
// belief this one held.
//
// ok=false covers malformed JSON, a missing field, and a present-but-blank
// value. All three mean the same thing to the caller — there is nothing to
// suggest on — but none of them is an error: on UserPromptSubmit a non-zero exit
// erases the user's prompt, so there is no failure here worth that price.
func PromptFromHookPayload(data []byte) (prompt, field string, ok bool) {
	if len(data) == 0 {
		return "", "", false
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		// Includes a JSON array or scalar, which unmarshals into a map with an
		// error rather than panicking. Unrecognised is not an error.
		return "", "", false
	}

	for _, candidate := range promptFieldCandidates {
		v, present := payload[candidate]
		if !present {
			continue
		}
		s, isString := v.(string)
		if !isString {
			continue
		}
		if strings.TrimSpace(s) == "" {
			continue
		}
		return s, candidate, true
	}

	return "", "", false
}
