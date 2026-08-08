---
id: async-python-patterns-skill
type: skill
status: active
created: "2026-06-02"
owner: manu
name: async-python-patterns
description: Master Python asyncio, concurrent programming, and async/await patterns for high-performance applications. Use when building async APIs, concurrent systems, or I/O-bound applications requiring non-blocking operations.
source: https://github.com/wshobson/agents (async-python-patterns)
license: MIT
---

# Async Python Patterns

Implementing asynchronous Python with asyncio and async/await for high-performance, non-blocking systems.

## When to use

Async web APIs (FastAPI, aiohttp, Sanic); concurrent I/O (DB, file, network); concurrent web scraping; real-time apps (WebSocket, chat); processing many independent tasks; async microservice communication; optimizing I/O-bound workloads; async background tasks and queues.

## Sync vs async decision guide

| Use case | Approach |
|----------|----------|
| Many concurrent network/DB calls | `asyncio` |
| CPU-bound computation | `multiprocessing` or a thread pool |
| Mixed I/O + CPU | offload CPU work with `asyncio.to_thread()` |
| Simple scripts, few connections | sync (simpler to debug) |
| Web APIs with high concurrency | async frameworks (FastAPI, aiohttp) |

**Key rule:** stay fully sync or fully async within a call path. Mixing creates hidden blocking.

> **Modern structured concurrency (3.11+):** prefer `asyncio.TaskGroup` over bare `gather` for supervised tasks, and `asyncio.timeout()` over `wait_for` where available — they cancel siblings on failure and compose cleanly.

## Core concepts

- **Event loop** — single-threaded cooperative scheduler; runs coroutines, handles I/O without blocking.
- **Coroutines** — `async def` functions that can pause/resume at `await`.
- **Tasks** — scheduled coroutines running concurrently (`asyncio.create_task`).
- **Futures** — low-level eventual results.
- **Async context managers / iterators** — `async with` / `async for` for resource cleanup and async streams.

## Patterns

```python
import asyncio

# 1. Basic async/await
async def fetch_data(url: str) -> dict:
    await asyncio.sleep(1)        # simulate I/O
    return {"url": url, "data": "result"}

# 2. Concurrent execution
async def fetch_all(ids: list[int]) -> list[dict]:
    return await asyncio.gather(*(fetch_one(i) for i in ids))

# 3. Task management
async def main():
    t1 = asyncio.create_task(work("a", 2))
    t2 = asyncio.create_task(work("b", 1))
    return await asyncio.gather(t1, t2)

# 4. Error handling — collect, don't crash the batch
async def process(ids: list[int]):
    results = await asyncio.gather(*(safe(i) for i in ids), return_exceptions=True)
    ok = [r for r in results if not isinstance(r, Exception)]
    return ok

# 5. Timeout
async def with_timeout():
    try:
        return await asyncio.wait_for(slow(5), timeout=2.0)
    except asyncio.TimeoutError:
        return None
```

## Common pitfalls

- **Forgetting `await`** → you get a coroutine object, nothing runs.
- **Blocking the loop** with `time.sleep()` / sync I/O → use `await asyncio.sleep()` / async libs, or `asyncio.to_thread()`.
- **Not handling `CancelledError`** → catch, clean up, and re-raise to propagate cancellation.
- **Calling `await` from a sync function** → use `asyncio.run()` at the boundary.

## Testing async code

```python
import pytest, asyncio

@pytest.mark.asyncio
async def test_fetch():
    assert await fetch_data("https://api.example.com") is not None

@pytest.mark.asyncio
async def test_timeout():
    with pytest.raises(asyncio.TimeoutError):
        await asyncio.wait_for(slow(5), timeout=1.0)
```

## Deep-dive topics (apply from memory; full reference file upstream)

Async context managers, async iterators/generators, producer–consumer with `asyncio.Queue`, `asyncio.Semaphore` rate limiting, async locks/synchronization, real-world apps (aiohttp scraping, async DB, WebSocket servers), and performance (connection pools, batching, avoiding blocking ops).

---
*Vendored from [wshobson/agents](https://github.com/wshobson/agents) `async-python-patterns` (MIT, © 2024 Seth Hobson). Adapted for the cross-agent skill pipeline; the advanced `references/details.md` remains upstream. See `harness/skills/ATTRIBUTION.md`.*
