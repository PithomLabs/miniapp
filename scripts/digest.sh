#!/usr/bin/env bash
# Deterministic source/tree digests (X-18/CR-11). Method: sorted
# sha256sum manifest over a defined file set; the digest is the
# SHA-256 of the manifest. LC_ALL=C throughout for byte-stable order.
#
#   scripts/digest.sh tree [dir]      full tree except .git and digests/
#   scripts/digest.sh source [dir]    executable content under test:
#                                     *.go, *.sql, .github/*, scripts/*,
#                                     */testdata/*, contract/schemas/*,
#                                     go.mod, go.sum
#   scripts/digest.sh files <paths>   digest over an explicit file list
#                                     (test-harness binding; paths must exist)
#
# Output: "<digest>  <scope>" on stdout. With MANIFEST_OUT set, the
# manifest is also written there; docs/evidence/digests/ is pruned
# from every digest, so manifests kept there never self-include.
set -u

manifest_of_find() {
  # $1 = dir; rest = find predicates producing NUL-separated paths.
  local dir=$1; shift
  ( cd "$dir" && find . -path ./.git -prune -o -path ./docs/evidence/digests -prune -o "$@" -print0 ) \
    | LC_ALL=C sort -z | xargs -0 -I{} sh -c 'cd "$0" && sha256sum "$1"' "$dir" {} 2>/dev/null \
    | sed 's| \./| |' | LC_ALL=C sort -k2
}

emit() {
  # $1 = scope label; manifest on stdin.
  local scope=$1 manifest digest
  manifest=$(cat)
  if [ -n "${MANIFEST_OUT:-}" ]; then
    printf '%s\n' "$manifest" > "$MANIFEST_OUT"
  fi
  digest=$(printf '%s\n' "$manifest" | sha256sum | cut -d' ' -f1)
  printf '%s  %s\n' "$digest" "$scope"
}

case "${1:-}" in
  tree)
    manifest_of_find "${2:-.}" -type f | emit "tree:${2:-.}"
    ;;
  source)
    manifest_of_find "${2:-.}" -type f \( \
      -name '*.go' -o -name '*.sql' -o -name 'go.mod' -o -name 'go.sum' \
      -o -path './.github/*' -o -path './scripts/*' \
      -o -path '*/testdata/*' -o -path './contract/schemas/*' \
    \) | emit "source:${2:-.}"
    ;;
  files)
    shift
    [ "$#" -gt 0 ] || { echo "digest.sh files: need paths" >&2; exit 2; }
    for f in "$@"; do [ -f "$f" ] || { echo "digest.sh files: missing $f" >&2; exit 2; }; done
    printf '%s\n' "$@" | LC_ALL=C sort | xargs sha256sum | emit "files:$*"
    ;;
  *)
    echo "usage: $0 {tree|source|files} [dir|paths...]" >&2
    exit 2
    ;;
esac
