# Stack Mapping

## Goal

This document maps the current Mailcow-style logical stack to the intended MailJail architecture on FreeBSD.

## Logical Mapping

### Mail Path

- `postfix`: SMTP ingress, submission, outbound delivery
- `dovecot`: IMAP, POP3, LMTP, authentication helpers
- `rspamd`: content filtering and policy engine
- `redis`: Rspamd state and cache

### Control Plane

- `db`: metadata, domains, accounts, aliases, and policy data
- `web`: Angie reverse proxy, admin UI, user UI, autodiscover, autoconfig

### Support Services

- `acme`: certificate issuance and renewal
- `dns`: prerequisite validation and optional local resolver integration
- `unbound`: optional local recursive resolver
- `clamav`: optional antivirus service
- `olefy`: optional document scanning helper
- `sogo`: optional groupware layer
- `tlspol`: optional outbound MTA-STS policy helper

## Recommended Product Profiles

### `core`

Includes:

- postfix
- dovecot
- rspamd
- redis
- acme
- web

Use when the goal is a secure and relatively lightweight mail server.

### `groupware`

Includes everything in `core` plus:

- db
- sogo
- unbound
- clamav
- olefy
- tlspol

Use when the goal is a more Mailcow-like all-in-one system.

## Angie Position

Angie should terminate HTTPS and proxy only HTTP-based services.

Angie should not be placed in front of SMTP, IMAP, or POP3 traffic. Those protocols should remain directly terminated by Postfix and Dovecot.
