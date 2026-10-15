package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/lane"
)

// laneExec starts the agent; a variable so a test records the argv instead of
// replacing the test process.
var laneExec = lane.Exec

func newLaneCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "lane",
		Short:        "Hand a work lane from one agent session to the next",
		SilenceUsage: true,
		RunE:         func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(newLaneExecCmd())
	return c
}

func newLaneExecCmd() *cobra.Command {
	var agent, promptFile string
	c := &cobra.Command{
		Use:   "exec",
		Short: "Start an agent on a prompt read from a file",
		Long: "exec replaces this process with the named agent's interactive command and\n" +
			"hands it the prompt as one argument, so no shell quoting is involved and the\n" +
			"same call works from zsh, bash and PowerShell. An agent dotf has no launch\n" +
			"command for is refused, naming the ones it has.",
		Example:       "  dotf lane exec --agent claude --prompt-file prompt.md",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			b, err := os.ReadFile(promptFile)
			if err != nil {
				return fmt.Errorf("lane exec: %w", err)
			}
			argv, err := lane.LaunchArgv(agent, strings.TrimSpace(string(b)))
			if err != nil {
				return err
			}
			return laneExec(argv)
		},
	}
	c.Flags().StringVar(&agent, "agent", "", "agent to start ("+strings.Join(lane.Agents(), ", ")+")")
	c.Flags().StringVar(&promptFile, "prompt-file", "", "file holding the prompt")
	_ = c.MarkFlagRequired("agent")
	_ = c.MarkFlagRequired("prompt-file")
	return c
}
