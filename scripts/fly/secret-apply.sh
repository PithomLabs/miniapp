#!/usr/bin/env bash
# Apply staged Fly secrets after showing metadata and confirming.
# Usage: secret-apply.sh --env dev|prod --app NAME
# Lists metadata only; secret values are never retrieved or printed.
set -eu
set +x

ENV="" APP=""
while [ $# -gt 0 ]; do
  case "$1" in
    --env) ENV="$2"; shift 2;;
    --app) APP="$2"; shift 2;;
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

echo "staged secret metadata for $APP (values never shown):"
flyctl secrets list --app "$APP" || { echo "REFUSED: cannot list secrets" >&2; exit 3; }
echo "Applying will restart machines on $APP."
if [ "$ENV" = "prod" ]; then
  printf 'Type the production application name to apply: '
  IFS= read -r confirm
  [ "$confirm" = "$APP" ] || { echo "REFUSED: confirmation mismatch" >&2; exit 3; }
else
  printf 'Apply staged secrets to %s? [y/N] ' "$APP"
  IFS= read -r confirm
  [ "$confirm" = "y" ] || { echo "aborted"; exit 0; }
fi
flyctl secrets deploy --app "$APP"
echo "applied staged secrets on $APP"
