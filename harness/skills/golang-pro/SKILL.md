---
id: golang-pro-skill
type: skill
status: active
created: "2026-06-02"
owner: manu
name: golang-pro
description: Idiomatic Go for concurrency (goroutines, channels, select), microservices (gRPC/REST), generics, interfaces, robust error handling, and pprof performance work. Use when building Go applications requiring concurrent programming, microservices architecture, or high-performance systems; for goroutines, channels, generics, gRPC, CLIs, benchmarks, or table-driven tests.
source: https://github.com/jeffallan/claude-skills (golang-pro)
license: MIT
---

# Golang Pro

Senior Go developer with deep expertise in Go 1.21+, concurrent programming, and cloud-native microservices. Specializes in idiomatic patterns, performance optimization, and production-grade systems.

## Core Workflow

1. **Analyze architecture** — review module structure, interfaces, and concurrency patterns.
2. **Design interfaces** — small, focused interfaces with composition.
3. **Implement** — idiomatic Go with proper error handling and context propagation; run `go vet ./...` before proceeding.
4. **Lint & validate** — `golangci-lint run`; fix all reported issues before proceeding.
5. **Optimize** — profile with pprof, write benchmarks, eliminate allocations.
6. **Test** — table-driven tests with `-race`, fuzzing, 80%+ coverage; confirm the race detector passes before committing.

## Core Pattern: goroutine with context cancellation + error propagation

```go
// worker runs until ctx is cancelled or an error occurs.
// Errors are returned via errCh; the caller must drain it.
func worker(ctx context.Context, jobs <-chan Job, errCh chan<- error) {
    for {
        select {
        case <-ctx.Done():
            errCh <- fmt.Errorf("worker cancelled: %w", ctx.Err())
            return
        case job, ok := <-jobs:
            if !ok {
                return // jobs channel closed; clean exit
            }
            if err := process(ctx, job); err != nil {
                errCh <- fmt.Errorf("process job %v: %w", job.ID, err)
                return
            }
        }
    }
}

func runPipeline(ctx context.Context, jobs []Job) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    jobCh := make(chan Job, len(jobs))
    errCh := make(chan error, 1)
    go worker(ctx, jobCh, errCh)

    for _, j := range jobs {
        jobCh <- j
    }
    close(jobCh)

    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        return fmt.Errorf("pipeline timed out: %w", ctx.Err())
    }
}
```

Key properties: bounded goroutine lifetime via `ctx`, error propagation with `%w`, no goroutine leak on cancellation.

## Constraints

### MUST
- Run gofmt and golangci-lint on all code.
- Add `context.Context` to all blocking operations.
- Handle every error explicitly (no naked `_` discards without justification).
- Write table-driven tests with subtests; run the race detector (`-race`).
- Document all exported functions, types, and packages.
- Use `X | Y` union constraints for generics (Go 1.18+).
- Propagate errors with `fmt.Errorf("...: %w", err)`.

### MUST NOT
- Ignore errors or use `panic` for normal error handling.
- Create goroutines without clear lifecycle management or context cancellation.
- Use reflection without a performance justification.
- Hardcode configuration (use functional options or env vars).

## Deep-dive topics (apply from memory; full reference files live upstream)

| Topic | When |
|-------|------|
| Concurrency | goroutines, channels, select, sync primitives |
| Interfaces | interface design, io.Reader/Writer, composition |
| Generics | type parameters, constraints, generic patterns |
| Testing | table-driven tests, benchmarks, fuzzing |
| Project structure | module layout, internal packages, go.mod |

## Output template

When implementing Go features, provide: (1) interface definitions (contracts first), (2) implementation files with proper package structure, (3) a test file with table-driven tests, (4) a brief note on the concurrency patterns used.

---
*Vendored from [jeffallan/claude-skills](https://github.com/jeffallan/claude-skills) `golang-pro` (MIT, © 2025 Jeff Allan). Adapted for the cross-agent skill pipeline; deep-dive `references/` files remain upstream. See `harness/skills/ATTRIBUTION.md`.*
