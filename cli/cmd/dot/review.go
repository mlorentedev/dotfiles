package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const reviewSystemPrompt = `You are a rigorous senior code reviewer. You receive a unified git diff.
Review ONLY what the diff changes. Report, in this order:
1. Bugs and logic errors
2. Security issues
3. Cross-platform pitfalls (Linux/Windows)
4. Maintainability concerns
For each finding give: severity (blocker/major/minor), file and hunk location, and a one-line fix suggestion.
If the diff is clean, say so briefly. Output GitHub-flavored markdown. Do not restate the diff.`

// reviewMaxTokens bounds the completion size — a cost guard so a single
// review can never bill unbounded output.
const reviewMaxTokens = 4096

type provider struct {
	baseURL      string
	apiKey       string
	defaultModel string
}

func resolveProvider(name string) (provider, error) {
	switch name {
	case "nan":
		base := os.Getenv("NAN_BASE_URL")
		if base == "" {
			return provider{}, fmt.Errorf("NAN_BASE_URL is not set (exported by the dotfiles shell profile)")
		}
		key := os.Getenv("NAN_API_KEY")
		if key == "" {
			return provider{}, fmt.Errorf("NAN_API_KEY is not set (loaded by load-secrets from the age store)")
		}
		return provider{baseURL: base, apiKey: key, defaultModel: "deepseek-v4-flash"}, nil
	case "openrouter":
		key := os.Getenv("OPENROUTER_API_KEY")
		if key == "" {
			return provider{}, fmt.Errorf("OPENROUTER_API_KEY is not set (loaded by load-secrets from the age store)")
		}
		return provider{baseURL: "https://openrouter.ai/api/v1", apiKey: key, defaultModel: "deepseek/deepseek-chat"}, nil
	default:
		return provider{}, fmt.Errorf("unknown provider %q (valid: nan, openrouter)", name)
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

func newReviewCmd() *cobra.Command {
	var (
		providerName string
		model        string
		maxBytes     int64
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Cross-model code review of a diff read from stdin",
		Long: `Reads a unified diff from stdin and asks a non-Claude model for a
decorrelated second-opinion review, printed to stdout as markdown.

Providers and required environment:
  nan         NAN_BASE_URL + NAN_API_KEY   (default model deepseek-v4-flash)
  openrouter  OPENROUTER_API_KEY           (default model deepseek/deepseek-chat)

Exit codes: 0 review produced; 1 on empty stdin, missing env, oversized diff,
HTTP error or timeout.

Privacy: the diff is sent to the selected third-party API. Think before piping
diffs from repositories you do not own.`,
		Example:      "  git diff main...HEAD | dot review --provider nan",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			diff, err := readDiff(cmd.InOrStdin(), maxBytes)
			if err != nil {
				return err
			}
			prov, err := resolveProvider(providerName)
			if err != nil {
				return err
			}
			if model == "" {
				model = prov.defaultModel
			}
			review, err := requestReview(prov, model, diff, timeout)
			if err != nil {
				return err
			}
			cmd.Println(review)
			return nil
		},
	}

	cmd.Flags().StringVar(&providerName, "provider", "nan", "review provider: nan or openrouter")
	cmd.Flags().StringVar(&model, "model", "", "model override (default: provider-specific)")
	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 200_000, "fail if the diff exceeds this size instead of reviewing it truncated")
	cmd.Flags().DurationVar(&timeout, "timeout", 120*time.Second, "HTTP timeout for the provider request")
	return cmd
}

func readDiff(in io.Reader, maxBytes int64) (string, error) {
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

func requestReview(prov provider, model, diff string, timeout time.Duration) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: reviewSystemPrompt},
			{Role: "user", Content: diff},
		},
		MaxTokens:   reviewMaxTokens,
		Temperature: 0.2,
	})
	if err != nil {
		return "", fmt.Errorf("encoding request: %w", err)
	}

	url := strings.TrimRight(prov.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+prov.apiKey)
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
