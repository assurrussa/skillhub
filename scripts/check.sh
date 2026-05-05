#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

skillhub_validate_defaults_sources_file
skillhub_validate_targets_file

for script in install.sh bin/skillhub scripts/lib.sh scripts/sources.sh scripts/skills.sh scripts/installed.sh scripts/targets.sh scripts/recommend.sh scripts/check.sh; do
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
' "$(skillhub_defaults_sources_file)"; then
  exit 1
fi

default_checked=0
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

  default_checked=$((default_checked + 1))
done < "$(skillhub_defaults_sources_file)"

if [ "$default_checked" -eq 0 ]; then
  printf 'No default source presets configured.\n' >&2
  exit 1
fi

target_checked=0
seen_planned_target=0
while IFS='	' read -r id label status adapter description extra; do
  case "$id" in
    ''|'#'*|'id')
      continue
      ;;
  esac

  case "$id" in
    *[!a-z0-9_-]*)
      printf 'Invalid target id: %s\n' "$id" >&2
      exit 1
      ;;
  esac

  case "$status" in
    supported|planned)
      ;;
    *)
      printf 'Unsupported target status for %s: %s\n' "$id" "$status" >&2
      exit 1
      ;;
  esac

  if [ "$status" = "planned" ]; then
    seen_planned_target=1
  fi
  if [ "$status" = "supported" ] && [ "$seen_planned_target" -eq 1 ]; then
    printf 'Supported target must be listed before planned targets: %s\n' "$id" >&2
    exit 1
  fi

  if [ -z "$label" ] || [ -z "$adapter" ] || [ -z "$description" ]; then
    printf 'Target row has empty fields for %s\n' "$id" >&2
    exit 1
  fi

  if [ -n "${extra:-}" ]; then
    printf 'Target row has too many columns for %s\n' "$id" >&2
    exit 1
  fi

  target_checked=$((target_checked + 1))
done < "$(skillhub_targets_file)"

if [ "$target_checked" -eq 0 ]; then
  printf 'No targets configured.\n' >&2
  exit 1
fi

printf 'Validated %d default source preset(s).\n' "$default_checked"
printf 'Validated %d install target(s).\n' "$target_checked"
