#!/usr/bin/env bash
# Phase 0 repository checks: formatting, pins, licenses, import boundaries.
# Fails closed on any violation. Run locally and in CI.
set -u

fail=0
say() { printf '%s\n' "$1"; }
bad() { say "CHECK-FAIL: $1"; fail=1; }

# 1. Formatting across the whole tree, including future packages.
fmt_out=$(gofmt -l . 2>&1); fmt_rc=$?
if [ "$fmt_rc" -ne 0 ]; then
  bad "gofmt execution failed: $fmt_out"
elif [ -n "$fmt_out" ]; then
  bad "gofmt reports unformatted files: $fmt_out"
fi

# 2. Pinned direct dependencies match the version register.
for pin in \
  "github.com/dbos-inc/dbos-transact-golang v1.9.0" \
  "github.com/riverqueue/river v0.51.2" \
  "github.com/riverqueue/river/riverdriver/riverpgxv5 v0.51.2" \
  "github.com/modelcontextprotocol/go-sdk v1.8.0" \
  "github.com/superfly/sprites-go v0.2.2" \
  "github.com/jackc/pgx/v5 v5.13.0" \
  "github.com/invopop/jsonschema v0.14.0" \
; do
  grep -qF "$pin" go.mod || bad "go.mod missing pin: $pin"
done

# 3. No enterprise-only DBOS modules without an approved decision.
if grep -rn "dbos-transact-golang/enterprise\|/conductor" --include="*.go" contract deploy internal spikes miniapps 2>/dev/null | grep -v "_test.go"; then
  bad "enterprise DBOS import detected (needs owner approval)"
fi

# 4. Framework never imports reference miniapps.
if grep -rn "PithomLabs/miniapp/miniapps" --include="*.go" contract deploy internal registry runtime executor capability auth tokens effects approvals audit transport testkit 2>/dev/null; then
  bad "framework package imports miniapps/"
fi

# 5. No provider credentials in test fixtures or schemas.
if grep -rniE "password|secret|bearer|api[_-]?key" --include="*.json" contract/schemas contract/testdata 2>/dev/null | grep -viE "password authentication|secret boundary"; then
  bad "possible credential in fixtures/schemas"
fi

if [ "$fail" -eq 0 ]; then
  say "ALL-CHECKS-PASS"
fi
exit "$fail"
