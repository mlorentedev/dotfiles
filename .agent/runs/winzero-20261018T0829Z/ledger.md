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
