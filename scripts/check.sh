#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

skillhub_validate_sources_file

for script in install.sh bin/skillhub scripts/lib.sh scripts/sources.sh scripts/skills.sh scripts/check.sh; do
  if [ ! -f "$repo_root/$script" ]; then
    printf 'Missing script: %s\n' "$script" >&2
    exit 1
  fi
  sh -n "$repo_root/$script"
done

if [ -f "$repo_root/go.mod" ]; then
  if ! command -v go >/dev/null 2>&1; then
    printf 'Go is required to validate this repository.\n' >&2
    exit 1
  fi
  (cd "$repo_root" && go test ./...)
fi

if ! awk -F '	' '
  NR == 1 { next }
  $1 == "" || $1 ~ /^#/ { next }
  seen[$1]++ { printf "Duplicate source: %s\n", $1 > "/dev/stderr"; exit 1 }
' "$(skillhub_sources_file)"; then
  exit 1
fi

checked=0
while IFS='	' read -r name type location ref catalog extra; do
  case "$name" in
    ''|'#'*|'name')
      continue
      ;;
  esac

  case "$name" in
    *[!a-z0-9_-]*)
      printf 'Invalid source name: %s\n' "$name" >&2
      exit 1
      ;;
  esac

  case "$type" in
    git|path)
      ;;
    *)
      printf 'Unsupported source type for %s: %s\n' "$name" "$type" >&2
      exit 1
      ;;
  esac

  if [ -z "$location" ] || [ -z "$ref" ] || [ -z "$catalog" ]; then
    printf 'Source row has empty fields for %s\n' "$name" >&2
    exit 1
  fi

  if [ -n "${extra:-}" ]; then
    printf 'Source row has too many columns for %s\n' "$name" >&2
    exit 1
  fi

  checked=$((checked + 1))
done < "$(skillhub_sources_file)"

if [ "$checked" -eq 0 ]; then
  printf 'No sources configured.\n' >&2
  exit 1
fi

printf 'Validated %d skill source(s).\n' "$checked"
