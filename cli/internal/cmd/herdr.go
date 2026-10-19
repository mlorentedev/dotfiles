package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// herdrRun runs the herdr CLI and returns its stdout. A package var, like
// forgeRun, because every call this command makes mutates the user's layout: a
// test asserts the argv it builds without a live server, and without a server
// there is no other way to see them.
var herdrRun = func(args ...string) ([]byte, error) {
	// herdr exports HERDR_BIN_PATH to the commands it launches; a herdr that is
	// merely on PATH is the case a shell test run hits.
	bin := os.Getenv("HERDR_BIN_PATH")
	if bin == "" {
		bin = "herdr"
	}
	cmd := exec.Command(bin, args...) //nolint:gosec // herdr with arguments this command builds
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return out, fmt.Errorf("herdr %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return out, fmt.Errorf("herdr %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// herdrLookPath finds an executable on PATH. A var so a test covers both the
// provisioned and the missing case without installing a file manager; the
// missing case is the one a new machine hits before `dotf tools sync` runs.
var herdrLookPath = exec.LookPath

func newHerdrCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "herdr",
		Short: "Glue between dotf and the herdr terminal workspace manager",
		Long: "herdr provides the parts of a herdr session that herdr's own config cannot\n" +
			"express: commands that reshape a tab.\n\n" +
			"  dotf herdr explorer   # toggle a yazi file explorer docked to the right\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(newHerdrExplorerCmd())
	return c
}

// herdrPaneContext is the pane this process is allowed to reshape, plus the
// directory the explorer should show.
type herdrPaneContext struct {
	pane string
	tab  string
	cwd  string
}

// paneContext reads what herdr exports, taking the custom-command variables
// first and the pane's own environment second. The second source is what lets
// the command be typed into a pane, which is how the docked explorer is usable
// before a released dotf carries the binding that calls it.
func paneContext() (herdrPaneContext, error) {
	ctx := herdrPaneContext{
		pane: firstNonEmpty(os.Getenv("HERDR_ACTIVE_PANE_ID"), os.Getenv("HERDR_PANE_ID")),
		tab:  firstNonEmpty(os.Getenv("HERDR_ACTIVE_TAB_ID"), os.Getenv("HERDR_TAB_ID")),
		cwd:  os.Getenv("HERDR_ACTIVE_PANE_CWD"),
	}
	if ctx.pane == "" {
		return ctx, errors.New("not inside a herdr pane: neither HERDR_ACTIVE_PANE_ID nor HERDR_PANE_ID is set")
	}
	// The tab bounds the toggle: without it a second invocation would have to
	// search every pane on every workspace for one running yazi.
	if ctx.tab == "" {
		return ctx, errors.New("herdr exported no tab id for this pane (HERDR_TAB_ID unset): cannot tell which tab to toggle")
	}
	if ctx.cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ctx, fmt.Errorf("resolve the pane's directory: %w", err)
		}
		ctx.cwd = wd
	}
	return ctx, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

type herdrPaneList struct {
	Result struct {
		Panes []struct {
			PaneID string `json:"pane_id"`
			TabID  string `json:"tab_id"`
		} `json:"panes"`
	} `json:"result"`
}

type herdrSplitResult struct {
	Result struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	} `json:"result"`
}

