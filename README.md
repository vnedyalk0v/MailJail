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

## CLI Contract

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

The current implementation is a single `mailjail` CLI binary written in Go. It drives Bastille, ZFS, PF, `pkg`, and service management from a declarative config file. See [docs/cli.md](/Users/vnedyalk0v/Projects/Personal/MailJail/docs/cli.md) for the command model and [docs/adr/0001-use-bastille.md](/Users/vnedyalk0v/Projects/Personal/MailJail/docs/adr/0001-use-bastille.md) for the orchestration decision.

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

## Current Implementation Status

The repository now contains the first working Go CLI slice:

- `mailjail init`
- `mailjail validate`
- `mailjail plan`
- `mailjail apply`
- `mailjail status`

The current implementation covers:

- YAML config loading
- schema validation
- host preflight checks
- deterministic plan generation
- ZFS dataset creation
- PF anchor rendering and loading
- Bastille release bootstrap
- base jail creation
- Redis module jail creation
- package installation inside the Redis jail
- Redis service enable and start orchestration
- Rspamd module jail creation
- package installation inside the Rspamd jail
- Rspamd service enable and start orchestration
- Dovecot module jail creation
- package installation inside the Dovecot jail
- Dovecot service enable and start orchestration
- Postfix module jail creation
- package installation inside the Postfix jail
- Postfix service enable and start orchestration
- local apply state recording

The next practical goals are:

- config rendering
- health checks
- richer mail-service configuration
- restart and upgrade workflows

## Development

Build and verify the current CLI with:

```text
go test ./...
go vet ./...
go run ./cmd/mailjail plan -c examples/mailjail.example.yml
```

For repeatable local workflows, use [Makefile](/Users/vnedyalk0v/Projects/Personal/MailJail/Makefile):

```text
make tools
make install-hooks
make check
make ci
make build-freebsd-amd64
make build-freebsd-arm64
```

`make check` is the closest local equivalent of the GitHub CI verification flow. It installs repo-local tool binaries under `bin/tools` and runs formatting, module drift checks, tests, race tests, vet, lint, and vulnerability scanning.

To enforce the same checks before every push, install the repository hook path once:

```text
make install-hooks
```

That enables [.githooks/pre-push](/Users/vnedyalk0v/Projects/Personal/MailJail/.githooks/pre-push), which runs `make check` automatically before `git push`.

GitHub Actions now builds release artifacts for:

- `freebsd/amd64`
- `freebsd/arm64`

## Install on FreeBSD

The release pipeline publishes signed release artifacts for:

- `freebsd/amd64`
- `freebsd/arm64`

To install the latest release directly on a FreeBSD host:

```text
fetch -o - https://raw.githubusercontent.com/vnedyalk0v/MailJail/main/scripts/install.sh | sh
```

If `curl` is preferred:

```text
curl -fsSL https://raw.githubusercontent.com/vnedyalk0v/MailJail/main/scripts/install.sh | sh
```

The installer:

- detects `amd64` or `arm64`
- downloads the matching GitHub Release artifact
- verifies its SHA-256 checksum
- installs `mailjail` to `/usr/local/bin`

Optional environment variables:

```text
MAILJAIL_VERSION=v0.1.0
MAILJAIL_INSTALL_DIR=/usr/local/bin
MAILJAIL_REPO=vnedyalk0v/MailJail
```

Example:

```text
env MAILJAIL_VERSION=v0.1.0 fetch -o - https://raw.githubusercontent.com/vnedyalk0v/MailJail/main/scripts/install.sh | sh
```
