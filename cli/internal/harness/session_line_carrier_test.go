package harness

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// driver loads the two deployed sources the way their hosts do and reports what
// each added to a session. The hosts are stood in for by the smallest fakes of
// the two calls the sources make: pi's on/sendMessage and opencode's plugin
// factory. It prints one JSON object.
const driver = `
import { pathToFileURL } from "node:url";
const [piSrc, ocSrc] = process.argv.slice(2);
const out = {};

const handlers = {};
const sent = [];
const pi = { on: (name, fn) => { handlers[name] = fn; }, sendMessage: async (m, o) => { sent.push({ ...m, options: o }); } };
(await import(pathToFileURL(piSrc))).default(pi);
await handlers.session_start({ reason: "reload" }, { cwd: process.cwd() });
const afterReload = sent.length;
await handlers.session_start({ reason: "startup" }, { cwd: process.cwd() });
out.pi = { afterReload, sent };

const mod = (await import(pathToFileURL(ocSrc))).default;
const hooks = await mod.server({ directory: process.cwd() });
const turns = [];
for (let i = 0; i < 2; i++) {
  const output = { system: ["base"] };
  await hooks["experimental.chat.system.transform"]({ sessionID: "s1" }, output);
  turns.push(output.system);
}
out.opencode = { id: mod.id, turns };
process.stdout.write(JSON.stringify(out));
`

type carrierRun struct {
	Pi struct {
		AfterReload int `json:"afterReload"`
		Sent        []struct {
			CustomType string `json:"customType"`
			Content    string `json:"content"`
			Options    struct {
				DeliverAs string `json:"deliverAs"`
			} `json:"options"`
		} `json:"sent"`
	} `json:"pi"`
	Opencode struct {
		ID    string     `json:"id"`
		Turns [][]string `json:"turns"`
	} `json:"opencode"`
}

// runDriver runs the two sources under node with PATH set to pathDirs.
func runDriver(t *testing.T, node string, pathDirs ...string) carrierRun {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	script := filepath.Join(t.TempDir(), "driver.mjs")
	if err := os.WriteFile(script, []byte(driver), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{script, filepath.Join(root, "ai", "pi", "extensions", "dotfiles-gate.ts"), filepath.Join(root, "ai", "opencode", "plugins", "dotfiles-gate.ts")}
	cmd := exec.Command(node, args...)
	cmd.Env = append(os.Environ(), "PATH="+strings.Join(pathDirs, string(os.PathListSeparator)))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node driver: %v\n%s", err, out)
	}
	var run carrierRun
	if err := json.Unmarshal(out, &run); err != nil {
		t.Fatalf("driver output %q: %v", out, err)
	}
	return run
}

// The pi extension and the opencode plugin are the carriers of the session line
// for their harnesses (HARNESS-195 AC3): run them, not just check they exist.
func TestSessionLineCarriersInjectTheBriefAndSurviveAMissingDotf(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in dotf is a shell script; the sources spawn `dotf` by name on every OS")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed; the carriers are TypeScript and run under it")
	}
	if out, err := exec.Command(node, "-e", "process.exit(typeof process.features.typescript === 'string' ? 0 : 1)").CombinedOutput(); err != nil {
		t.Skipf("this node cannot run TypeScript without a build step: %s", out)
	}

	bin := t.TempDir()
	stub := "#!/bin/sh\n[ \"$1 $2 $3 $4\" = \"mem session-start --format stdout\" ] || exit 9\nprintf 'lane `x` ACTIVE on `b`\\n'\n"
	if err := os.WriteFile(filepath.Join(bin, "dotf"), []byte(stub), 0o755); err != nil { // #nosec G306 -- a test stand-in that must be executable
		t.Fatal(err)
	}
	nodeDir := filepath.Dir(node)

	run := runDriver(t, node, bin, nodeDir)
	const line = "lane `x` ACTIVE on `b`"
	if run.Pi.AfterReload != 0 {
		t.Errorf("pi told the session on a reload")
	}
	if len(run.Pi.Sent) != 1 || run.Pi.Sent[0].Content != line || run.Pi.Sent[0].Options.DeliverAs != "nextTurn" {
		t.Errorf("pi sent %+v, want the line once, for the next turn", run.Pi.Sent)
	}
	if len(run.Opencode.Turns) != 2 || strings.Join(run.Opencode.Turns[0], "|") != "base|"+line || strings.Join(run.Opencode.Turns[1], "|") != "base|"+line {
		t.Errorf("opencode system prompts = %v, want base plus the line on every turn", run.Opencode.Turns)
	}

	// No dotf on PATH: nothing is added and nothing throws.
	missing := runDriver(t, node, nodeDir)
	if len(missing.Pi.Sent) != 0 || len(missing.Opencode.Turns) != 2 || len(missing.Opencode.Turns[0]) != 1 {
		t.Errorf("a missing dotf changed a session: %+v", missing)
	}
}
