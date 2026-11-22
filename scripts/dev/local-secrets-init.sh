#!/usr/bin/env bash
# Initialize owner-only local development secrets. Idempotent: never
# overwrites existing credentials. Prints paths only, never values.
# Usage: local-secrets-init.sh [SECRETS_DIR]
set -eu
set +x

DIR="${1:-${MINIAPP_SECRETS_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/miniapp}}"
mkdir -p "$DIR"
chmod 700 "$DIR"

new_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 33 | tr -d '\n'
  else
    head -c 48 /dev/urandom | base64 | tr -d '\n'
  fi
}

write_once() {
  local file="$1"
  if [ -f "$file" ]; then
    echo "keep: $file"
    chmod 600 "$file"
    return 0
  fi
  umask 077
  new_secret > "$file"
  chmod 600 "$file"
  echo "created: $file"
}

write_once "$DIR/db_password"
echo "secrets dir: $DIR (mode 700, files mode 600)"
