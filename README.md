# MailJail

MailJail is a modular mail server stack for FreeBSD that uses native FreeBSD jails and ZFS instead of Docker containers. The project uses Bastille as its jail orchestration layer and Angie as its web edge/proxy layer. The goal is to provide a reproducible, automated, and maintainable way to deploy production-ready email infrastructure.

## Goals

- Fully automated deployment from a single configuration file
- Isolation of each major component in a separate jail
- Deterministic provisioning behavior
- Independent upgrade, restart, and diagnostics workflows per module
- Clear separation between host preparation, jail orchestration, and service configuration

## Core Modules

- `postfix`: SMTP ingress and egress
- `dovecot`: IMAP/POP3 and mail storage access
- `rspamd`: spam filtering and policy decisions
- `redis`: cache/backend for Rspamd and other stateful dependencies
- `db`: configuration and application database
- `web`: Angie-based web edge, admin and user web interface
- `acme`: TLS certificates and rotation
- `dns`: DNS prerequisite validation, without necessarily hosting DNS

## Architectural Principles

### 1. Host and workload separation

The host machine is responsible for:

- ZFS datasets
- jail networking
- packet filter rules
- secrets placement
- lifecycle orchestration
- Bastille bootstrap and jail lifecycle integration

Each jail is responsible only for its specific service.

### 2. Declarative provisioning

The desired state is described in a configuration file. The CLI tool:

1. validates the configuration
2. prepares host prerequisites
3. prepares Bastille and datasets
4. creates and connects jails
5. installs packages
6. renders configuration files
7. starts services in the correct order
8. runs post-deploy checks

### 3. Modularity

Each component must be able to:

- install independently
- upgrade independently
- restart independently
- be diagnosed independently

### 4. Reproducibility first

Manual steps outside the CLI tool should not be required for a standard deployment.

## Planned CLI Contract

Example commands:

```text
mailjail init
mailjail validate -c mailjail.yml
mailjail plan -c mailjail.yml
mailjail apply -c mailjail.yml
mailjail status
mailjail upgrade <module>
mailjail restart <module>
mailjail destroy --keep-data
```

The first implementation target is a single `mailjail` CLI binary that drives Bastille, ZFS, PF, `pkg`, and service management from a declarative config file. See [docs/cli.md](/Users/vnedyalk0v/Projects/Personal/MailJail/docs/cli.md) for the command model and [docs/adr/0001-use-bastille.md](/Users/vnedyalk0v/Projects/Personal/MailJail/docs/adr/0001-use-bastille.md) for the orchestration decision.

## Example Configuration Model

```yaml
apiVersion: mailjail.io/v1alpha1
kind: MailStack

metadata:
  name: mx1

host:
  hostname: mx1.example.com
  externalInterface: vtnet0
  zfsPool: zroot
  jailDatasetRoot: zroot/mailjail
  bastille:
    dataset: zroot/bastille
    release: 15.0-RELEASE

network:
  domain: example.com
  bridge: bridge0
  jailsSubnet: 10.77.0.0/24
  gateway4: 10.77.0.1

tls:
  mode: acme
  email: admin@example.com

profiles:
  - core

modules:
  postfix:
    enabled: true
    ip4: 10.77.0.10
  dovecot:
    enabled: true
    ip4: 10.77.0.11
  rspamd:
    enabled: true
    ip4: 10.77.0.12
  redis:
    enabled: true
    ip4: 10.77.0.13
  db:
    enabled: false
    ip4: 10.77.0.14
  web:
    enabled: true
    ip4: 10.77.0.15
    edge: angie
```

## Implementation Phases

### Phase 1: Foundation

- define the config schema
- dry-run planning engine
- host preflight checks
- Bastille bootstrap integration
- ZFS dataset provisioning
- base jail lifecycle

### Phase 2: Core Mail Path

- Postfix jail
- Dovecot jail
- TLS wiring
- network connectivity between jails

### Phase 3: Filtering and State

- Rspamd
- Redis
- DKIM/DMARC/SPF integration checks

### Phase 4: Control plane

- database module
- Angie web edge and web UI
- account/domain management

### Phase 5: Operations

- health checks
- backup/restore
- rolling upgrades
- disaster recovery docs

## Security Expectations

- minimal privileges on the host and inside jails
- no unnecessary embedding of secrets in generated configurations
- clear service-to-service allow lists
- separate datasets for persistent data
- support for immutable templates where practical
- audit-friendly logs and predictable state transitions

## What Is Missing Today

The repository does not yet contain an implementation. The first practical goal is to lock down:

- config schema
- CLI execution model
- module dependency graph
- filesystem/network layout
- Bastille integration boundary

After that, the implementation language can be chosen and the first vertical slice can be built: `validate -> plan -> create base jail`.
