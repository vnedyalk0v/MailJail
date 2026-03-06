# Architecture

## System layout

MailJail splits the mail stack into independent jails with narrowly scoped responsibilities.

### Host Responsibilities

- ZFS pool and dataset management
- bridge and VNET networking management
- packet filter rules
- jail templates and image lifecycle
- secrets and certificates
- orchestration state

### Jail Responsibilities

- installation of service-specific packages
- rendering of service configuration files
- startup and supervision of the specific service

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

## CLI Execution Model

### `validate`

Checks:

- schema correctness
- IP conflicts
- module dependencies
- available host prerequisites
- DNS/TLS prerequisites when required

### `plan`

Generates a deterministic execution plan:

- which datasets will be created
- which jails will be created or changed
- which packages will be installed
- which configurations will be rendered
- which services will be restarted

### `apply`

Executes the plan in dependency order and maintains rollback points where practical.

## First Implementation Slice

The first vertical slice should solve only:

1. parsing and validation of the configuration
2. host preflight checks
3. creation of the dataset root
4. creation of one base jail
5. status report

Everything else should build on top of this mechanism rather than being implemented ad hoc for each module.
