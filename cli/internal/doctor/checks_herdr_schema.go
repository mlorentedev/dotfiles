package doctor

import "encoding/json"

// checkHerdrWorktreePath reports a herdr that can no longer open a worktree at
// a given path (#2638, CLI-118 review round 1). `dotf worktree add --agent`
// runs `herdr worktree open --path <worktree>`, and a herdr upgrade that drops
// the parameter would break it with nothing upstream of the add to say so. The
// bundled API schema is herdr's own statement of what it accepts, so the check
// reads that rather than probing a live server.
func checkHerdrWorktreePath(sys *System, rep *Report) {
	rep.Section("herdr worktree API")
	if !sys.has("herdr") {
		rep.Skip("herdr not installed (`dotf worktree add --agent` needs it)")
		return
	}
	out, err := sys.CommandOutput("herdr", "api", "schema", "--json")
	if err != nil {
		rep.Warn("herdr api schema --json failed, so `dotf worktree add --agent` cannot verify it: " + err.Error())
		return
	}
	var schema struct {
		Schemas struct {
			Request struct {
				Defs map[string]struct {
					Properties map[string]json.RawMessage `json:"properties"`
				} `json:"$defs"`
			} `json:"request"`
		} `json:"schemas"`
	}
	if err := json.Unmarshal([]byte(out), &schema); err != nil {
		rep.Warn("herdr api schema --json is not JSON, so `dotf worktree add --agent` cannot verify it: " + err.Error())
		return
	}
	if _, ok := schema.Schemas.Request.Defs["WorktreeOpenParams"].Properties["path"]; !ok {
		rep.Fail("this herdr's API schema has no WorktreeOpenParams.path: `dotf worktree add --agent` passes `herdr worktree open --path` and will fail; pin HERDR_VERSION back in versions.conf or update cli/internal/herdr")
		return
	}
	rep.Pass("herdr accepts WorktreeOpenParams.path (`dotf worktree add --agent` opens worktrees by path)")
}
