package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/mem"
)

// agyHookFormat is the --format of agy's PreInvocation hook (HARNESS-195).
const agyHookFormat = "agy-hook"

// runAgyHook answers agy's PreInvocation hook with the session brief on the
// first invocation of a conversation and `{}` on every later one. It never
// fails: a hook that errors would block the user's prompt over a banner, so a
// problem is reported on stderr and the answer is `{}`.
func runAgyHook(cmd *cobra.Command) error {
	payload, _ := io.ReadAll(cmd.InOrStdin())
	hook := mem.ParseAgyHook(payload)
	text := ""
	first, err := mem.FirstInvocation(os.TempDir(), hook.ConversationID)
	if err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "agy session line skipped: %v\n", err)
	}
	if first {
		cwd := hook.Cwd
		if cwd == "" {
			cwd, _ = os.Getwd()
		}
		text = agnosticBrief(cmd, cwd)
	}
	out, err := mem.AgyInjection(text)
	if err != nil {
		out = "{}\n"
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}
