# Research Snapshot: 2026-03-06

## Scope

This document captures the initial external research used to shape MailJail decisions as of 2026-03-06.

## Key Conclusions

- Angie is a viable replacement for Nginx for the HTTP and HTTPS edge.
- Bastille is the selected jail orchestration layer.
- FreeBSD security and jail documentation support a design centered on VNET jails, ZFS datasets, firewall segmentation, auditing, and resource controls.
- Mailcow remains the closest functional reference point for the logical service set, but its Docker-centric operational model should not be copied directly.

## Angie

Angie is suitable for:

- HTTPS termination
- reverse proxying web applications
- autodiscover and autoconfig endpoints
- Rspamd or admin UI exposure

Angie is not intended to proxy SMTP, IMAP, or POP3. Those protocols should remain directly terminated by Postfix and Dovecot.

## FreeBSD Direction

The FreeBSD-native direction for MailJail is:

- Bastille for jail lifecycle
- VNET jails for network isolation
- ZFS datasets for root and persistent data separation
- PF-based network policy
- `rctl` and `racct` for jail-level resource controls
- host-side auditing and structured logs

## Mailcow Logical Reference

Mailcow currently provides a broad logical stack including:

- Postfix
- Dovecot
- Rspamd
- Redis
- MariaDB
- SOGo
- web edge
- Unbound
- ACME automation
- ClamAV
- Olefy
- watchdog and operational helpers

MailJail should borrow the service decomposition, not the container orchestration model.

## Source Links

- [Angie documentation](https://en.angie.software/angie/)
- [Angie package installation](https://en.angie.software/angie/docs/installation/oss_packages/)
- [FreeBSD Handbook: Jails](https://docs.freebsd.org/en/books/handbook/jails/)
- [FreeBSD Handbook: ZFS](https://docs.freebsd.org/en/books/handbook/zfs/)
- [FreeBSD Handbook: Security](https://docs.freebsd.org/en/books/handbook/security/)
- [FreeBSD Handbook: Audit](https://docs.freebsd.org/en/books/handbook/audit/)
- [FreeBSD security information](https://www.freebsd.org/security/)
- [FreeBSD release engineering schedule](https://www.freebsd.org/releng/)
- [BastilleBSD](https://bastillebsd.org/)
- [Mailcow documentation](https://docs.mailcow.email/)
- [Mailcow docker-compose](https://github.com/mailcow/mailcow-dockerized/blob/master/docker-compose.yml)
