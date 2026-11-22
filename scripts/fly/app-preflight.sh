#!/usr/bin/env bash
# Preflight checks for a Fly application target. Fails closed.
# Usage: app-preflight.sh --app NAME [--org ORG]
set -eu
set +x

APP="" ORG=""
while [ $# -gt 0 ]; do
  case "$1" in
    --app) APP="$2"; shift 2;;
    --org) ORG="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
[ -n "$APP" ] || { echo "REFUSED: --app is required" >&2; exit 2; }

identity=$(flyctl auth whoami 2>/dev/null) || { echo "REFUSED: flyctl authentication failed" >&2; exit 3; }
apps_json=$(flyctl apps list --json 2>/dev/null) || { echo "REFUSED: cannot list Fly applications" >&2; exit 3; }
app_org=$(printf '%s' "$apps_json" | python3 -c "
import json,sys
for a in json.load(sys.stdin):
    if a.get('Name') == '$APP':
        print(a.get('Organization',{}).get('Slug',''))
" 2>/dev/null || true)
[ -n "$app_org" ] || { echo "REFUSED: Fly application '$APP' not found or not accessible" >&2; exit 3; }
if [ -n "$ORG" ] && [ "$app_org" != "$ORG" ]; then
  echo "REFUSED: app '$APP' belongs to org '$app_org', expected '$ORG'" >&2
  exit 3
fi
echo "OK: app=$APP org=$app_org identity=$identity"
