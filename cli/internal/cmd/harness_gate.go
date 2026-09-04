package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mlorentedev/dotfiles/cli/internal/harness"
)

// gateExitBlock is the exit code every harness reads as "stop this tool call".
// Claude documents 2 for a blocking hook; the emitted pi and opencode wrappers
// translate it into their own primitive.
const gateExitBlock = 2

// newHarnessGateCmd is the binding primitive for HARNESS-045 (#561).
//
// THIS COMMAND IS THE AGNOSTIC SEAM. Every harness's hook is a thin wrapper
// around one line that calls it, so the persona logic exists once, in Go,
// testable with no harness installed. The alternative — a per-harness adapter
// carrying its own logic — is a second implementation that will drift, which is
// the failure this repository has catalogued repeatedly.
//
// IT EXITS 0 OR 2 AND NOTHING ELSE. The harnesses disagree about what a hook
// *error* means: claude's hooks must never crash a session (fail-open), while pi
// documents that `tool_call` errors block the tool (fail-safe, fail-closed). A
// command that could exit 1 would therefore mean opposite things on two
// harnesses. Every path here resolves to an explicit decision instead, so the
// harness never sees an error and its error semantics never apply.
//
// The single non-nil return is tagged with exit 2 explicitly. An untagged error
// would resolve to 1 through ExitCode — the one status this command must never
// produce — so nothing here may return a bare error.
//
// AND EVERY PATH LEAVES A DURABLE RECORD. An exit status is not an audit trail:
// the harness consumes and discards it, `warn` and `allow` share it, and a
// PreToolUse hook's stderr on exit 0 is not persisted anywhere the session can
// be asked about later. Three paths below return before any persona is loaded,
// and before the journal they left nothing at all — so a harness whose payload
// this cannot parse looked exactly like a harness with nothing to enforce.
func newHarnessGateCmd() *cobra.Command {
	var (
		harnessName string
		role        string
		repoRoot    string
		stateDir    string
	)

	cmd := &cobra.Command{
		Use:   "gate",
		Short: "Decide whether a tool call may proceed under a persona's forced skills",
		Long: `gate reads a harness's tool-call event on stdin and answers with an exit code:
0 to allow, 2 to block. It is what a persona's "enforced by hook" claim compiles
to, and it is the same binary on every harness.

A skill declared 'enforce: block' that has not been consumed this session blocks
the call. One declared 'enforce: warn' is reported on stderr and allowed. A skill
carrying no declared severity is neither — it is a migration gap, surfaced by
` + "`dotf doctor`" + `, not noise on every tool call.

Invoking a skill is never blocked: forbidding it would deadlock the session.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload, _ := io.ReadAll(cmd.InOrStdin())
			if stateDir == "" {
				stateDir = defaultGateStateDir()
			}
			// One closure, used by every exit below, because the requirement is
			// that NO path returns without leaving a record. Writing it at each
			// `return` is how the three early paths came to leave none: each was
			// individually correct, and the omission existed only in aggregate —
			// the same shape as the defect this spec started from.
			record := func(scope string, rec harness.DecisionRecord) {
				rec.Harness = harnessName
				rec.Scope = scope
				_ = harness.RecordDecision(harness.DecisionPath(stateDir, scope), rec)
			}

			call, understood := normaliseToolCall(harnessName, payload)
			if !understood {
				record(harness.UnparsedScope, harness.DecisionRecord{
					Outcome:      harness.OutcomePayloadUnrecognised,
					Allowed:      true,
					Reason:       "payload not recognised",
					PayloadBytes: len(payload),
				})
				// MEASURED 2026-08-26: this branch is why it exists. Without
				// it a malformed payload produced a zero ToolCall, Decide saw
				// a valid persona with nothing consumed, and the gate BLOCKED
				// — the opposite of the documented contract, and invisible to
				// unit tests that call Decide directly. A gate that blocks on
				// input it cannot read blocks on every harness upgrade.
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "[gate] allow: payload not recognised")
				return nil
			}
			// Scoped to the acting persona when the harness names one: a subagent
			// reuses the parent session id, so keying by session alone would let one
			// persona's skill runs satisfy another's gate.
			scope := call.ConsumptionScope()
			statePath := harness.StatePath(stateDir, scope)

			// Every record from here carries what the harness said was acting.
			// THIS IS WHAT CONVERTS agent_type FROM INFERRED TO MEASURED: it is
			// read straight off the payload, before any persona lookup, so even
			// the skill path below — which returns before that lookup — leaves
			// evidence that the field arrived and of what it contained.
			base := harness.DecisionRecord{
				Session:   call.SessionID,
				AgentType: call.AgentType,
				AgentID:   call.AgentID,
				Tool:      call.Tool,
			}

			// HARNESS-109: if THIS call is a dispatch, remember what it
			// dispatched before deciding anything about it. PreToolUse is
			// synchronous, so this write completes before the child exists —
			// the ordering the whole mechanism rests on, and the reason it
			// happens here rather than on some later event. The error is
			// ignored for RecordConsumed's reason: losing the entry costs
			// enforcement for one subagent, while failing here would block a
			// session over a full disk.
			if call.DispatchType != "" {
				_ = harness.RecordDispatch(
					harness.DispatchPath(stateDir, call.SessionID),
					call.DispatchName, call.DispatchType)
			}

			// A skill invocation is the act the gate exists to require: record
			// it and get out of the way. Recording failures are ignored on
			// purpose — losing the record costs a redundant skill run, while
			// failing here would block a session over a full disk.
			if call.Skill != "" {
				_ = harness.RecordConsumed(statePath, call.Skill)
				rec := base
				rec.Skill = call.Skill
				rec.Outcome = harness.OutcomeSkillConsumed
				rec.Allowed = true
				rec.Reason = "skill invocation recorded"
				record(scope, rec)
				return nil
			}
			if call.IsSkillTool {
				// The tool IS the skill primitive but its argument was
				// unreadable. There is nothing to record, and blocking would
				// deadlock the session on the one action that could satisfy the
				// gate — a well-formed payload with a missing argument, not a
				// parse failure. Raised by the reviewer on #1272.
				rec := base
				rec.Outcome = harness.OutcomeSkillUnnamed
				rec.Allowed = true
				rec.Reason = "skill invocation with no readable name"
				record(scope, rec)
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "[gate] allow: skill invocation with no readable name")
				return nil
			}

			requested := effectiveRole(role, call.AgentType)
			// Whether the operator NAMED the role decides which source is
			// authoritative below. A --role flag is a human's explicit
			// instruction and outranks anything the payload implies; a
			// payload-derived role is a caller-supplied string the dispatch map
			// can correct.
			fromFlag := strings.TrimSpace(role) != ""
			dispatched := harness.LoadDispatched(harness.DispatchPath(stateDir, call.SessionID))
			persona, resolution := loadGatePersona(cmd.ErrOrStderr(), repoRoot, requested, dispatched, fromFlag)
			result := harness.Decide(harness.GateInput{
				Persona:  persona,
				Call:     call,
				Consumed: harness.LoadConsumed(statePath),
			})

			rec := base
			rec.RoleRequested = requested
			rec.Reason = result.Reason
			rec.Warned = result.Warned
			rec.Missing = result.Missing
			rec.Allowed = result.Decision != harness.Block
			if persona != nil {
				rec.RoleResolved = persona.Name
			}
			switch {
			case resolution == roleUnresolved:
				// Checked BEFORE the decision, because this call allowed and a
				// healthy allow looks identical to it. The distinction is the
				// whole reason AC5 names this case: enforcement was off, and
				// nothing outside this record says so.
				rec.Outcome = harness.OutcomeRoleUnresolved
			case resolution == roleNotAsked || resolution == roleNotAPersona:
				// Deliberately one case: "nobody with forced skills is acting"
				// is the same fact whether nobody was asked for or a witnessed
				// dispatch turned out to be a built-in agent. RoleRequested
				// keeps the raw name, so the journal still separates them
				// without spending a ninth Outcome.
				rec.Outcome = harness.OutcomeNoRole
			case result.Decision == harness.Block:
				rec.Outcome = harness.OutcomeBlock
			case len(result.Warned) > 0:
				rec.Outcome = harness.OutcomeWarn
			default:
				rec.Outcome = harness.OutcomeAllow
			}
			// Written before the return below, not after: the block path used to
			// call os.Exit, which runs no defers, so a record deferred past it
			// would be the one decision never written — the only one anybody
			// would go looking for.
			record(scope, rec)

			for _, w := range result.Warned {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[gate] warn: %s not consumed\n", w)
			}
			if result.Decision == harness.Block {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "[gate] blocked: %s\n", result.Reason)
				// withExitCode rather than os.Exit, which is what this called
				// before. The status is identical — main resolves it through
				// ExitCode — but os.Exit kills the test runner, so the ONE path
				// that matters most could never be driven end to end in process.
				// The command sets SilenceErrors, so main prints nothing extra
				// and the reason a human reads is still the line above.
				return withExitCode(gateExitBlock, fmt.Errorf("%s", result.Reason))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&harnessName, "harness", "", "harness emitting the event (claude|pi|opencode)")
	cmd.Flags().StringVar(&role, "role", "", "persona in scope")
	cmd.Flags().StringVar(&repoRoot, "repo-root", "", "root containing harness/agents (default: the deploy dir, else the checkout)")
	cmd.Flags().StringVar(&stateDir, "state-dir", "", "where per-session consumption is recorded")
	return cmd
}

// loadGatePersona resolves the persona, returning nil rather than an error.
//
// Nil means Allow. A misconfigured or missing record must not block every tool
// call in a session — that is the "normal state is red" failure, and it is worse
// than the gate being absent, because an operator disables a noisy gate and then
// has nothing.
//
// BUT A ROLE THAT WAS ASKED FOR AND DID NOT RESOLVE IS SAID OUT LOUD. Failing
// open is the right decision and silence is not: `--role reviewr` used to exit 0
// with no output, so a typo, a renamed record or a repo-root that resolved
// elsewhere disabled enforcement entirely while every session reported health.
// That is the guard-fails-open shape lesson 219 named, one level down — and it
// is the failure this whole gate exists to prevent, committed inside the gate.
// warn takes an io.Writer rather than reaching for os.Stderr so the message is
// assertable; a diagnostic nothing can test is the same silence with extra steps.
// roleResolution separates the two ways loadGatePersona returns nil.
//
// They were indistinguishable to the caller, and both allow, so from outside
// "no persona was asked for" and "a persona was asked for and enforcement is
// therefore OFF" produced the same silence. The stderr line tells them apart for
// a human reading a live terminal; nothing told them apart afterwards, which is
// the gap AC5 names by calling out `role did not resolve` explicitly.
type roleResolution int

const (
	// roleNotAsked: the caller declared no persona. A main-thread call.
	roleNotAsked roleResolution = iota
	// roleUnresolved: a persona was named and could not be loaded.
	roleUnresolved
	// roleResolved: a persona is in scope.
	roleResolved
	// roleNotAPersona: the gate WITNESSED this dispatch and knows what it is —
	// a built-in agent (`general-purpose`, `Explore`, `Plan`) that no persona
	// governs by design.
	//
	// It allows, quietly, and records OutcomeNoRole: it is the same FACT as a
	// main-thread call — nobody with forced skills is acting — so it spends no
	// slot in the Outcome vocabulary, which is closed and pinned by tests on
	// purpose. It is a separate resolution nonetheless because the two differ
	// in the code's terms: one was never asked, the other was asked and
	// correctly answered "nobody". The raw name survives in RoleRequested, so
	// the journal can still tell them apart.
	//
	// Before HARNESS-109 this case was reported as roleUnresolved with
	// "ENFORCEMENT IS OFF" on stderr — a misclassification that made 271 of 274
	// records look like faults and left the genuine faults unfindable among
	// them.
	roleNotAPersona
)

// loadGatePersona answers "who is acting" from two sources, and WHICH ONE WINS
// is the whole of #1434 plus its shadow case.
//
// THE DISPATCH MAP OUTRANKS THE PAYLOAD'S OWN STRING. `agent_type` is
// caller-supplied: it carries the dispatch's NAME when one was given, and a name
// is unconstrained — it may happen to equal a persona's. So looking the payload
// string up in the roster FIRST resolves `Agent(name: "reviewer",
// subagent_type: "builder")` to `reviewer`, enforcing the wrong persona's skills
// while the journal reads healthy. Found by the independent review on #1471,
// with a reproduction; nothing in the 274 observed records shadows a persona
// today, which is exactly why it would have been discovered late. The map does
// not have this weakness: it was written from the dispatch's own
// `subagent_type`, so when it has an answer, that answer is what the caller
// DECLARED rather than what it happened to name.
//
// A `--role` FLAG STILL WINS OVER BOTH. It is a human's explicit instruction and
// the map is not about it — deliberately, because a blunt map-first reorder
// would let a session's dispatch history override the operator's own override.
//
// Resolution, in order:
//
//  1. `--role` given: the roster, and nothing else. Unresolved stays loud.
//  2. The session's map witnessed this identity: its recorded type is the
//     answer. A persona → that persona; not a persona → nobody, quietly.
//  3. Not witnessed: the roster, as before. This is what still resolves a
//     dispatch the gate never saw whose agent_type happens to name a persona.
//  4. Neither: genuinely unknown, and loud.
func loadGatePersona(warn io.Writer, repoRoot, role string, dispatched map[string]string, fromFlag bool) (*harness.Persona, roleResolution) {
	if strings.TrimSpace(role) == "" {
		// Not asked for: the caller declared no persona, so there is nothing to
		// fail to find. Distinct from the case below, and NOT worth a line on
		// every tool call.
		return nil, roleNotAsked
	}
	if repoRoot == "" {
		repoRoot = os.Getenv("DOTFILES_DIR")
		if repoRoot == "" {
			repoRoot = filepath.Join(os.Getenv("HOME"), ".dotfiles")
		}
	}

	lookup := role
	if !fromFlag {
		if trueType, witnessed := dispatched[role]; witnessed {
			lookup = trueType
		}
	}

	agentsDir := filepath.Join(repoRoot, "harness", "agents")
	path := filepath.Join(agentsDir, lookup, "AGENT.md")
	p, err := harness.LoadPersona(path)
	if err == nil {
		return p, roleResolved
	}

	// A RECORD THAT EXISTS AND WOULD NOT LOAD IS A FAULT, NOT A BUILT-IN.
	//
	// The second finding of the same review: concluding "this is not a persona"
	// from a failed load conflates `general-purpose` — which has no record BY
	// DESIGN — with `reviewer` whose AGENT.md a bad merge just broke. The first
	// is correctly quiet; the second is the exact fault this spec exists to make
	// findable, and quieting it would hide enforcement being off behind the very
	// cleanup that was supposed to surface it.
	//
	// Membership is decided by the DIRECTORY, not by whether the file parses,
	// so a corrupt record still counts as a persona and stays loud.
	// LoadPersonas is not reusable here: it fails wholesale on one bad record,
	// so it cannot answer a question about a single name.
	if _, statErr := os.Stat(filepath.Join(agentsDir, lookup)); statErr == nil {
		_, _ = fmt.Fprintf(warn, "[gate] allow: role %q has a record that did not load (%s): %v — ENFORCEMENT IS OFF for this call\n",
			lookup, path, err)
		return nil, roleUnresolved
	}

	if lookup != role || dispatched[role] != "" {
		// Witnessed, and its true type has no record directory at all: a
		// built-in agent. Nobody governs it, and that is a correct answer.
		return nil, roleNotAPersona
	}

	_, _ = fmt.Fprintf(warn, "[gate] allow: role %q did not resolve (%s): %v — ENFORCEMENT IS OFF for this call\n",
		role, path, err)
	return nil, roleUnresolved
}

// effectiveRole picks the persona in scope: the operator's --role when given,
// otherwise whatever the harness said was acting.
//
// THE FLAG WINS ON PURPOSE, and it is the smaller of the two jobs. The manifest
// emits one hook for the whole harness, so a static --role in it would pin every
// session to a single persona — which is why the flag alone left the gate inert:
// nothing passed one, `loadGatePersona` got "", and `Decide` returned "no persona
// in scope" for all 35 skills. The payload is the real source. The flag survives
// as an override for testing and for a harness that reports no agent at all.
func effectiveRole(flag, fromPayload string) string {
	if r := strings.TrimSpace(flag); r != "" {
		return r
	}
	return strings.TrimSpace(fromPayload)
}

func defaultGateStateDir() string {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "dotfiles")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "state", "dotfiles")
}

// commandHookPayload is the JSON a COMMAND hook receives on stdin.
//
// Two harnesses use this family. Claude sends it to `PreToolUse`; **agy sends it
// to `BeforeTool`** — measured 2026-08-26, `~/.gemini/settings.json` carries
// hooks in Claude's exact `{"hooks":[{"type":"command",...}]}` shape and declares
// `BeforeAgent`, `AfterAgent`, `BeforeTool`, `AfterTool`. So agy is NOT
// presence-only as #561 and ADR-027 assumed; it has a tool gate, and it is the
// cheapest harness to add after claude rather than a deferred one.
//
// agy's exact FIELD names are unverified — no agy payload has been captured — so
// a mismatch degrades to "not understood", which allows. The gate never blocks
// on a guess.
// AgentType and AgentID are how the gate learns WHICH persona is acting, and
// they are the reason no session-state mechanism was built for it. Claude
// documents both on every hook event fired inside a subagent: `agent_type`
// carries the dispatched agent's name (`reviewer`), `agent_id` a unique id for
// that invocation. A MAIN-THREAD call carries NEITHER — absence means "no
// persona in scope", which is not an error and is exactly the pre-existing
// allow.
//
// DOCUMENTED, NOT YET MEASURED on this box: the gate is not live in any deployed
// settings file here, so no real payload has been captured. The design is built
// to make that acceptable — a wrong or renamed field yields an empty AgentType,
// hence a nil persona, hence Allow: the behaviour before this existed. A guess
// can therefore cost enforcement, never a blocked session. Confirm it by
// observing `[gate] warn` lines on a real dispatch before any skill is promoted
// to `enforce: block`.
type commandHookPayload struct {
	SessionID string         `json:"session_id"`
	ToolName  string         `json:"tool_name"`
	ToolInput map[string]any `json:"tool_input"`
	AgentType string         `json:"agent_type"`
	AgentID   string         `json:"agent_id"`
}

// canonicalPayload is the shape THIS REPOSITORY's own generated wrappers emit.
//
// pi and opencode run no command hook at all. pi's `tool_call` is an in-process
// TypeScript handler returning `{block: true, reason}`; opencode's blocking
// primitive is `permission.ask`, whose `output.status` accepts `"deny"`. Its
// `tool.execute.before` — the event #561 and ADR-027 name as opencode's gate —
// is typed `(input, output: {args}) => Promise<void>` and can only mutate
// arguments. **It cannot deny.** Measured against the installed
// `@opencode-ai/plugin` type definitions.
//
// Because the wrapper for those two is generated here, it emits ONE shape rather
// than each harness's own. That deletes two per-harness parsers instead of
// adding them: the only payload the gate must adapt to is the one it did not
// author.
type canonicalPayload struct {
	SessionID string `json:"session"`
	Tool      string `json:"tool"`
	Skill     string `json:"skill"`
}

// normaliseToolCall turns a payload into the neutral shape and reports whether
// it understood it at all.
//
// The `false` return is not defensive decoration. Without it, garbage on stdin
// produced a zero ToolCall, Decide saw a valid persona with nothing consumed,
// and the gate blocked every call — the opposite of the contract, measured and
// fixed 2026-08-26. A gate that blocks on input it cannot read blocks on every
// harness upgrade.
func normaliseToolCall(harnessName string, payload []byte) (harness.ToolCall, bool) {
	if len(payload) == 0 {
		return harness.ToolCall{}, false
	}
	switch harnessName {
	case "pi", "opencode":
		var p canonicalPayload
		if json.Unmarshal(payload, &p) != nil || p.Tool == "" {
			return harness.ToolCall{}, false
		}
		return harness.ToolCall{SessionID: p.SessionID, Tool: p.Tool, Skill: strings.TrimSpace(p.Skill), IsSkillTool: isSkillTool(p.Tool)}, true
	default:
		var p commandHookPayload
		if json.Unmarshal(payload, &p) != nil || p.ToolName == "" {
			return harness.ToolCall{}, false
		}
		name, agentType := dispatchArgs(p.ToolName, p.ToolInput)
		return harness.ToolCall{
			SessionID:    p.SessionID,
			Tool:         p.ToolName,
			Skill:        skillArg(p.ToolName, p.ToolInput),
			IsSkillTool:  isSkillTool(p.ToolName),
			AgentType:    strings.TrimSpace(p.AgentType),
			AgentID:      strings.TrimSpace(p.AgentID),
			DispatchName: name,
			DispatchType: agentType,
		}, true
	}
}

// skillArg extracts the skill name when the tool IS the harness's skill
// primitive. Matched case-insensitively on the tool name because the three
// harnesses spell it differently and none of them promises stability.
// isSkillTool reports that the tool IS the harness's skill primitive, whatever
// its arguments turned out to say. The gate must never block it: doing so
// forbids the only action that could satisfy the gate.
func isSkillTool(tool string) bool {
	return strings.EqualFold(strings.TrimSpace(tool), "skill")
}

// isDispatchTool reports that the tool IS the harness's subagent-dispatch
// primitive — the call whose arguments say which persona a subagent will be.
//
// Two spellings are matched because the primitive was renamed: `Task` in earlier
// Claude Code releases, `Agent` in the ones this repository runs (all 6 dispatch
// records in the journal read `Agent`). Accepting both costs nothing, and
// missing the one in use costs exactly the enforcement this spec restores —
// silently, since an unrecognised dispatch writes no map entry and the gate
// simply behaves as it did before.
func isDispatchTool(tool string) bool {
	t := strings.TrimSpace(tool)
	return strings.EqualFold(t, "Agent") || strings.EqualFold(t, "Task")
}

// dispatchArgs extracts the name and true type from a dispatch's arguments.
//
// THESE ARE THE ONLY TOOL-INPUT VALUES THE GATE READS besides the skill name,
// and both are schema-bounded identifiers rather than free text — the caller's
// own tool constrains them, and RecordDispatch validates them again before
// anything reaches disk. Returning empty for anything unreadable degrades to no
// map entry, hence a later fail-open, hence the behaviour that existed before
// this function: a wrong guess about a field name can cost enforcement and can
// never cause a block. Same containment as skillArg.
func dispatchArgs(tool string, args map[string]any) (name, agentType string) {
	if !isDispatchTool(tool) {
		return "", ""
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := args[k].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	// ONE SPELLING FOR THE TYPE, because one is what was measured. The extra
	// spellings this carried at first were belt-and-braces with nothing behind
	// them, and the independent review on #1471 was right to ask: this spec's
	// entire argument is that it reads a field it measured rather than one it
	// guessed, and a guessed key in the same function undercuts it. If a rename
	// ever happens the correct response is to measure the new name, not to have
	// pre-guessed it — a wrong guess here would map a name onto the wrong type,
	// which resolves the WRONG persona, and that is worse than resolving none.
	return str("name"), str("subagent_type")
}

func skillArg(tool string, args map[string]any) string {
	if !isSkillTool(tool) {
		return ""
	}
	for _, k := range []string{"skill", "name", "command"} {
		if v, ok := args[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
