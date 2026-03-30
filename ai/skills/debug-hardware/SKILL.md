---
name: debug-hardware
description: Use when troubleshooting hardware or firmware issues -- device communication, register configuration, signal processing, camera/sensor behavior, or embedded systems.
---

# Hardware Debugging

Systematic approach for hardware/firmware issues. Evidence before hypotheses.

## The Iron Rule

```
NO GUESSING. NO CYCLING THROUGH HYPOTHESES WITHOUT EVIDENCE.
```

## Process

1. **Read reference code** — Find ALL related source files and working implementations (GUI code, vendor examples, known-good configurations)
2. **Read documentation** — Check firmware/hardware datasheets, register maps, timing diagrams, protocol specs
3. **Gather observations** — Ask user to describe observed vs expected behavior. Get concrete data: register values, signal traces, error codes
4. **Form single hypothesis** — Only AFTER steps 1-3. State clearly: "I think X because evidence Y shows Z"
5. **Propose minimal fix** — Smallest possible change to test the hypothesis. One variable at a time
6. **Verify** — Run tests after every change

## When Fix Fails

- Do NOT guess another cause
- Ask user for more observations
- Re-read documentation with new context
- Repeat from step 3

## Common Pitfalls

| Pitfall | Correct Approach |
|---------|-----------------|
| Assuming register defaults | Read datasheet for actual reset values |
| Changing multiple registers at once | One register change per test |
| Ignoring timing requirements | Check setup/hold times, clock domains |
| Guessing endianness | Verify byte order from documentation |
| Skipping reference implementation | Always compare against known-working code |
