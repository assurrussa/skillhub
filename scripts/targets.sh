#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/targets.sh list [--tsv]' \
    '  sh scripts/targets.sh detect'
}

skillhub_validate_targets_file

list_targets() {
  format="${1:-table}"

  if [ "$format" = "tsv" ]; then
    skillhub_targets_header
  else
    printf '%-14s %-12s %-10s %-10s %s\n' "id" "label" "status" "adapter" "description"
    printf '%-14s %-12s %-10s %-10s %s\n' "--------------" "------------" "----------" "----------" "-----------"
  fi

  while IFS='	' read -r id label status adapter description extra; do
    case "$id" in
      ''|'#'*|'id')
        continue
        ;;
    esac

    if [ "$format" = "tsv" ]; then
      printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$label" "$status" "$adapter" "$description"
    else
      printf '%-14s %-12s %-10s %-10s %s\n' "$id" "$label" "$status" "$adapter" "$description"
    fi
  done < "$(skillhub_targets_file)"
}

detect_targets() {
  printf '%-10s %-8s %s\n' "target" "scope" "path"
  printf '%-10s %-8s %s\n' "----------" "--------" "----"
  printf '%-10s %-8s %s\n' "codex" "global" "$(skillhub_target_root codex global "" "" 0)"
  printf '%-10s %-8s %s\n' "codex" "project" "$(skillhub_target_root codex project "" "" 0)"
  printf '%-10s %-8s %s\n' "directory" "custom" "requires --dir <path>"
}

cmd="${1:-list}"
case "$cmd" in
  list)
    format="table"
    if [ "${2:-}" = "--tsv" ]; then
      format="tsv"
      if [ "$#" -gt 2 ]; then
        usage >&2
        exit 1
      fi
    elif [ "$#" -gt 1 ]; then
      usage >&2
      exit 1
    fi
    list_targets "$format"
    ;;
  detect)
    if [ "$#" -ne 1 ]; then
      usage >&2
      exit 1
    fi
    detect_targets
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
