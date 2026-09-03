package harness

import "testing"

func TestPromptFromHookPayloadAcceptsPlausibleSpellings(t *testing.T) {
	// AC6. The published hooks reference documents session_id, prompt_id,
	// transcript_path, cwd, permission_mode, effort and hook_event_name — and
	// does NOT name the field carrying the prompt text. Picking one spelling and
	// hoping is exactly how #1434 happened, so every plausible spelling is
	// accepted and the one that arrived is reported back.
	for _, field := range []string{"prompt", "user_prompt", "userPrompt", "prompt_text", "promptText", "text", "message", "input"} {
		payload := []byte(`{"hook_event_name":"UserPromptSubmit","` + field + `":"refactor the docker setup"}`)
		got, gotField, ok := PromptFromHookPayload(payload)
		if !ok {
			t.Errorf("%s: not recognised", field)
			continue
		}
		if got != "refactor the docker setup" {
			t.Errorf("%s: prompt = %q", field, got)
		}
		if gotField != field {
			t.Errorf("%s: reported field = %q, want %q", field, gotField, field)
		}
	}
}

func TestPromptFromHookPayloadReportsUnrecognised(t *testing.T) {
	// An unrecognised payload records THAT FACT rather than guessing. Same
	// defensive shape as OutcomePayloadUnrecognised: a spelling nobody has seen
	// must be visible, not silently treated as an empty prompt.
	cases := map[string][]byte{
		"no prompt field":  []byte(`{"hook_event_name":"UserPromptSubmit","session_id":"abc"}`),
		"empty object":     []byte(`{}`),
		"malformed json":   []byte(`{"prompt":`),
		"not an object":    []byte(`["prompt"]`),
		"empty input":      nil,
		"empty prompt val": []byte(`{"prompt":"   "}`),
	}
	for name, payload := range cases {
		got, field, ok := PromptFromHookPayload(payload)
		if ok {
			t.Errorf("%s: should not be recognised, got %q from %q", name, got, field)
		}
		if got != "" {
			t.Errorf("%s: prompt should be empty, got %q", name, got)
		}
	}
}

func TestPromptFromHookPayloadPrefersTheMostSpecificSpelling(t *testing.T) {
	// A payload carrying several candidates must be deterministic. `prompt` is
	// the most specific and wins over the generic `text`/`message`, so the same
	// payload never renders two different suggestions on two runs.
	payload := []byte(`{"text":"generic","message":"generic","prompt":"the real one"}`)
	got, field, ok := PromptFromHookPayload(payload)
	if !ok || got != "the real one" || field != "prompt" {
		t.Errorf("want (the real one, prompt, true), got (%q, %q, %v)", got, field, ok)
	}
}
