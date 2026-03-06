# AGENTS.md

## Purpose

This repository builds `mailjail`, a security-focused FreeBSD mail-stack CLI.

This file defines the engineering rules for coding in Go in this repository. Follow these rules when designing, implementing, reviewing, or refactoring the Go codebase.

The goals are:

- secure by default behavior
- deterministic and testable execution
- low CPU and memory overhead
- maintainable code with minimal duplication
- clear operator-facing behavior

## Working Style

### Completion contract

Treat a task as incomplete until all requested deliverables are done or explicitly marked blocked.

Keep an internal checklist for:

- code changes
- tests
- documentation updates
- verification steps

If something is blocked, state exactly what is missing.

### Follow-through policy

If the user intent is clear and the next step is reversible and low-risk, proceed without asking.

Ask before taking actions that are:

- destructive
- irreversible
- production-affecting
- dependent on missing sensitive information

### Missing context policy

Do not guess when required context is missing.

Prefer to:

1. inspect the repository
2. inspect existing docs and configs
3. use tools to verify assumptions
4. ask a short clarifying question only when the missing information cannot be discovered safely

### Verification loop

Before finalizing any task:

- verify that every requirement was addressed
- verify that code changes are grounded in the current repository state
- verify that tests or checks relevant to the change were run
- verify that output format and docs match the requested style

## Go Version Policy

Until explicitly changed, target:

- latest stable Go patch release available at implementation time
- current project baseline: `Go 1.26.1`

Recommended `go.mod` policy for this repository:

```go
go 1.25.0
toolchain go1.26.1
```

Rationale:

- develop and build with the latest secure toolchain
- keep the language floor conservative unless a newer feature is required
- allow reproducible toolchain selection

## Project Layout Rules

Prefer a single Go module for this repository.

Use this structure unless there is a strong reason not to:

```text
cmd/mailjail/
internal/
  apply/
  config/
  health/
  host/
    bastille/
    pf/
    preflight/
    service/
    zfs/
  modules/
  plan/
  render/
  schema/
  state/
```

Rules:

- put the CLI entrypoint in `cmd/mailjail`
- keep non-public code in `internal/`
- do not create multiple Go modules without a clear need
- do not add `pkg/` by default
- add packages only when a package boundary is clearer than a file boundary

## CLI Architecture Rules

The CLI is the control plane. It must remain predictable and easy to reason about.

Design rules:

- declarative input, imperative execution
- plan before mutate
- keep side effects at the edges
- make actions typed and explicit
- preserve idempotent behavior where practical
- prefer composition over inheritance-like abstractions

The command flow should remain:

1. load config
2. validate config and host prerequisites
3. build a deterministic plan
4. apply the plan in ordered stages
5. verify health and report results

Do not let module-specific logic bypass the planner.

## Dependency Rules

Prefer the standard library first.

Allowed by default:

- `context`
- `errors`
- `flag` or a thin command layer
- `log/slog`
- `os/exec`
- `path/filepath`
- `testing`

Third-party dependencies must clear a high bar:

- they remove meaningful complexity
- they are maintained and widely trusted
- they do not pull in a large dependency tree unnecessarily
- they are justified in the change

Avoid:

- framework-heavy abstractions
- hidden global state
- reflection-heavy configuration magic
- large utility libraries for small problems

## Security Rules

Security takes priority over convenience.

### Command execution

Use `exec.CommandContext`, never `sh -c` unless there is no viable alternative.

Rules:

- pass arguments explicitly
- set timeouts with `context.Context`
- capture stdout and stderr separately when useful
- surface exit status and stderr clearly
- sanitize and validate user-derived inputs before passing them to system commands

Never rely on implicit current-directory execution.

### Filesystem and secrets

- use least-privilege file modes
- never log secrets
- never write secrets into debug logs
- avoid storing secrets in long-lived in-memory structures unless necessary
- redact sensitive config values in user-visible output

### Dependency and toolchain hygiene

- keep Go patched to the latest secure point release
- review dependency upgrades, do not blindly update everything
- run `govulncheck ./...` on meaningful dependency or code changes
- run `go vet ./...` regularly

### Concurrency safety

- do not introduce goroutines unless there is a clear measured need
- prefer simple sequential execution for stateful orchestration
- if concurrency is needed, define ownership and cancellation explicitly
- run `go test -race ./...` for changes touching concurrent code

## Performance and Resource Rules

This CLI should be fast and lightweight.

Rules:

