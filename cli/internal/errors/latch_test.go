package errors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTerminalFailure(t *testing.T) {
	err := NewTerminalFailure("spec verification failed")

	if !IsTerminalFailure(err) {
		t.Errorf("expected err to be a TerminalFailure")
	}

	msg := err.Error()
	if !strings.HasPrefix(msg, "GENTLE_AI_SDD_FAILURE ") {
		t.Fatalf("expected GENTLE_AI_SDD_FAILURE prefix, got: %s", msg)
	}

	jsonPart := strings.TrimPrefix(msg, "GENTLE_AI_SDD_FAILURE ")
	var payload map[string]interface{}
	if jerr := json.Unmarshal([]byte(jsonPart), &payload); jerr != nil {
		t.Fatalf("failed to unmarshal JSON payload: %v", jerr)
	}

	if payload["schemaName"] != "gentle-ai.sdd-task-result-failure/v1" {
		t.Errorf("unexpected schemaName: %v", payload["schemaName"])
	}
	if payload["retryGuidance"] != "Do not retry or advance SDD; inspect the existing artifact state and surface the terminal failure to the user." {
		t.Errorf("unexpected retryGuidance: %v", payload["retryGuidance"])
	}
	if payload["reason"] != "spec verification failed" {
		t.Errorf("unexpected reason: %v", payload["reason"])
	}
}
