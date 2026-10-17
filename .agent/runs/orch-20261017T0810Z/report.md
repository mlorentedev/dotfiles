# Run orch-20261017T0810Z: ended goal-reached at 18:04Z, 0 merged by this run (merge: never)

Six runs worked in parallel without a path collision; 16 PRs landed between them. The refill never ran: every
run refused a peer-relayed widening of its envelope, and the reservations kept those tickets out of the
"unassigned" fallback pools. Lesson: a refill must be in the envelope at arming, or come from the owner directly.

## Waiting on the owner

- #2672 (Windows sshtarget) and #2683 (powercfg, stacked): held for acceptance on the work-lab Windows box. #2672 has an untriaged CodeRabbit round.
- #2681 (pf anchor, draft): waits for the owner's three root measurements.
- #2673 (deploy dst contract tokens): triaged, CI green, owner-lands (outside night's write_paths).
- #2699: OPS-053 report carrier, close without merging.
- #2680: release-please.
- Decisions: #2688 BUG-154 option A or B (blocks PLAT-003 AC8); #2399 A or B; #2388 and #2464 parked with options; #2552 waits on ADR-050.
- Reviews to run (not by the implementer): `dotf spec review OPS-053-self-deploy-trigger`; TOOL-030 after PR 0 and #2629.

## Refill tickets still reserved (assigned, commented)

tool030: #2470 #1186 #2303 #1314 · night: #2102 #2462 #1712 · pilot-r5: #1975 #2448 · plat003: #2136 #2320 · OPS-053: #1331 (needs a spec).
#2374 and #2201 closed by their runs; #2399 and #2464 parked.

## Landed tonight (by the runs)

#2665 #2669 #2676 #2675 #2679 #2682 #2689 #2692 #2693 #2694 #2695 #2697 #2701; owner landed #2664 #2667 #2671.

## Window

Covered: pilot-r5's `caffeinate -i -t 36000` held the Mac awake through 17:53Z.

## Next action

Decide whether the reserved refill tickets go into new runs armed by the owner, or release them (unassign).
