#!/usr/bin/env bash
# Interactively stage one allowlisted Fly secret via stdin import.
# Usage: secret-input.sh --env dev|prod --app NAME --secret NAME
# Never prints, logs, or persists the value. Always stages (--stage);
# applying staged secrets is a separate explicit step.
set -eu
set +x

ENV="" APP="" SECRET=""
while [ $# -gt 0 ]; do
  case "$1" in
    --env) ENV="$2"; shift 2;;
    --app) APP="$2"; shift 2;;
    --secret) SECRET="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done

case "$ENV" in
  dev)
    [ "$APP" = "miniapp-demo" ] || { echo "REFUSED: dev secrets target only miniapp-demo" >&2; exit 3; }
    ;;
  prod)
    [ -n "${MINIAPP_PROD_APP:-}" ] || { echo "REFUSED: MINIAPP_PROD_APP is not configured" >&2; exit 3; }
    [ "$APP" = "$MINIAPP_PROD_APP" ] || { echo "REFUSED: prod secrets target only \$MINIAPP_PROD_APP" >&2; exit 3; }
    ;;
  *) echo "REFUSED: --env must be dev or prod" >&2; exit 2;;
esac

case "$SECRET" in
  DATABASE_URL|DBOS_SYSTEM_DATABASE_URL) ;;
  *) echo "REFUSED: secret '$SECRET' is not allowlisted" >&2; exit 2;;
esac

identity=$(flyctl auth whoami 2>/dev/null) || { echo "REFUSED: flyctl authentication failed" >&2; exit 3; }
echo "target app : $APP"
echo "environment: $ENV"
echo "identity   : $identity"
echo "secret     : $SECRET (value will not be displayed)"

if [ "$ENV" = "prod" ]; then
  printf 'Type the production application name to confirm: '
  IFS= read -r confirm
  [ "$confirm" = "$APP" ] || { echo "REFUSED: confirmation mismatch" >&2; exit 3; }
fi

printf 'Enter value for %s (hidden): ' "$SECRET"
IFS= read -rs value
echo
[ -n "$value" ] || { echo "REFUSED: empty value" >&2; unset value; exit 2; }
printf '%s=%s\n' "$SECRET" "$value" | flyctl secrets import --app "$APP" --stage
rc=$?
unset value confirm
if [ "$rc" -ne 0 ]; then
  echo "FAILED: staging $SECRET on $APP" >&2
  exit "$rc"
fi
echo "staged: $SECRET on $APP (not yet deployed)"
