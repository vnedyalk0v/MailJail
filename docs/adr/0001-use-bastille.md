# ADR 0001: Use Bastille as the Jail Orchestration Layer

## Status

Accepted

## Date

2026-03-06

## Context

MailJail needs a stable, FreeBSD-native way to create, bootstrap, start, stop, and destroy jails without building low-level host orchestration from scratch in the first implementation.

The project goal is not to compete with Bastille. The project goal is to provide a mail-stack-specific control plane with deterministic planning, security-oriented defaults, and module-aware orchestration.

## Decision

MailJail will use Bastille as its jail orchestration layer.

MailJail will not delegate the full stack lifecycle to Bastille templates alone. Instead, it will use Bastille for base jail primitives and keep stack-specific planning and module orchestration inside the `mailjail` CLI.

## Rationale

- Bastille is a mature FreeBSD jail management tool with established operational workflows
- it reduces time-to-first-working-release
- it avoids reimplementing low-level jail lifecycle behavior immediately
- it still allows MailJail to own planning, validation, and module semantics
- it fits the project goal of a jail-native deployment model

## Consequences

Positive:

- faster delivery of a working v1
- lower host orchestration complexity
- easier troubleshooting for FreeBSD operators already familiar with Bastille

Negative:

- MailJail now has a runtime dependency on Bastille
- some operations will need careful mapping between Bastille concepts and MailJail module concepts
- the project must define clear ownership boundaries to avoid split-brain behavior

## Ownership Boundary

Bastille owns:

- release bootstrap
- jail creation and destruction primitives
- jail start and stop primitives
- template hooks where useful

MailJail owns:

- config schema
- dependency graph
- ZFS layout
- firewall policy
- package selection
- configuration rendering
- health checks
- safe apply sequencing
