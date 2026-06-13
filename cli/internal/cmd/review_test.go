package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleDiff = `diff --git a/foo.go b/foo.go
index 1111111..2222222 100644
--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,3 @@
-func foo() {}
+func foo() error { return nil }
`

// openAIMock returns an httptest server speaking the chat-completions shape,
// recording the last request body and auth header.
func openAIMock(t *testing.T, status int, content string, delay time.Duration) (*httptest.Server, *capturedRequest) {
	t.Helper()
	cap := &capturedRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.auth = r.Header.Get("Authorization")
		var body struct {
			Model     string `json:"model"`
			MaxTokens int    `json:"max_tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		cap.model = body.Model
		cap.maxTokens = body.MaxTokens
		if delay > 0 {
			time.Sleep(delay)
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"content": content}},
				},
			})
		}
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

type capturedRequest struct {
	auth      string
	model     string
	maxTokens int
}

func executeReview(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := New("dev")
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"review"}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func TestReviewErrorContract(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		args    []string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "empty stdin",
			stdin:   "",
			env:     map[string]string{"NAN_BASE_URL": "http://localhost:1", "NAN_API_KEY": "k"},
			wantErr: "no diff on stdin",
		},
		{
			name:    "missing NAN_API_KEY",
			stdin:   sampleDiff,
			env:     map[string]string{"NAN_BASE_URL": "http://localhost:1", "NAN_API_KEY": ""},
			wantErr: "NAN_API_KEY",
		},
		{
			name:    "missing NAN_BASE_URL",
			stdin:   sampleDiff,
			env:     map[string]string{"NAN_BASE_URL": "", "NAN_API_KEY": "k"},
			wantErr: "NAN_BASE_URL",
		},
		{
			name:    "missing OPENROUTER_API_KEY",
			stdin:   sampleDiff,
			args:    []string{"--provider", "openrouter"},
			env:     map[string]string{"OPENROUTER_API_KEY": ""},
			wantErr: "OPENROUTER_API_KEY",
		},
		{
			name:    "unknown provider",
			stdin:   sampleDiff,
			args:    []string{"--provider", "bogus"},
			wantErr: "unknown provider",
		},
		{
			name:    "diff exceeds max-bytes",
			stdin:   sampleDiff,
			args:    []string{"--max-bytes", "10"},
			env:     map[string]string{"NAN_BASE_URL": "http://localhost:1", "NAN_API_KEY": "k"},
			wantErr: "max-bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			_, err := executeReview(t, tt.stdin, tt.args...)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestReviewHappyPath(t *testing.T) {
	srv, cap := openAIMock(t, http.StatusOK, "## Review\nLooks good.", 0)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "test-key")

	out, err := executeReview(t, sampleDiff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Looks good.") {
		t.Errorf("output %q does not contain the review", out)
	}
	if cap.auth != "Bearer test-key" {
		t.Errorf("auth header = %q, want Bearer test-key", cap.auth)
	}
	if cap.model != "deepseek-v4-flash" {
		t.Errorf("default nan model = %q, want deepseek-v4-flash", cap.model)
	}
}

func TestReviewModelOverride(t *testing.T) {
	srv, cap := openAIMock(t, http.StatusOK, "ok", 0)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "k")

	if _, err := executeReview(t, sampleDiff, "--model", "qwen3.6"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.model != "qwen3.6" {
		t.Errorf("model = %q, want qwen3.6", cap.model)
	}
}

func TestReviewMaxTokensBounded(t *testing.T) {
	srv, cap := openAIMock(t, http.StatusOK, "ok", 0)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "k")

	if _, err := executeReview(t, sampleDiff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.maxTokens <= 0 {
		t.Errorf("max_tokens = %d, must be a bounded positive cost guard", cap.maxTokens)
	}
}

func TestReviewNoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "k")

	_, err := executeReview(t, sampleDiff)
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Errorf("expected no-choices error, got %v", err)
	}
}

func TestReviewHTTPError(t *testing.T) {
	srv, _ := openAIMock(t, http.StatusInternalServerError, "", 0)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "k")

	_, err := executeReview(t, sampleDiff)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Errorf("expected HTTP 500 error, got %v", err)
	}
}

func TestReviewTimeout(t *testing.T) {
	srv, _ := openAIMock(t, http.StatusOK, "late", 300*time.Millisecond)
	t.Setenv("NAN_BASE_URL", srv.URL)
	t.Setenv("NAN_API_KEY", "k")

	_, err := executeReview(t, sampleDiff, "--timeout", "50ms")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
