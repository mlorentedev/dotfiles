package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

func newHarnessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "harness",
		Short: "Harness, pattern trigger and skill suggestion tools",
		Long: `harness provides utilities for inspecting agent harness configuration,
pattern triggers, and workflow integrations.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newHarnessTriggersCmd())
	cmd.AddCommand(newHarnessSuggestCmd())
	cmd.AddCommand(newHarnessResolveTierCmd())
	cmd.AddCommand(newHarnessResolveCapabilitiesCmd())
	cmd.AddCommand(newHarnessResolveSkillsCmd())
	cmd.AddCommand(newHarnessGateCmd())
	cmd.AddCommand(newHarnessMirrorCmd())
	cmd.AddCommand(newHarnessPresenceCmd())
	return cmd
}

func newHarnessSuggestCmd() *cobra.Command {
	var (
		prompt   string
		diffMode bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "suggest [paths...]",
		Short: "Suggest relevant patterns and skills for a prompt, files, or diff",
		Long: `suggest analyzes a task description (prompt), touched files, or git diff
and suggests matching patterns and skills to load.`,
		Example: `  dotf harness suggest --prompt "we need to debug this memory leak in python"
  dotf harness suggest specs/FEATURE-1/proposal.md
  dotf harness suggest --prompt "refactor docker setup" Dockerfile
  git diff main...HEAD | dotf harness suggest --diff`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := harness.LoadTriggers("")
			if err != nil {
				return err
			}

			var paths []string
			if diffMode {
				var diffContent string
				if len(args) > 0 {
					data, err := os.ReadFile(args[0])
					if err != nil {
						return err
					}
					diffContent = string(data)
				} else {
					data, err := io.ReadAll(cmd.InOrStdin())
					if err != nil {
						return fmt.Errorf("read diff from stdin: %w", err)
					}
					diffContent = string(data)
				}
				paths = harness.ExtractPathsFromDiff(diffContent)
			} else {
				if len(args) == 0 && prompt == "" {
					if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
						if data, err := io.ReadAll(cmd.InOrStdin()); err == nil && len(data) > 0 {
							paths = harness.ExtractPathsFromDiff(string(data))
						}
					}
				} else {
					paths = args
				}
			}

			sugg := harness.Suggest(cfg.Triggers, prompt, paths)

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(sugg)
			}

			out := cmd.OutOrStdout()
			if len(sugg.Patterns) > 0 {
				_, _ = fmt.Fprintln(out, "Patterns:")
				for _, p := range sugg.Patterns {
					_, _ = fmt.Fprintf(out, "  - %s\n", p)
				}
			}
			if len(sugg.Skills) > 0 {
				if len(sugg.Patterns) > 0 {
					_, _ = fmt.Fprintln(out)
				}
				_, _ = fmt.Fprintln(out, "Skills:")
				for _, s := range sugg.Skills {
					_, _ = fmt.Fprintf(out, "  - %s\n", s)
				}
			}
			if len(sugg.Patterns) == 0 && len(sugg.Skills) == 0 {
				_, _ = fmt.Fprintln(out, "No matching patterns or skills found.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "user prompt or task description to analyze")
	cmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "read unified diff from file or stdin")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output results as a JSON object")
	return cmd
}

func newHarnessTriggersCmd() *cobra.Command {
	var (
		diffMode bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "triggers [paths...]",
		Short: "Resolve matching vault patterns for file paths or git diff",
		Long: `triggers inspects file paths or a git diff and resolves the matching
pattern IDs from the harness trigger rules.`,
		Example: `  dotf harness triggers specs/FEATURE-1/proposal.md
  dotf harness triggers Dockerfile tests/foo_test.go
  git diff main...HEAD | dotf harness triggers --diff
  dotf harness triggers --json specs/tasks.md`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := harness.LoadTriggers("")
			if err != nil {
				return err
			}

			var patterns []string
			if diffMode {
				diffContent := ""
				if len(args) > 0 {
					data, err := os.ReadFile(args[0])
					if err != nil {
						return err
					}
					diffContent = string(data)
				} else {
					data, err := io.ReadAll(cmd.InOrStdin())
					if err != nil {
						return fmt.Errorf("read diff from stdin: %w", err)
					}
					diffContent = string(data)
				}
				patterns = harness.MatchDiff(cfg.Triggers, diffContent)
			} else {
				if len(args) == 0 {
					if stat, err := os.Stdin.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
						if data, err := io.ReadAll(cmd.InOrStdin()); err == nil && len(data) > 0 {
							patterns = harness.MatchDiff(cfg.Triggers, string(data))
						}
					}
				} else {
					patterns = harness.MatchPaths(cfg.Triggers, args)
				}
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(patterns)
			}
			for _, p := range patterns {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), p)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&diffMode, "diff", "d", false, "read and parse unified diff instead of raw file paths")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output results as a JSON array")
	return cmd
}
