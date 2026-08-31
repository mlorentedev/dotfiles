package errors

import (
	"encoding/json"
	goerrors "errors"
)

const (
	HandoffPrefix = "GENTLE_AI_SDD_FAILURE "
	SchemaName    = "gentle-ai.sdd-task-result-failure/v1"
	RetryGuidance = "Do not retry or advance SDD; inspect the existing artifact state and surface the terminal failure to the user."
)

// TerminalFailureError represents an unrecoverable error that requires agents
// to stop and not retry blindly. It formats itself as a strict JSON latch.
type TerminalFailureError struct {
	reason string
}

// handoffPayload matches the structured format expected by robust orchestrators.
type handoffPayload struct {
	SchemaName    string `json:"schemaName"`
	RetryGuidance string `json:"retryGuidance"`
	Reason        string `json:"reason"`
}

func (e *TerminalFailureError) Error() string {
	payload := handoffPayload{
		SchemaName:    SchemaName,
		RetryGuidance: RetryGuidance,
		Reason:        e.reason,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		// Fallback if marshaling fails (should be impossible with standard types)
		return HandoffPrefix + `{"schemaName":"` + SchemaName + `","retryGuidance":"` + RetryGuidance + `","reason":"serialization error"}`
	}
	return HandoffPrefix + string(bytes)
}

// NewTerminalFailure creates a new TerminalFailureError.
func NewTerminalFailure(reason string) error {
	return &TerminalFailureError{reason: reason}
}

// IsTerminalFailure checks if the given error is a TerminalFailureError.
func IsTerminalFailure(err error) bool {
	var tfe *TerminalFailureError
	return goerrors.As(err, &tfe)
}
