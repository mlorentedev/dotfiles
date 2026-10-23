package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Agent records are rendered like skill records, one harness-native file per
// agents.deploy[] entry, but with the model and capability lines resolved for
// that harness. This is the Go form of compile-harness.sh's deploy_agents,
// render_agent, agent_model_line and agent_capability_line (CLI-026). The
// presence region is `dotf harness presence`, not this.

// RenderAgentMD is the one render kind agents.deploy[] declares.
const RenderAgentMD = "agent-md"

// AgentDeployTarget is one manifest agents.deploy[] entry.
type AgentDeployTarget struct {
	Agent  string `json:"agent"`
	Render string `json:"render"`
	Dir    string `json:"dir"`
}

// AgentManifest is the part of the manifest's agents block the render reads.
type AgentManifest struct {
	VaultSubpath string              `json:"vault_subpath"`
	RecordDir    string              `json:"record_dir"`
	Deploy       []AgentDeployTarget `json:"deploy"`
}

// LoadAgents reads the manifest's agents block; a manifest without one yields
// nil and no error.
func LoadAgents(manifestPath string) (*AgentManifest, error) {
	raw, err := os.ReadFile(manifestPath) //nolint:gosec // repo-relative, fixed name
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", ManifestFile, err)
	}
	var m struct {
		Agents *AgentManifest `json:"agents"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", ManifestFile, err)
	}
	if m.Agents == nil {
		return nil, nil
	}
	if m.Agents.RecordDir == "" {
		return nil, fmt.Errorf("%s: agents.record_dir is empty", ManifestFile)
	}
	for i, t := range m.Agents.Deploy {
		if t.Agent == "" || t.Dir == "" || t.Render != RenderAgentMD {
			return nil, fmt.Errorf("%s: agents.deploy[%d] needs agent, dir and render %q", ManifestFile, i, RenderAgentMD)
		}
	}
	return m.Agents, nil
}

// agentNativeKey is the frontmatter an agent-md render keeps: the native
// required subset. Unlike a skill render, a kept key's continuation lines go.
var agentNativeKey = regexp.MustCompile(`^(name|description):`)

// RenderAgentRecord renders one agent record: provenance after the first
// fence, name and description, then the resolved model and capability lines
// (either may be empty) before the second fence, and the body verbatim.
func RenderAgentRecord(record []byte, srcPath, modelLine, capLine string) string {
	record = normalizeLF(record)
	sha := PresenceSHA(string(record))
	var b strings.Builder
	emit := func(l string) { b.WriteString(l); b.WriteByte('\n') }
	fences := 0
	for _, l := range recordLines(record) {
		if fences < 2 && fenceLine.MatchString(l) {
			fences++
			if fences == 1 {
				emit(l)
				emit("generated: true")
				emit("generated_from: " + srcPath)
				emit("generated_sha: " + sha)
				continue
			}
			for _, extra := range []string{modelLine, capLine} {
				if extra != "" {
					emit(extra)
				}
			}
			emit(l)
			continue
		}
		if fences == 1 {
			if agentNativeKey.MatchString(l) {
				emit(l)
			}
			continue
		}
		emit(l)
	}
	return b.String()
}

// agentResolver answers the model and capability lines for a record on one
// harness, reading each map once and only when a record asks for it.
type agentResolver struct {
	repoRoot         string
	models, caps     map[string]any
	modelErr, capErr error
}

// RecordModelTier is the neutral tier an agent record declares in its
// frontmatter `model:` line, read as the render reads it; "" when it declares
// none.
func RecordModelTier(record []byte) string {
	return skillField(record, "model")
}

// modelLine is `model: <id>` for the record's tier, or "" when it declares
// none. A tier the map cannot resolve is an error, never a missing line.
func (r *agentResolver) modelLine(record []byte, agent string) (string, error) {
	tier := RecordModelTier(record)
	if tier == "" {
		return "", nil
	}
	if r.models == nil && r.modelErr == nil {
		r.models, r.modelErr = LoadModelMap(r.repoRoot)
	}
	if r.modelErr != nil {
		return "", r.modelErr
	}
	id, err := ResolveTier(r.models, tier, agent)
	if err != nil {
		return "", err
	}
	if strings.ContainsAny(id, " \t\r\n") {
		return "", fmt.Errorf("%s gives tier %q on harness %q the id %q — a model id is one token with no whitespace", ModelMapFile, tier, agent, id)
	}
	return "model: " + id, nil
}

// blockCapabilities matches `capabilities:` with its value on the lines below.
var blockCapabilities = regexp.MustCompile(`^capabilities:[[:space:]]*$`)

// capabilityLine is the harness-native capability line for the record's
// inline `capabilities: [...]`, or "" when it declares none. The block form is
// refused, and so is an empty list: rendering no line for an allow-list field
// grants the harness's whole default tool set instead of denying it. The
// unsupported capabilities the harness drops are returned so the caller can say
// so.
func (r *agentResolver) capabilityLine(record []byte, agent string) (line string, unsupported []string, err error) {
	caps := skillField(record, "capabilities")
	if caps == "" {
		if frontmatterHas(record, blockCapabilities) {
			return "", nil, errors.New("declares `capabilities:` in YAML block style, which the render does not read; " +
				"use the inline form `capabilities: [read, shell]`, because no capability line would GRANT the full default tool set")
		}
		return "", nil, nil
	}
	caps = strings.ReplaceAll(strings.TrimSuffix(strings.TrimPrefix(caps, "["), "]"), " ", "")
	if caps == "" {
		return "", nil, errors.New("declares an empty `capabilities: []`, which the render cannot express: " +
			"no capability line would GRANT the full default tool set, not deny it; drop the field or name what the agent needs")
	}
	if r.caps == nil && r.capErr == nil {
		r.caps, r.capErr = LoadCapabilityMap(r.repoRoot)
	}
	if r.capErr != nil {
		return "", nil, r.capErr
	}
	requested := strings.Split(caps, ",")
	if line, err = ResolveCapabilities(r.caps, requested, agent); err != nil {
		return "", nil, err
	}
	unsupported, _ = UnsupportedFor(r.caps, requested, agent)
	return line, unsupported, nil
}

// frontmatterHas reports whether a line of the record's frontmatter matches re.
func frontmatterHas(record []byte, re *regexp.Regexp) bool {
	fences := 0
	for _, l := range recordLines(normalizeLF(record)) {
		if fenceLine.MatchString(l) {
			if fences++; fences >= 2 {
				return false
			}
			continue
		}
		if fences == 1 && re.MatchString(l) {
			return true
		}
	}
	return false
}

// DeployAgents renders every agent record to each agents.deploy[] target under
// home that it targets, as <dir>/<name>.md. An output that already holds its
// render is not rewritten, and a symlink there is replaced by a copy. A record
// whose model or capabilities cannot be resolved is an error for that record
// only, reported after every other agent is deployed, and its previous output
// is left as it was.
func DeployAgents(repoRoot, home string, dryRun bool) ([]SkillOutcome, error) {
	m, err := LoadAgents(filepath.Join(repoRoot, filepath.FromSlash(ManifestFile)))
	if err != nil || m == nil {
		return nil, err
	}
	recDir := filepath.Join(repoRoot, filepath.FromSlash(m.RecordDir))
	names, err := recordNamesOf(recDir, "AGENT.md")
	if err != nil {
		return nil, err
	}
	r := &agentResolver{repoRoot: repoRoot}
	var out []SkillOutcome
	var errs []error
	for _, t := range m.Deploy {
		for _, name := range names {
			o, err := deployAgent(r, t, recDir, name, filepath.Join(home, filepath.FromSlash(t.Dir)), m.VaultSubpath+"/"+name+"/AGENT.md", dryRun)
			if err != nil {
				errs = append(errs, fmt.Errorf("agent %s for %s: %w", name, t.Agent, err))
				continue
			}
			if o != nil {
				out = append(out, *o)
			}
		}
	}
	return out, errors.Join(errs...)
}

// deployAgent renders one record for one target; nil means the record does not
// target that harness.
func deployAgent(r *agentResolver, t AgentDeployTarget, recDir, name, base, srcPath string, dryRun bool) (*SkillOutcome, error) {
	record, err := os.ReadFile(filepath.Join(recDir, name, "AGENT.md")) //nolint:gosec // a record under the manifest's record_dir
	if err != nil {
		return nil, err
	}
	if targeted, err := SkillTargetsAgent(record, t.Agent); err != nil || !targeted {
		return nil, err
	}
	model, err := r.modelLine(record, t.Agent)
	if err != nil {
		return nil, err
	}
	capLine, unsupported, err := r.capabilityLine(record, t.Agent)
	if err != nil {
		return nil, err
	}
	outp := filepath.Join(base, name+".md")
	wrote, err := writeIfChanged(outp, []byte(RenderAgentRecord(record, srcPath, model, capLine)), 0o644, dryRun)
	if err != nil {
		return nil, err
	}
	o := &SkillOutcome{Agent: t.Agent, Path: outp, Status: SkillCurrent}
	if wrote {
		o.Status = SkillDeployed
	}
	if len(unsupported) > 0 {
		o.Reason = t.Agent + " declares no native equivalent for " + strings.Join(unsupported, ", ") + " — omitted from the value, not granted"
	}
	return o, nil
}
