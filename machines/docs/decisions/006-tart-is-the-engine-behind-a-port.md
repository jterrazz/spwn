# ADR-006: Tart is the engine, behind a port

**Status:** Superseded by [ADR-014](014-families-and-contracts.md)
**Date:** 2026-10-31

## Context

Development moves into disposable VMs, with Linux guests for dev and macOS guests for Xcode and simulators. The candidates (Tart, Lima, OrbStack, Apple container, UTM, Parallels, VirtualBuddy) were compared on performance, RAM, isolation, features and SSD use. All sit on Apple's Virtualization.framework, so they differ in images, isolation defaults, disk economy and scripting.

## Decision

Tart is the engine for both guest kinds: OCI images, APFS clones that cost no disk, nothing shared with the host by default, JSON output. Station reaches it only through an `Engine` port, so Lima can replace it for Linux without touching a command. Tart's storage stays `~/.tart`; station owns only the VMs its records name and the `station-image-` and `station-build-` ones.

## Consequences

- One adapter and one image pipeline for Linux and macOS.
- Tart's licence and ownership are a risk the port contains.
- Tart has no cloud-init, so the recipe makes the first boot itself.
- On Tart's shared NAT a VM reaches its host's services and its sibling VMs; network isolation waits in the TODO.
