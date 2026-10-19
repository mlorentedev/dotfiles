package herdr

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mlorentedev/dotfiles/cli/internal/tools"
)

// ExplorerLabel is the pane label this package puts on the explorer it creates,
// and the identity it closes later. Identity is a label rather than "a pane
// running yazi" because a user's own yazi is a pane running yazi too, and
// closing that one is worse than refusing to toggle (#2815 review).
const ExplorerLabel = "files"

// ExplorerRatioDefault is the fraction of the tab the docked explorer takes.
const ExplorerRatioDefault = 0.3

// PaneContext is the pane to reshape, and the directory to show in it.
type PaneContext struct {
	Pane string
	Cwd  string
}

// ExplorerContext reads what herdr exports, taking the custom-command variables
// first and the pane's own environment second: the second is what lets the
// command be typed into a pane, which is how the docked explorer is usable
// before a released dotf carries the binding that calls it.
//
// It reads no tab id on purpose. herdr keeps exporting the id a process was
// launched under, so a pane moved to another tab would send the toggle to the
// tab it left; ToggleExplorer asks herdr which tab the pane is in now.
func ExplorerContext() (PaneContext, error) {
	ctx := PaneContext{
		Pane: firstNonEmpty(os.Getenv("HERDR_ACTIVE_PANE_ID"), os.Getenv("HERDR_PANE_ID")),
		Cwd:  os.Getenv("HERDR_ACTIVE_PANE_CWD"),
	}
	if ctx.Pane == "" {
		return ctx, errors.New("not inside a herdr pane: neither HERDR_ACTIVE_PANE_ID nor HERDR_PANE_ID is set")
	}
	if ctx.Cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ctx, fmt.Errorf("resolve the pane's directory: %w", err)
		}
		ctx.Cwd = wd
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

// ValidateExplorerRatio rejects ratios herdr would either refuse or render too
// thin to press again: the toggle has to be able to find what it made.
func ValidateExplorerRatio(ratio float64) error {
	if ratio <= 0 || ratio >= 1 {
		return fmt.Errorf("--ratio must be greater than 0 and less than 1, got %v", ratio)
	}
	return nil
}

type paneEntry struct {
	PaneID string `json:"pane_id"`
	TabID  string `json:"tab_id"`
	Label  string `json:"label"`
}

type paneList struct {
	Result struct {
		Panes []paneEntry `json:"panes"`
	} `json:"result"`
}

type splitResult struct {
	Result struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	} `json:"result"`
}

