// Deployed by `dotf deploy` (ai/deploy.json, entry pi-gate). Edit the
// source in the dotfiles repository, not this copy.
//
// Tells each pi session where it stands: runs `dotf mem session-start` and adds
// its output, which carries the lane line (HARNESS-195), to the next turn's
// context. A missing or failing dotf is logged and skipped; it never breaks
// session start.
import { execFile } from "node:child_process";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

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

export default function (pi: ExtensionAPI) {
  pi.on("session_start", async (event, ctx) => {
    // A reload keeps the session, which was told when it started.
    if (event.reason === "reload") return;
    const brief = await sessionBrief(ctx.cwd);
    if (!brief) return;
    await pi.sendMessage(
      { customType: "dotf-session-start", content: brief, display: true },
      { deliverAs: "nextTurn" },
    );
  });
}