- avoid unnecessary goroutines
- avoid caching without evidence
- avoid background daemons for v1
- keep data structures simple and bounded
- stream where possible instead of loading large blobs eagerly
- prefer zero-copy or low-allocation paths only when they improve a measured hot path

Logging rules:

- use `log/slog`
- keep default logs concise and structured
- use debug logs for verbose operational detail
- avoid chatty logs in hot paths
- prefer `Logger.With` for repeated attributes
- prefer `LogAttrs` on frequently executed paths when allocations matter

## DRY and Code Quality Rules

Apply DRY to behavior, not to superficial syntax.

Rules:

- duplicate a small amount of code if it keeps control flow obvious
- extract shared logic only after the duplication is proven real
- do not create generic helper layers too early
- prefer clear package-local functions over premature reusable frameworks
- keep functions small, but not artificially tiny

Write code that is:

- explicit
- readable
- easy to test
- easy to delete or replace

Prefer:

- concrete types over `interface{}`-style designs
- typed errors where branching matters
- sentinel errors only when they improve calling code
- table-driven tests where they improve coverage clarity

## Error Handling Rules

- return errors with context
- do not swallow errors
- do not panic for expected operational failures
- panic only for truly impossible states or programmer bugs
- user-facing command errors must be actionable

Error messages should answer:

- what failed
- where it failed
- what command or resource was involved
- whether the error is retryable

## Configuration Rules

- configuration is the source of truth
- local state supports execution history, not desired state
- validate as early as possible
- reject ambiguous configuration
- apply defaults explicitly in code, not implicitly by scattered call sites

Config loading rules:

- parse once
- validate once centrally
- pass typed config objects downward
- do not re-read config files deep inside modules

## Module Design Rules

Each module should have explicit ownership and dependencies.

A module should define:

- its name
- its dependencies
- config validation
- planning behavior
- health checks

Do not let modules:

- directly mutate unrelated modules
- execute host commands without going through host adapters
- hide required dependencies

## Testing Rules

Every non-trivial change should add or update tests.

Default test stack:

- `go test ./...`
- `go test -race ./...` for concurrency-sensitive changes
- targeted fuzz tests for parser, renderer, and config validation paths
- integration tests for planner and host command adapters where feasible

Testing rules:

- prefer table-driven tests for validators and planners
- use golden files for rendered config output when stable
- isolate system-command execution behind interfaces so it can be faked in tests
- do not make unit tests depend on a real FreeBSD host

## Logging and Observability Rules

Use structured logging from day one.

Rules:

- all major actions should log start, target, and result
- include enough fields for operators to trace a failed apply
- keep secrets and raw credentials out of logs
- prefer stable field names
- separate human-readable command output from machine-readable logs where useful

## Refactoring Rules

Before refactoring:

- identify the current responsibility boundary
- preserve behavior unless the change explicitly intends to alter it
- add tests first when the current behavior is under-specified

Do not perform broad refactors together with unrelated feature changes unless necessary.

## Documentation Rules

When behavior changes, update the relevant docs in the same change when practical:

- `README.md`
- `docs/architecture.md`
- `docs/cli.md`
- example configs
- ADRs for durable architectural decisions

## Code Review Checklist

Before considering a Go change done, check:

- is the command flow deterministic?
- are side effects isolated?
- are errors actionable?
- is command execution shell-safe?
- is the package structure still simple?
- did we avoid needless dependencies?
- did we preserve low resource usage?
- were tests or checks run?
- were docs updated if behavior changed?

## Default Commands

Use these commands as the default verification baseline when the repository contains Go code:

```bash
gofmt -w .
go test ./...
go vet ./...
govulncheck ./...
```

Additionally run when relevant:

```bash
go test -race ./...
go test -fuzz=Fuzz -run=^$
```

## Source Basis

This file is based on:

- OpenAI prompt guidance for explicit output contracts, completion rules, follow-through, tool persistence, and verification loops
- official Go guidance on release policy, security best practices, module organization, structured logging, toolchain selection, and safe command execution

References:

- [OpenAI Prompt Guidance](https://developers.openai.com/api/docs/guides/prompt-guidance)
- [Go Release History](https://go.dev/doc/devel/release)
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [Organizing a Go module](https://go.dev/doc/modules/layout)
- [Go Toolchains](https://go.dev/doc/toolchain)
- [Structured Logging with slog](https://go.dev/blog/slog)
- [os/exec](https://pkg.go.dev/os/exec)
