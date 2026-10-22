package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const agentRecord = `---
name: demo
description: a demo persona
kind: invocable
model: mid
capabilities: [read, search]
skills:
  - handoff
---

# Demo

---

After a rule.
`

func TestRenderAgentRecord(t *testing.T) {
	got := RenderAgentRecord([]byte(agentRecord), "src", "model: m", "tools: Read")
	want := "---\ngenerated: true\ngenerated_from: src\ngenerated_sha: " + PresenceSHA(agentRecord) + "\n" +
		"name: demo\ndescription: a demo persona\nmodel: m\ntools: Read\n---\n\n# Demo\n\n---\n\nAfter a rule.\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if bare := RenderAgentRecord([]byte(agentRecord), "src", "", ""); strings.Contains(bare, "model:") || strings.Contains(bare, "tools:") {
		t.Errorf("empty lines were rendered:\n%s", bare)
	}
}

func TestRenderAgentRecord_CRLFRendersTheLFBytes(t *testing.T) {
	crlf := strings.ReplaceAll(agentRecord, "\n", "\r\n")
	if RenderAgentRecord([]byte(crlf), "s", "", "") != RenderAgentRecord([]byte(agentRecord), "s", "", "") {
		t.Error("a CRLF record renders different bytes from its LF form")
	}
}

// agentsRepo is a checkout with the committed model and capability maps and
// one claude agent target.
func agentsRepo(t *testing.T, records map[string]string) (repo, home string) {
	t.Helper()
	repo, home = t.TempDir(), t.TempDir()
	for _, f := range []string{ModelMapFile, ModelMapSchemaFile, CapabilityMapFile, CapabilityMapSchemaFile} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "..", f)) //nolint:gosec // committed maps
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(repo, f), string(raw))
	}
	writeFile(t, filepath.Join(repo, "harness", "manifest.json"), `{"agents":{
		"vault_subpath":"00_meta/agents/definitions","record_dir":"harness/agents",
		"deploy":[{"agent":"claude","render":"agent-md","dir":".claude/agents"}]}}`)
	for name, body := range records {
		writeFile(t, filepath.Join(repo, "harness", "agents", name, "AGENT.md"), body)
	}
	return repo, home
}

func TestDeployAgents_RendersWithResolvedLinesThenConverges(t *testing.T) {
	repo, home := agentsRepo(t, map[string]string{"demo": agentRecord})
	out, err := DeployAgents(repo, home, false)
	if err != nil || countStatus(out, SkillDeployed) != 1 {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	got, _ := os.ReadFile(filepath.Join(home, ".claude", "agents", "demo.md"))
	models, _ := LoadModelMap(repo)
	id, _ := ResolveTier(models, "mid", "claude")
	caps, _ := LoadCapabilityMap(repo)
	line, _ := ResolveCapabilities(caps, []string{"read", "search"}, "claude")
	if !strings.Contains(string(got), "\nmodel: "+id+"\n"+line+"\n---\n") {
		t.Errorf("the resolved lines are missing:\n%s", got)
	}
	again, err := DeployAgents(repo, home, false)
	if err != nil || countStatus(again, SkillCurrent) != 1 {
		t.Errorf("second run: %+v, %v", again, err)
	}
}

func TestDeployAgents_ABadRecordFailsAloneAndKeepsItsOutput(t *testing.T) {
	block := strings.Replace(agentRecord, "capabilities: [read, search]", "capabilities:\n  - read", 1)
	empty := strings.Replace(agentRecord, "capabilities: [read, search]", "capabilities: [ ]", 1)
	badTier := strings.Replace(agentRecord, "model: mid", "model: nonesuch", 1)
	repo, home := agentsRepo(t, map[string]string{"a-good": agentRecord, "block": block, "empty": empty, "tier": badTier})
	previous := filepath.Join(home, ".claude", "agents", "tier.md")
	writeFile(t, previous, "previous definition\n")
	out, err := DeployAgents(repo, home, false)
	if err == nil || !strings.Contains(err.Error(), "block style") || !strings.Contains(err.Error(), "agent empty") || !strings.Contains(err.Error(), "nonesuch") {
		t.Fatalf("error: %v", err)
	}
	if countStatus(out, SkillDeployed) != 1 {
		t.Errorf("the good record did not deploy: %+v", out)
	}
	if got, _ := os.ReadFile(previous); string(got) != "previous definition\n" {
		t.Error("a failed record overwrote its previous output")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "block.md")); err == nil {
		t.Error("a record declaring block capabilities was deployed without its allow-list")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "agents", "empty.md")); err == nil {
		t.Error("a record declaring an empty allow-list was deployed with the default tool set")
	}
}

func TestDeployAgents_NoTierNoCapabilitiesRendersNoLines(t *testing.T) {
	plain := "---\nname: plain\ndescription: d\n---\nbody\n"
	repo, home := agentsRepo(t, map[string]string{"plain": plain})
	if _, err := DeployAgents(repo, home, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(home, ".claude", "agents", "plain.md"))
	if string(got) != RenderAgentRecord([]byte(plain), "00_meta/agents/definitions/plain/AGENT.md", "", "") {
		t.Errorf("got:\n%s", got)
	}
}

func TestDeployAgents_DryRunWritesNothing(t *testing.T) {
	repo, home := agentsRepo(t, map[string]string{"demo": agentRecord})
	out, err := DeployAgents(repo, home, true)
	if err != nil || countStatus(out, SkillDeployed) != 1 {
		t.Fatalf("outcomes: %+v, %v", out, err)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("a dry run wrote %d entries", len(entries))
	}
}

func TestLoadAgents_RejectsAnUnknownRender(t *testing.T) {
	p := filepath.Join(t.TempDir(), "manifest.json")
	writeFile(t, p, `{"agents":{"record_dir":"r","deploy":[{"agent":"a","render":"skill","dir":"d"}]}}`)
	if _, err := LoadAgents(p); err == nil {
		t.Error("an unknown render kind was accepted")
	}
}
