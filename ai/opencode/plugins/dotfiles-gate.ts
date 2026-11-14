// Deployed by `dotf deploy` (ai/deploy.json, entry opencode-gate). Edit
// the source in the dotfiles repository, not this copy.
//
// Tells each opencode session where it stands: runs `dotf mem session-start`
// once per session and adds its output, which carries the lane line
// (HARNESS-195), to the system prompt. A missing or failing dotf is logged and
// skipped; it never breaks a turn.
import type { Plugin } from "@opencode-ai/plugin";
import { execFile } from "node:child_process";

function sessionBrief(cwd: string): Promise<string> {
  return new Promise((resolve) => {
    execFile(
      "dotf",
      ["mem", "session-start", "--format", "stdout"],
      { cwd, timeout: 30_000, windowsHide: true },
      (err, stdout) => {
        if (err) {
          console.error(`[dotfiles] session brief skipped: ${err.message}`);
          resolve("");
          return;
        }
        resolve(stdout.trim());
      },
    );
  });
}

const plugin: Plugin = async ({ directory }) => {
  // The hook runs before every model call. One brief per session keeps the
  // system prompt identical across a session's turns.
  const briefs = new Map<string, Promise<string>>();
  return {
    "experimental.chat.system.transform": async ({ sessionID }, output) => {
      const key = sessionID ?? "";
      if (!briefs.has(key)) briefs.set(key, sessionBrief(directory));
      const brief = await briefs.get(key);
      if (brief) output.system.push(brief);
    },
  };
};

export default { id: "dotfiles-gate", server: plugin };
