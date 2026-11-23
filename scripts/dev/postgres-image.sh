#!/usr/bin/env bash
# Resolve and report the local PostgreSQL image decision.
# Usage: postgres-image.sh [IMAGE]
# Never pulls; dev:db:pull performs the deliberate pull.
# Prints: tag, server version, digest (when the image is local).
set -eu
set +x

IMAGE="${1:-${POSTGRES_IMAGE:-postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60}}"

lower=$(printf '%s' "$IMAGE" | tr '[:upper:]' '[:lower:]')
case "$lower" in
  *rc*|*beta*|*alpha*|*dev*|*nightly*|*snapshot*)
    echo "REFUSED: image '$IMAGE' looks like a pre-release; use a stable GA tag" >&2
    exit 2
    ;;
esac

echo "tag=$IMAGE"
if ! command -v docker >/dev/null 2>&1; then
  echo "server=unknown (no docker CLI)"
  echo "digest=unknown (not pulled)"
  exit 0
fi
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
  echo "server=unknown (image not present locally; run dev:db:pull)"
  echo "digest=unknown (not pulled)"
  exit 0
fi
digest=$(docker image inspect --format='{{index .RepoDigests 0}}' "$IMAGE" 2>/dev/null || true)
echo "digest=${digest:-unknown}"
pgdata=$(docker image inspect --format='{{range .Config.Env}}{{println .}}{{end}}' "$IMAGE" 2>/dev/null | grep '^PGDATA=' | cut -d= -f2- || true)
echo "default_pgdata=${pgdata:-unknown}"
server=$(docker run --rm --entrypoint postgres "$IMAGE" --version 2>/dev/null || true)
echo "server=${server:-unknown}"
major=$(printf '%s' "$server" | grep -oE '[0-9]+' | head -n 1 || true)
echo "major=${major:-unknown}"
