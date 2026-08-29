package orca

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DX-006 (lesson 111): Orca generates two files for its Copilot integration
// and regenerates both on every install or upgrade —
//
//	~/.copilot/hooks/orca.json           registers the hooks with "timeoutSec": 5
//	~/.orca/agent-hooks/copilot-hook.ps1 POSTs each event with Invoke-WebRequest
//
// Invoke-WebRequest's cold start on Windows PowerShell 5.1 is ~4.5 s; with a
// 5 s timeout the CLI kills the hook and EVERY Copilot tool call is denied with
// "hook errored". The repair is two edits — raise every timeout below a floor,
// swap the POST for HttpWebRequest — that setup re-applies after Orca undoes
// them. It lived in scripts/orca-hook-tune.ps1; it lives here (CLI-062, #1338)
// so doctor, setup and a hand invocation share one implementation.

// DefaultHookTimeout is the floor a hook timeout is raised to.
const DefaultHookTimeout = 30

// timeoutRe extracts every "timeoutSec": N declared in orca.json.
var timeoutRe = regexp.MustCompile(`"timeoutSec"\s*:\s*(\d+)`)

// invokeWebRequestRe matches the POST line the script rewrites, capturing its
// indentation so the replacement sits where the original did.
var invokeWebRequestRe = regexp.MustCompile(`(?m)^([ \t]*)Invoke-WebRequest\b.*$`)

// httpWebRequestLines is what replaces the Invoke-WebRequest line — the exact
// block orca-hook-tune.ps1 wrote, so a box the script already tuned reads as
// clean. Each line is prefixed with the indentation of the line it replaces
// and joined with the file's own line ending. Assembled by hand rather than
// through a regexp replacement template: PowerShell's `$req` would read as a
// (missing) named group to Go's template expansion and vanish.
var httpWebRequestLines = []string{
	"$uri = 'http://127.0.0.1:' + $env:ORCA_AGENT_HOOK_PORT + '/hook/copilot'",
	"$req = [System.Net.HttpWebRequest]::Create($uri)",
	"$req.Method = 'POST'",
	"$req.ContentType = 'application/json'",
	"$req.Headers.Add('X-Orca-Agent-Hook-Token', $env:ORCA_AGENT_HOOK_TOKEN)",
	"$req.Timeout = 2000",
	"$req.ReadWriteTimeout = 2000",
	"$reqBytes = [System.Text.Encoding]::UTF8.GetBytes($body)",
	"$req.ContentLength = $reqBytes.Length",
	"$reqStream = $req.GetRequestStream()",
	"$reqStream.Write($reqBytes, 0, $reqBytes.Length)",
	"$reqStream.Close()",
	"$resp = $req.GetResponse()",
	"$resp.Close()",
}

// TimeoutBelow reports whether any "timeoutSec" in content is below min.
func TimeoutBelow(content []byte, min int) bool {
	for _, m := range timeoutRe.FindAllSubmatch(content, -1) {
		if n, err := strconv.Atoi(string(m[1])); err == nil && n < min {
			return true
		}
	}
	return false
}

// TuneTimeout raises every "timeoutSec": N below min to min, leaving a
// generous one as it is.
func TuneTimeout(content []byte, min int) []byte {
	return timeoutRe.ReplaceAllFunc(content, func(match []byte) []byte {
		m := timeoutRe.FindSubmatch(match)
		if len(m) > 1 {
			if n, err := strconv.Atoi(string(m[1])); err == nil && n < min {
				return []byte(fmt.Sprintf(`"timeoutSec": %d`, min))
			}
		}
		return match
	})
}

// ScriptUsesInvokeWebRequest reports the slow POST — the second DX-006 signal.
func ScriptUsesInvokeWebRequest(content []byte) bool {
	return regexp.MustCompile(`Invoke-WebRequest`).Match(content)
}

// TuneScript swaps the Invoke-WebRequest POST line for the HttpWebRequest
// block. ok is false when the script mentions Invoke-WebRequest but no line
// has the shape the swap knows — then nothing is changed and the caller says
// so, as the script did, rather than guessing at a different POST.
func TuneScript(content []byte) (out []byte, ok bool) {
	if !invokeWebRequestRe.Match(content) {
		return content, false
	}
	eol := "\n"
	if bytes.Contains(content, []byte("\r\n")) {
		eol = "\r\n"
	}
	out = invokeWebRequestRe.ReplaceAllFunc(content, func(line []byte) []byte {
		// `.` matches the CR of a CRLF line, so the match carries it; the
		// block re-adds the file's own ending after its last line.
		indent := string(invokeWebRequestRe.FindSubmatch(line)[1])
		trailing := ""
		if bytes.HasSuffix(line, []byte("\r")) {
			trailing = "\r"
		}
		var b strings.Builder
		for i, l := range httpWebRequestLines {
			if i > 0 {
				b.WriteString(eol)
			}
			b.WriteString(indent)
			b.WriteString(l)
		}
		b.WriteString(trailing)
		return []byte(b.String())
	})
	return out, true
}

