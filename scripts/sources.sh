#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/sources.sh list' \
    '  sh scripts/sources.sh sync [source-name]'
}

skillhub_validate_sources_file
cmd="${1:-list}"

case "$cmd" in
  list)
    printf '%-20s %-8s %-48s %-12s %s\n' "name" "type" "location" "ref" "catalog"
    printf '%-20s %-8s %-48s %-12s %s\n' "--------------------" "--------" "------------------------------------------------" "------------" "-------"
    while IFS='	' read -r name type location ref catalog extra; do
      case "$name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      printf '%-20s %-8s %-48s %-12s %s\n' "$name" "$type" "$location" "$ref" "$catalog"
    done < "$(skillhub_sources_file)"
    ;;
  sync)
    wanted="${2:-}"
    synced=0
    while IFS='	' read -r name type location ref catalog extra; do
      case "$name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      if [ -n "$wanted" ] && [ "$name" != "$wanted" ]; then
        continue
      fi
      source_path=$(skillhub_sync_source "$name" "$type" "$location" "$ref")
      printf 'Synced %s to %s\n' "$name" "$source_path"
      synced=$((synced + 1))
    done < "$(skillhub_sources_file)"

    if [ "$synced" -eq 0 ]; then
      printf 'No source matched: %s\n' "$wanted" >&2
      exit 1
    fi
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