type herdrProcessInfo struct {
	Result struct {
		ProcessInfo struct {
			ForegroundProcesses []struct {
				Name  string `json:"name"`
				Argv0 string `json:"argv0"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	} `json:"result"`
}

func newHerdrExplorerCmd() *cobra.Command {
	var ratio float64
	c := &cobra.Command{
		Use:   "explorer",
		Short: "Toggle a yazi file explorer docked to the right of this herdr pane",
		Long: "explorer opens the tab's worktree as a real split on the right, running\n" +
			"yazi in this pane's directory, and closes it again on the next call: a\n" +
			"toggle, so a second press does not stack a second explorer.\n\n" +
			"yazi is not installed here: YAZI_VERSION in versions.conf pins it and\n" +
			"`dotf tools sync` provisions it on every OS. The command refuses rather\n" +
			"than opening an empty pane when it is missing.\n\n" +
			"Inside a herdr pane it reads HERDR_ACTIVE_* (a custom command) or\n" +
			"HERDR_PANE_ID/HERDR_TAB_ID (the pane's own environment), so it works both\n" +
			"from a herdr keybinding and typed into the pane.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := paneContext()
			if err != nil {
				return err
			}
			if ratio <= 0 || ratio >= 1 {
				return fmt.Errorf("--ratio must be greater than 0 and less than 1, got %v: herdr refuses a split outside that range or renders one too thin to toggle", ratio)
			}
			if _, err := herdrLookPath("yazi"); err != nil {
				return errors.New("yazi is not on PATH: run `dotf tools sync` (YAZI_VERSION in versions.conf pins it; a machine whose deployed copy of versions.conf predates the pin needs setup to refresh it)")
			}
			open, err := explorerPane(ctx.tab, ctx.pane)
			if err != nil {
				return err
			}
			if open != "" {
				if _, err := herdrRun("pane", "close", open); err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "closed the explorer in %s\n", open)
				return nil
			}
			id, err := splitRight(ctx, ratio)
			if err != nil {
				return err
			}
			if _, err := herdrRun("pane", "run", id, "yazi"); err != nil {
				// Failing to start yazi in the pane just made is the same outcome
				// AC4 refuses: an empty pane left behind. Take it back.
				_, _ = herdrRun("pane", "close", id)
				return fmt.Errorf("start yazi in %s (the empty pane was closed): %w", id, err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "opened the explorer in %s at %s\n", id, ctx.cwd)
			return nil
		},
	}
	c.Flags().Float64Var(&ratio, "ratio", 0.3, "fraction of the tab the explorer takes, as herdr's split ratio")
	return c
}

// explorerPane returns the pane of tab that already runs yazi, or "" when none
// does. It detects the foreground process rather than a pane label: a label is
// ours to set and hers to lose (any rename clears it), while yazi holds the
// foreground for as long as it runs.
//
// Two things it refuses to guess about. The invoking pane is excluded, because
// someone who runs yazi by hand in the pane they then press the key in would
// otherwise have that pane closed under them. More than one yazi left in the tab
// is reported rather than closed: the toggle cannot tell ours from theirs, and
// closing the wrong pane is worse than not toggling.
func explorerPane(tab, invoking string) (string, error) {
	out, err := herdrRun("pane", "list")
	if err != nil {
		return "", err
	}
	var list herdrPaneList
	if err := json.Unmarshal(out, &list); err != nil {
		return "", fmt.Errorf("parse `herdr pane list`: %w", err)
	}
	var found []string
	for _, p := range list.Result.Panes {
		if p.TabID != tab || p.PaneID == invoking {
			continue
		}
		runs, err := paneRunsYazi(p.PaneID)
		if err != nil {
			return "", err
		}
		if runs {
			found = append(found, p.PaneID)
		}
	}
	switch len(found) {
	case 0:
		return "", nil
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("more than one pane in this tab is running yazi (%s): close the ones you did not expect and try again", strings.Join(found, ", "))
	}
}

func paneRunsYazi(pane string) (bool, error) {
	out, err := herdrRun("pane", "process-info", "--pane", pane)
	if err != nil {
		return false, err
	}
	var info herdrProcessInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return false, fmt.Errorf("parse `herdr pane process-info --pane %s`: %w", pane, err)
	}
	for _, p := range info.Result.ProcessInfo.ForegroundProcesses {
		name := p.Name
		if name == "" {
			name = filepath.Base(p.Argv0)
		}
		if name == "yazi" {
			return true, nil
		}
	}
	return false, nil
}

func splitRight(ctx herdrPaneContext, ratio float64) (string, error) {
	out, err := herdrRun("pane", "split",
		"--pane", ctx.pane,
		"--direction", "right",
		"--cwd", ctx.cwd,
		"--ratio", strconv.FormatFloat(ratio, 'f', 2, 64))
	if err != nil {
		return "", err
	}
	var res herdrSplitResult
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("parse `herdr pane split`: %w", err)
	}
	if res.Result.Pane.PaneID == "" {
		return "", errors.New("herdr pane split returned no pane id")
	}
	return res.Result.Pane.PaneID, nil
}
