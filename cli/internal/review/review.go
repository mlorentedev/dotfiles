// Package review implements the cross-model code-review domain: provider
// resolution, diff intake from stdin, and the OpenAI-compatible
// chat-completions request (stdlib net/http only, per CLI-003).
package review

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const systemPrompt = `You are a rigorous senior code reviewer. You receive a unified git diff.
Review ONLY what the diff changes. Report, in this order:
1. Bugs and logic errors
2. Security issues
3. Cross-platform pitfalls (Linux/Windows)
4. Maintainability concerns
For each finding give: severity (blocker/major/minor), file and hunk location, and a one-line fix suggestion.
If the diff is clean, say so briefly. Output GitHub-flavored markdown. Do not restate the diff.`

// maxTokens bounds the completion size — a cost guard so a single
// review can never bill unbounded output.
const maxTokens = 4096

// Provider is a resolved chat-completions endpoint plus its default model.
type Provider struct {
	BaseURL      string
	APIKey       string
	DefaultModel string
}

// Resolve maps a provider name to its endpoint, reading the required
// environment variables and naming the missing one in the error.
func Resolve(name string) (Provider, error) {
	switch name {
	case "nan":
		base := os.Getenv("NAN_BASE_URL")
		if base == "" {
			return Provider{}, fmt.Errorf("NAN_BASE_URL is not set (exported by the dotfiles shell profile)")
		}
		key := os.Getenv("NAN_API_KEY")
		if key == "" {
			return Provider{}, fmt.Errorf("NAN_API_KEY is not set (loaded by load-secrets from the age store)")
		}
		return Provider{BaseURL: base, APIKey: key, DefaultModel: "deepseek-v4-flash"}, nil
	case "openrouter":
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return Provider{}, fmt.Errorf("OPENROUTER_API_KEY is not set (loaded by load-secrets from the age store)")
		}
		return Provider{BaseURL: "https://openrouter.ai/api/v1", APIKey: key, DefaultModel: "deepseek/deepseek-chat"}, nil
	default:
		return Provider{}, fmt.Errorf("unknown provider %q (valid: nan, openrouter)", name)
	}
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float32       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ReadDiff consumes stdin up to maxBytes and fails — never truncates — when
// the diff is oversized or empty.
func ReadDiff(in io.Reader, maxBytes int64) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(in, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	if int64(len(raw)) > maxBytes {
		return "", fmt.Errorf("diff exceeds --max-bytes=%d; review a smaller range (try git diff --stat to split)", maxBytes)
	}
	diff := strings.TrimSpace(string(raw))
	if diff == "" {
		return "", fmt.Errorf("no diff on stdin; usage: git diff main...HEAD | dot review")
	}
	return diff, nil
}

// Request sends the diff to the provider and returns the review markdown.
func Request(prov Provider, model, diff string, timeout time.Duration) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: diff},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	url := strings.TrimRight(prov.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+prov.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("provider request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("provider returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decoding provider response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}
