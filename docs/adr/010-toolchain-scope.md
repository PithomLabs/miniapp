# ADR-010: Miniapp-Toolchain Scope Deferral

Status: Adopted.

## Context

A durable miniapp that manages agent/tool operations risks duplicating
the harness reasoning loop and turning the platform into another
general-purpose agent runtime.

## Decision

`agent-toolchain` and any AI miniapp generator work require separate
scope justification with explicit dependency and boundary analysis.
Default: deferred past the initial catalog. Trusted Go execution in any
such miniapp defaults to Machine placement.

## Consequences

- The initial catalog proves the framework without absorbing
  agent-runtime responsibilities.
