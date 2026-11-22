#!/usr/bin/env bash
# Fail closed when the target data volume does not match the selected
# PostgreSQL major version. Never creates, migrates, or deletes volumes.
# Usage: postgres-volume-check.sh EXPECTED_MAJOR [VOLUME_PREFIX]
# Override for an explicit fresh start: MINIAPP_ALLOW_FRESH_DB=1.
set -eu
set +x

EXPECTED_MAJOR="${1:?usage: postgres-volume-check.sh EXPECTED_MAJOR [VOLUME_PREFIX]}"
PREFIX="${2:-miniapp-pgdata}"
TARGET="$PREFIX-$EXPECTED_MAJOR"

volumes=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep "^$PREFIX-[0-9][0-9]*$" || true)
if [ -z "$volumes" ]; then
  echo "OK: no existing $PREFIX volumes; $TARGET will be created fresh."
  exit 0
fi
if printf '%s\n' "$volumes" | grep -qx "$TARGET"; then
  echo "OK: target volume $TARGET exists for major $EXPECTED_MAJOR."
  others=$(printf '%s\n' "$volumes" | grep -vx "$TARGET" || true)
  if [ -n "$others" ]; then
    echo "note: other major-version volumes preserved alongside: $(printf '%s' "$others" | tr '\n' ' ')"
  fi
  exit 0
fi
if [ "${MINIAPP_ALLOW_FRESH_DB:-0}" = "1" ]; then
  echo "OK: explicit fresh start acknowledged; existing volumes preserved: $(printf '%s' "$volumes" | tr '\n' ' ')"
  exit 0
fi
echo "REFUSED: image major $EXPECTED_MAJOR wants volume $TARGET," >&2
echo "but other major-version volumes already exist (data preserved):" >&2
printf '  - %s\n' "$volumes" >&2
echo "Migrate with dump/restore (see docs/development/local-postgres.md)," >&2
echo "or set MINIAPP_ALLOW_FRESH_DB=1 to start a new empty volume." >&2
exit 3
