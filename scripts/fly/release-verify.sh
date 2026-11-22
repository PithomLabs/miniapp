#!/usr/bin/env bash
# Read-only release inspection for a Fly application.
# Usage: release-verify.sh --app NAME [--expect-deployed]
# Never deploys, restarts, or changes anything.
set -eu
set +x

APP="" EXPECT=""
while [ $# -gt 0 ]; do
  case "$1" in
    --app) APP="$2"; shift 2;;
    --expect-deployed) EXPECT=1; shift;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
[ -n "$APP" ] || { echo "REFUSED: --app is required" >&2; exit 2; }

flyctl status --app "$APP" || { echo "REFUSED: cannot inspect $APP" >&2; exit 3; }
echo "--- machines ---"
flyctl machine list --app "$APP" 2>/dev/null || echo "(no machines listed)"
if [ -n "$EXPECT" ]; then
  running=$(flyctl machine list --app "$APP" --quiet 2>/dev/null | wc -l | tr -d ' ')
  if [ "${running:-0}" -eq 0 ]; then
    echo "REFUSED: $APP has no machines; nothing deployed to verify" >&2
    exit 3
  fi
  echo "OK: $APP has $running machine(s)"
fi
