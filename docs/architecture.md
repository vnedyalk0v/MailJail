# Architecture

## System Layout

MailJail splits the mail stack into independent jails with narrowly scoped responsibilities.

### Host Responsibilities

- ZFS pool and dataset management
- bridge and VNET networking management
- packet filter rules
- jail templates and image lifecycle
- secrets and certificates
- orchestration state
- Bastille bootstrap, templates, and jail lifecycle

### Jail Responsibilities

- installation of service-specific packages
- rendering of service configuration files
- startup and supervision of the specific service

## Orchestration Layer

MailJail uses Bastille as the host-side jail orchestration layer.

Bastille is responsible for:

- base release bootstrapping
- jail creation and destruction primitives
- template application hooks where useful
- consistent host-side operational workflows

MailJail remains responsible for:

- parsing the declarative stack configuration
- module dependency ordering
- ZFS layout planning
- PF rule generation
- package and service orchestration inside each jail
- idempotent plan/apply behavior

## Initial Module Dependency Graph

```text
db ----+
       |
redis -+--> rspamd

db ----------> web

postfix -----> rspamd
postfix -----> dovecot
dovecot -----> db

acme --------> postfix
acme --------> dovecot
acme --------> web
```

## Dataset Layout

Example ZFS layout:

```text
zroot/mailjail
zroot/mailjail/base
zroot/mailjail/jails/postfix
zroot/mailjail/jails/dovecot
zroot/mailjail/jails/rspamd
zroot/mailjail/jails/redis
zroot/mailjail/jails/db
zroot/mailjail/jails/web
zroot/mailjail/data/vmail
zroot/mailjail/data/db
zroot/mailjail/data/redis
zroot/mailjail/data/rspamd
zroot/mailjail/data/certs
zroot/mailjail/log
```

Separating `jails/*` and `data/*` enables a cleaner lifecycle:

- the jail root filesystem can be rebuilt
- persistent data remains separate
- snapshot and promotion strategy becomes clearer

## Networking Model

Baseline model:

- one bridge for internal jail communication
- separate static IP addresses for each jail
- only the required ports are exposed on the host
- default-deny east-west traffic between jails

Example:

```text
Host / bridge0: 10.77.0.1
postfix: 10.77.0.10
dovecot: 10.77.0.11
rspamd: 10.77.0.12
redis: 10.77.0.13
db: 10.77.0.14
web: 10.77.0.15
```

Recommended exposure policy:

- expose `25/tcp`, `465/tcp`, and `587/tcp` to `postfix`
- expose `143/tcp`, `993/tcp`, `110/tcp`, and `995/tcp` to `dovecot` only if those protocols are enabled
- expose `80/tcp` and `443/tcp` to `web`
- do not expose `redis`, `db`, or `rspamd` directly to the Internet

## CLI Execution Model

### `validate`

Checks:

- schema correctness
- IP conflicts
- module dependencies
- available host prerequisites
- DNS/TLS prerequisites when required
- Bastille installation and release availability
- PF/ZFS tooling availability

### `plan`

Generates a deterministic execution plan:

- which datasets will be created
- which jails will be created or changed
- which packages will be installed
- which configurations will be rendered
- which services will be restarted
- which Bastille actions will be executed

### `apply`

Executes the plan in dependency order and maintains rollback points where practical.

The initial `apply` model should be split into clearly bounded stages:

1. host preflight
2. Bastille bootstrap
3. ZFS dataset creation
4. network and firewall preparation
5. jail creation
6. package installation
7. configuration rendering
8. service start and health checks

## First Implementation Slice

The first vertical slice should solve only:

1. parsing and validation of the configuration
2. host preflight checks
3. Bastille bootstrap validation
4. creation of the dataset root
5. creation of one base jail
6. status report

Everything else should build on top of this mechanism rather than being implemented ad hoc for each module.
