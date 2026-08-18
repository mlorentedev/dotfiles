---
generated: true
generated_from: 00_meta/skills/debug-hardware/SKILL.md
generated_sha: 5e6679e8127538cb
id: debug-hardware-skill
type: skill
status: active
created: '2026-05-30'
owner: manu
name: debug-hardware
description: Use when troubleshooting hardware or firmware issues -- device communication,
  register configuration, signal processing, camera/sensor behavior, or embedded systems.
keywords: [debug hardware, firmware, sensor, camera register, embedded, signal processing]
paths: ['**/firmware/**', '**/hardware/**', '**/embedded/**']
---
# Hardware Debugging

The evidence-first debugging **method** lives in one skill: `systematic-debugging` (the Iron Rule — no guessing, a single hypothesis formed from evidence, the minimal fix, verify after every change). This skill is the **hardware/firmware specialization** of it: same method, plus the domain-specific evidence sources and pitfalls below. Reach for `systematic-debugging` for the process; reach for this for what "evidence" means in hardware.

## Hardware-specific evidence sources

Before forming a hypothesis (the "gather evidence" step of `systematic-debugging`), read the hardware truth — never assume it:

- **Reference implementation** — vendor examples, GUI code, known-good configurations. Always compare against known-working code.
- **Datasheets / register maps / timing diagrams / protocol specs** — register *reset* values, setup/hold times, clock domains, byte order.
- **Concrete observations** — register values, signal traces, error codes. Observed vs expected, with numbers.

## Common Pitfalls (the hardware delta)

| Pitfall | Correct Approach |
|---------|-----------------|
| Assuming register defaults | Read datasheet for actual reset values |
| Changing multiple registers at once | One register change per test |
| Ignoring timing requirements | Check setup/hold times, clock domains |
| Guessing endianness | Verify byte order from documentation |
| Skipping reference implementation | Always compare against known-working code |

> Method owner: `systematic-debugging`. This skill adds only the hardware evidence sources + pitfalls; it does not restate the Iron Rule or the generic process (that would be triplication — see HARNESS-021 D3).
