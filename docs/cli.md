# CLI Design

## Goal

The `mailjail` CLI is the control plane for the entire stack. It reads declarative configuration, computes an execution plan, and applies that plan through Bastille, ZFS, PF, `pkg`, and service management.

The first implementation should be a single local CLI binary, not a daemon and not a distributed controller.

## Implementation Language

The recommended implementation language is Go.

Reasons:

- simple static binary distribution
- straightforward YAML parsing and schema validation support
- strong support for structured CLI applications
- good fit for executing and supervising host commands
- low operational overhead on FreeBSD hosts

The CLI should use a standard command framework such as Cobra and structured logging in JSON and human-readable formats.

## Core Design Principles

- declarative input, imperative execution
- deterministic planning before mutation
- idempotent `apply` where practical
- explicit host-side command execution boundaries
- no hidden state outside the configured state directory
- strong separation between planning logic and command runners

## Command Surface

### `mailjail init`

Creates a starter configuration file and optional host-side directory layout.

### `mailjail validate -c mailjail.yml`

Validates:

- schema and required fields
- module dependency graph
- network addressing
- dataset naming
- Bastille prerequisites
- host tools and privileges

### `mailjail plan -c mailjail.yml`

Builds a deterministic execution plan and prints:

- host actions
- Bastille jail actions
- dataset actions
- firewall actions
- package actions
- rendered file targets
- service restart actions

### `mailjail apply -c mailjail.yml`

Executes the plan in ordered stages and records results in a local state directory.

### `mailjail status`

Shows:

- configured modules
- jail existence and running state
- service health
- recent apply history

### `mailjail restart <module>`

Restarts one module without rebuilding unrelated components.

### `mailjail upgrade <module>`

Upgrades one module using the same declarative config and compatibility checks.

### `mailjail destroy`

Destroys selected jails and runtime wiring. Persistent data is removed only when explicitly requested.

## Internal Package Layout

Suggested initial package layout:

```text
cmd/mailjail
internal/config
internal/schema
internal/plan
internal/apply
internal/state
internal/modules
internal/modules/postfix
internal/modules/dovecot
internal/modules/rspamd
internal/modules/redis
internal/modules/db
internal/modules/web
internal/host/bastille
internal/host/zfs
internal/host/pf
internal/host/pkg
internal/host/service
internal/host/preflight
internal/render
internal/health
```

## Execution Model

### Config loading

The CLI loads one YAML file and resolves defaults into an in-memory model.

### Validation phase

The validator produces structured errors with field paths and actionable messages.

### Planning phase

The planner converts desired state into a linear execution plan composed of typed actions.

Example action categories:

- `EnsureBastilleInstalled`
- `EnsureBastilleRelease`
- `EnsureDataset`
- `EnsureBridge`
- `EnsurePfAnchor`
- `EnsureJail`
- `InstallPackages`
- `RenderFile`
- `EnableService`
- `RestartService`
- `RunHealthCheck`

### Apply phase

The applier executes actions in order, records results, and stops on hard failures. Each action should emit:

- action type
- target
- command
- duration
- result

## State Model

MailJail should maintain a local state directory, for example:

```text
/var/db/mailjail/
  state.json
  plans/
  renders/
  backups/
  logs/
```

This state is not the source of truth for desired configuration. The YAML config remains the source of truth. The local state exists to support:

- change detection
- plan/apply history
- file render backups
- health and diagnostic output

## Bastille Integration Boundary

MailJail should treat Bastille as an infrastructure adapter, not as the source of stack semantics.

Recommended responsibilities:

- MailJail decides what jails should exist
- Bastille creates and destroys jails
- MailJail decides what packages and configs belong in each jail
- Bastille provides stable host-side primitives for jail lifecycle

## Module Model

Each module should implement a common interface:

```text
Name()
Dependencies()
Validate(config)
Plan(context)
HealthChecks()
```

This keeps module-specific logic out of the global planner and makes it possible to add modules later without rewriting the core engine.

## First Deliverable

The first working milestone should include only:

1. `init`
2. `validate`
3. `plan`
4. `apply` for Bastille bootstrap, one dataset root, and one base jail
5. `status`

That is enough to prove the engine before implementing Postfix, Dovecot, or the rest of the mail stack.