// HookTuneReport is what one TuneHooks run found and did.
type HookTuneReport struct {
	ConfigExists, ScriptExists bool
	ConfigDrift, ScriptDrift   bool
	// ScriptUnrecognised: Invoke-WebRequest is present but its line has an
	// unknown shape; the script was left unchanged.
	ScriptUnrecognised bool
	// Backups made, one per file changed.
	Backups []string
	Changed int
}

// Drift reports whether either file still needs the repair.
func (r *HookTuneReport) Drift() bool { return r.ConfigDrift || r.ScriptDrift }

// Nothing reports the "Orca not installed for this user" case.
func (r *HookTuneReport) Nothing() bool { return !r.ConfigExists && !r.ScriptExists }

// TuneHooks measures both files and, unless check is set, repairs them: every
// timeout below minTimeout is raised, the POST is swapped, and each changed
// file is backed up beside itself first (`<file>.bak.<stamp>`) and written
// atomically. A file that does not exist is skipped; both absent is nothing
// to do.
func TuneHooks(hookConfig, hookScript string, minTimeout int, check bool, now func() time.Time) (*HookTuneReport, error) {
	rep := &HookTuneReport{}
	cfg, cfgErr := os.ReadFile(hookConfig) //nolint:gosec // caller-supplied path, the user's own hook file
	if cfgErr == nil {
		rep.ConfigExists = true
		rep.ConfigDrift = TimeoutBelow(cfg, minTimeout)
	} else if !errors.Is(cfgErr, os.ErrNotExist) {
		return rep, fmt.Errorf("read %s: %w", hookConfig, cfgErr)
	}
	scr, scrErr := os.ReadFile(hookScript) //nolint:gosec // caller-supplied path, the user's own hook file
	if scrErr == nil {
		rep.ScriptExists = true
		rep.ScriptDrift = ScriptUsesInvokeWebRequest(scr)
	} else if !errors.Is(scrErr, os.ErrNotExist) {
		return rep, fmt.Errorf("read %s: %w", hookScript, scrErr)
	}
	if check || rep.Nothing() {
		return rep, nil
	}
	if rep.ConfigDrift {
		bak, err := writeTuned(hookConfig, cfg, TuneTimeout(cfg, minTimeout), now)
		if err != nil {
			return rep, err
		}
		rep.Backups = append(rep.Backups, bak)
		rep.Changed++
		rep.ConfigDrift = false
	}
	if rep.ScriptDrift {
		res, err := TuneScriptFile(hookScript, now)
		if err != nil {
			return rep, err
		}
		if res.Unrecognised {
			rep.ScriptUnrecognised = true
		} else {
			rep.Backups = append(rep.Backups, res.Backup)
			rep.Changed++
			rep.ScriptDrift = false
		}
	}
	return rep, nil
}

// ScriptTuneResult is what TuneScriptFile did to one copilot-hook.ps1.
type ScriptTuneResult struct {
	// Changed: the POST was swapped and the file rewritten (Backup names the copy).
	Changed bool
	Backup  string
	// Unrecognised: Invoke-WebRequest is present but on a line the swap does
	// not know; the file was left byte-identical.
	Unrecognised bool
}

// TuneScriptFile applies the script half of the repair to one file — the
// entry point doctor --fix and TuneHooks share, so the two cannot drift. A
// script without Invoke-WebRequest is already tuned and untouched.
func TuneScriptFile(path string, now func() time.Time) (ScriptTuneResult, error) {
	var res ScriptTuneResult
	scr, err := os.ReadFile(path) //nolint:gosec // caller-supplied path, the user's own hook file
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}
	if !ScriptUsesInvokeWebRequest(scr) {
		return res, nil
	}
	tuned, ok := TuneScript(scr)
	if !ok {
		res.Unrecognised = true
		return res, nil
	}
	bak, err := writeTuned(path, scr, tuned, now)
	if err != nil {
		return res, err
	}
	res.Changed, res.Backup = true, bak
	return res, nil
}

// writeTuned backs the original up beside itself, then writes the tuned
// content through a temp file and a rename, so a crash mid-write leaves
// either the old file or the new one — never a truncated hook.
func writeTuned(path string, original, tuned []byte, now func() time.Time) (string, error) {
	bak := path + ".bak." + now().Format("20060102-150405")
	if err := os.WriteFile(bak, original, 0o644); err != nil { //nolint:gosec // a backup of the user's own hook file
		return "", fmt.Errorf("back up %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, tuned, 0o644); err != nil { //nolint:gosec // the user's own hook file
		return bak, fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return bak, fmt.Errorf("replace %s: %w", path, err)
	}
	return bak, nil
}
