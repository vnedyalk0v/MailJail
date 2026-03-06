# ADR 0002: Security and Deployment Model

## Status

Accepted

## Date

2026-03-06

## Context

MailJail is intended to be an Internet-facing mail stack on FreeBSD. The project needs a deployment model that is secure by default, operationally realistic, and easy to reproduce on a new host.

## Decision

MailJail will use:

- VNET jails for Internet-facing and network-isolated services
- ZFS datasets separated between jail roots and persistent data
- default-deny firewall policy between jails
- Bastille-driven jail lifecycle
- one declarative YAML config file as the desired-state source of truth
- one host-local CLI as the control plane

The web edge will use Angie instead of Nginx.

## Security Baseline

The default security posture is:

- separate jail per major service
- no direct Internet exposure for internal services such as Redis and the database
- limited east-west traffic based on explicit allow rules
- least-privilege filesystem layout with separate persistent datasets
- host-level auditing and structured operation logs
- resource limits for each jail

## Deployment Model

The deployment workflow is:

1. validate config and host prerequisites
2. ensure Bastille and base release availability
3. create datasets
4. create network and firewall wiring
5. create jails
6. install packages
7. render configurations
8. start services
9. run health checks

## Why Not a Shell Script Collection

A shell-script-only system would be easy to start but hard to keep deterministic, testable, and safe as module count grows.

MailJail therefore adopts a planned-action CLI model rather than an unstructured script collection.
