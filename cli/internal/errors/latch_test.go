package errors

import (
	"encoding/json"
	goerrors "errors"
	"strings"
	"testing"
)

func TestIsTerminalFailure(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "terminal failure",
			err:  NewTerminalFailure("spec verification failed"),
			want: true,
		},
		{
			name: "retryable error",
			err:  goerrors.New("retryable"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTerminalFailure(tt.err)
			if got != tt.want {
				t.Errorf("IsTerminalFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTerminalFailureFormat(t *testing.T) {
	err := NewTerminalFailure("spec verification failed")
	msg := err.Error()

	if !strings.HasPrefix(msg, HandoffPrefix) {
		t.Fatalf("expected prefix %q, got: %s", HandoffPrefix, msg)
	}

	jsonPart := strings.TrimPrefix(msg, HandoffPrefix)
	
	type testPayload struct {
		SchemaName    string `json:"schemaName"`
		RetryGuidance string `json:"retryGuidance"`
		Reason        string `json:"reason"`
	}
	var payload testPayload
	
	if jerr := json.Unmarshal([]byte(jsonPart), &payload); jerr != nil {
		t.Fatalf("failed to unmarshal JSON payload: %v", jerr)
	}

	if payload.SchemaName != SchemaName {
		t.Errorf("unexpected schemaName: %v", payload.SchemaName)
	}
	if payload.RetryGuidance != RetryGuidance {
		t.Errorf("unexpected retryGuidance: %v", payload.RetryGuidance)
	}
	if payload.Reason != "spec verification failed" {
		t.Errorf("unexpected reason: %v", payload.Reason)
	}
}