type processInfo struct {
	Result struct {
		ProcessInfo struct {
			ForegroundProcesses []struct {
				Name  string `json:"name"`
				Argv0 string `json:"argv0"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	} `json:"result"`
}

// ToggleExplorer closes the explorer when this tab has one, and opens it
// otherwise. It returns a line describing what it did.
//
// Three rules, each of them a way of not guessing:
//   - the invoking pane is never a candidate, so someone who runs yazi by hand
//     in the pane they press the key in keeps it, agent included;
//   - only a pane carrying ExplorerLabel is ours; another pane running yazi is
//     named in an error instead of closed;
//   - a labelled pane is closed even when yazi has already exited inside it,
//     which is the empty pane a later press would otherwise stack beside.
func ToggleExplorer(run tools.Runner, ctx PaneContext, ratio float64) (string, error) {
	if err := ValidateExplorerRatio(ratio); err != nil {
		return "", err
	}
	if out, err := run("yazi", "--version"); err != nil {
		return "", fmt.Errorf("yazi is not on PATH (%s): run `dotf tools sync` (YAZI_VERSION in versions.conf pins it; a machine whose deployed copy of versions.conf predates the pin needs setup to refresh it)", strings.TrimSpace(string(out)))
	}
	tab, panes, err := currentTab(run, ctx.Pane)
	if err != nil {
		return "", err
	}
	ours, strangers, err := classify(run, panes, ctx.Pane, tab)
	if err != nil {
		return "", err
	}
	if len(ours) > 0 {
		if out, err := run("herdr", "pane", "close", ours[0]); err != nil {
			_, msg := errorCode(out, err)
			return "", fmt.Errorf("close the explorer in %s: %s", ours[0], msg)
		}
		return fmt.Sprintf("closed the explorer in %s", ours[0]), nil
	}
	if len(strangers) > 0 {
		return "", fmt.Errorf("a pane running yazi is not this explorer (%s): close it and try again, or open the explorer in another tab", strings.Join(strangers, ", "))
	}
	id, err := splitRight(run, ctx, ratio)
	if err != nil {
		return "", err
	}
	// The label goes on before yazi starts, so a run that fails leaves a pane the
	// next press recognises as ours and closes, rather than a stray empty one.
	if out, err := run("herdr", "pane", "rename", id, ExplorerLabel); err != nil {
		_, msg := errorCode(out, err)
		_, _ = run("herdr", "pane", "close", id)
		return "", fmt.Errorf("label the explorer in %s: %s", id, msg)
	}
	if out, err := run("herdr", "pane", "run", id, "yazi"); err != nil {
		_, msg := errorCode(out, err)
		_, _ = run("herdr", "pane", "close", id)
		return "", fmt.Errorf("start yazi in %s: %s", id, msg)
	}
	return fmt.Sprintf("opened the explorer in %s at %s", id, ctx.Cwd), nil
}

// currentTab returns the tab the pane is in now, and every pane herdr lists.
func currentTab(run tools.Runner, pane string) (string, []paneEntry, error) {
	out, err := run("herdr", "pane", "list")
	if err != nil {
		_, msg := errorCode(out, err)
		return "", nil, fmt.Errorf("herdr pane list: %s", msg)
	}
	var list paneList
	if err := json.Unmarshal(out, &list); err != nil {
		return "", nil, fmt.Errorf("parse `herdr pane list`: %w", err)
	}
	for _, p := range list.Result.Panes {
		if p.PaneID == pane {
			return p.TabID, list.Result.Panes, nil
		}
	}
	return "", nil, fmt.Errorf("herdr does not list pane %s: is this shell inside the herdr server you think it is?", pane)
}

// classify splits the tab's panes into the explorers this command owns and
// panes running yazi that it does not. The invoking pane is neither.
func classify(run tools.Runner, panes []paneEntry, invoking, tab string) (ours, strangers []string, err error) {
	for _, p := range panes {
		if p.TabID != tab || p.PaneID == invoking {
			continue
		}
		if p.Label == ExplorerLabel {
			ours = append(ours, p.PaneID)
			continue
		}
		if p.Label != "" {
			continue
		}
		name, err := paneForeground(run, p.PaneID)
		if err != nil {
			return nil, nil, err
		}
		if name == "yazi" {
			strangers = append(strangers, p.PaneID)
		}
	}
	return ours, strangers, nil
}

// paneForeground is the process herdr reports as owning the pane's terminal.
func paneForeground(run tools.Runner, pane string) (string, error) {
	out, err := run("herdr", "pane", "process-info", "--pane", pane)
	if err != nil {
		_, msg := errorCode(out, err)
		return "", fmt.Errorf("herdr pane process-info %s: %s", pane, msg)
	}
	var info processInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("parse `herdr pane process-info --pane %s`: %w", pane, err)
	}
	for _, p := range info.Result.ProcessInfo.ForegroundProcesses {
		name := p.Name
		if name == "" {
			name = filepath.Base(p.Argv0)
		}
		if name != "" {
			return name, nil
		}
	}
	return "", nil
}

func splitRight(run tools.Runner, ctx PaneContext, ratio float64) (string, error) {
	out, err := run("herdr", "pane", "split",
		"--pane", ctx.Pane,
		"--direction", "right",
		"--cwd", ctx.Cwd,
		"--ratio", strconv.FormatFloat(ratio, 'f', 2, 64))
	if err != nil {
		_, msg := errorCode(out, err)
		return "", fmt.Errorf("herdr pane split: %s", msg)
	}
	var res splitResult
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("parse `herdr pane split`: %w", err)
	}
	if res.Result.Pane.PaneID == "" {
		return "", errors.New("herdr pane split returned no pane id")
	}
	return res.Result.Pane.PaneID, nil
}
