package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/env"
	"github.com/mlorentedev/dotfiles/cli/internal/search"
	"github.com/mlorentedev/dotfiles/cli/internal/vault"
)

func newSearchCmd() *cobra.Command {
	var (
		filterType string
		limit      int
		jsonOut    bool
		customDir  string
	)

	cmd := &cobra.Command{
		Use:   "search <query...>",
		Short: "Search knowledge vault patterns, skills, lessons, and specs",
		Long: `search queries the knowledge vault and returns relevant patterns,
skills, lessons, and documentation ranked by relevance.`,
		Example: `  dotf search socratic diagnostic
  dotf search --type pattern "root cause"
  dotf search --type skill debugging
  dotf search --json "review attestation"`,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			searchDir := customDir
			if searchDir == "" {
				searchDir = vault.ResolveVault()
			}
			if searchDir == "" {
				// Fallback to dotfiles repo root or current directory
				searchDir = env.RepoDir()
			}

			results, err := search.Search(searchDir, query, search.ItemType(filterType), limit)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}

			out := cmd.OutOrStdout()
			if len(results) == 0 {
				_, _ = fmt.Fprintf(out, "No results found for %q in %s\n", query, searchDir)
				return nil
			}

			_, _ = fmt.Fprintf(out, "Search results for %q (%d hits):\n\n", query, len(results))
			for i, r := range results {
				typeTag := fmt.Sprintf("[%s]", strings.ToUpper(string(r.Type)))
				_, _ = fmt.Fprintf(out, "%2d. %-10s %s (id: %s)\n", i+1, typeTag, r.Title, r.ID)
				_, _ = fmt.Fprintf(out, "    Path: %s\n", r.Path)
				if r.Description != "" {
					_, _ = fmt.Fprintf(out, "    Description: %s\n", r.Description)
				} else if r.Snippet != "" {
					_, _ = fmt.Fprintf(out, "    Snippet: %s\n", r.Snippet)
				}
				_, _ = fmt.Fprintln(out)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&filterType, "type", "t", "all", "filter by type (all, pattern, skill, lesson, spec, doc)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "maximum number of results to display")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output results as JSON")
	cmd.Flags().StringVarP(&customDir, "dir", "d", "", "override directory to search (defaults to knowledge vault)")

	return cmd
}
