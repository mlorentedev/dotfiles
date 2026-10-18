# Ledger — winzero-20261018T0829Z

Armed 2026-10-18T08:29:15Z, expires 16:29:15Z. Wake-lock: `caffeinate -dims -t 28800` (Mac on power, lid open, per the owner).

Cold start: `dotf doctor --quick` 21 passed, 0 failed; `gh auth status` ok; `dotf pr triage-queue` exit 0.

| Time (UTC) | Item | Event |
|---|---|---|
| 08:29 | — | armed; VM run b30bt02mo (main + #2770 + #2771 + #2775, GitBinDirs swapped) in flight |
| 08:29–10:31 | — | host session paused (usage limit); the VM run kept going; wake-lock held (`caffeinate -dims` pid 83036 alive at 10:33) |
| 06:26–06:39 guest | AC9 | `dotf vm windows --ref tmp/vm-check-2754-2734 --dotf … --keep`: first converge `9 changed, 1 ok, 1 opaque, 7 skipped, 0 failed`; second `0 changed, 11 ok, 1 opaque, 6 skipped, 0 failed`; "converged, and the second run changed nothing"; exit 0. Log: scratchpad final4.log |
| 10:35 | #2734 | repro in kept guest: git uninstalled, repo moved aside → `installed git, then cloned …`, `1 changed … 0 failed`. Same-shell second run skipped the fast-forward (stale PATH) → fixed in #2770 |
| 10:36 | #2771 | found merged by mlorentedev at 08:33:33Z (`273c0701`, head `0ff8037e`), not by this run. CodeRabbit finding 3 carried into #2770 |
| 10:45 | #2770 | merged main (`24c418e8`): GitBinDirs → registered PATH; reach before fast-forward; mutation killed; go test/vet/lint ok; bats 3 files exit 0 |
| 10:40 | #2770 | live in kept guest: build of `24c418e8`, shell with no Git dir on PATH → `[ OK ] checkout … already current (origin/main)` (was `[SKIP] git is not on PATH`) |
| 10:42 | #2775 | merged main (`9bd38b27`), go test/vet/lint ok, pushed, marked ready (<50 prod LOC, no spec gate) |
| 10:45 | #2772 | merged main, ticked PR 3c + AC9, evidence in verification.md (`8760e4fa`), pushed; stays draft until #2770/#2775 land |
| 10:50 | #2764 | claimed; TDD red with the guest's error → `/qh`; PR #2780 ready. Guest probes after this were DENIED by the auto-mode classifier ([Security Weaken]); no further guest commands |
| 10:52 | #2780 | knowledge-gate red: the Runbook line named a path the PR does not change; body reworded via REST |
| 10:55 | #2758 | claimed; TDD red (raw read error) → probe the service when sshd_config is missing; mutation killed; PR #2781 |
| 11:05 | #2765 | claimed; TDD red ×2 → Windows firewall refusal surfaces at plan; failed apply returns 0 changes + `planned as root: …; <err>`; PR #2782 |
| 11:20 | #2763 | advisor: ship part 1 (Include form), park part 2 (sshd -T proof). Part 1 TDD red, 2 mutations killed, PR #2783 (Refs, skip-sdd + rationale). Part 2 parked: ERROR/WHY/FIX comment, board Blocked (verified) |
| 10:54 | #2770 | self-triage posted (issuecomment-6248976662: merge content disclosed, pr-agent did not re-review); triage-queue --pr exit 0; no path overlap; `dotf pr land` → MERGED `5e3f383d`. #2754 closed/Done |
| 11:00 | #2775 | pr-agent 2 THEORETICAL findings both applied in `81096247`: cache only a non-empty pin (an early empty answer no longer freezes), install-time test `TestInstallUvTool_AsksThePinUntilItAnswers` (2 mutations killed); misplaced installUvTool doc comment fixed |
| 11:02 | housekeeping | worktrees fix-windows-git-bootstrap and fix-windows-uv-wait removed (`dotf worktree done`) |
| 11:15 | #2780 | attestation re-evaluated to attested after pr-agent's review; self-triage (issuecomment-6249116878); landed. #2764 closed |
| 11:25 | #2781 | pr-agent 2 findings applied (`115ef3e3`): probe error wrapped with the missing path; installed-service-but-no-config no longer asks to install. Body re-patched |
| 11:30 | #2782 | pr-agent no findings; self-triage; landed `9a3fe8d8`. #2765 closed |
| 11:33 | #2783 | pr-agent 2 harness findings declined (body has both sections, label present), part-2 non-compliance by design (parked); self-triage; land waited on another session's `pr land 2779` lock, then MERGED. #2763 stays open (part 2 Blocked) |
| 11:36 | #2781 | merged main (conflict in target_test.go: both sides added tests; kept both) `034c0daa`, pushed |
| 11:45 | #2775 | self-triage; landed `81096247`. #2769 closed |
| 11:47 | #2772 | merged main `9aebb0ae`; body updated with the AC9 run; marked ready |
