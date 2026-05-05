#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/targets.sh list [--tsv]' \
    '  sh scripts/targets.sh detect [--tsv] [--project <path>]'
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

detect_row() {
  target="$1"
  scope="$2"
  status="$3"
  path="$4"
  format="$5"

  if [ "$path" = "-" ]; then
    exists="-"
    skills="-"
    managed="-"
  else
    exists=$(skillhub_dir_exists_label "$path")
    skills=$(skillhub_count_skill_dirs "$path")
    managed=$(skillhub_count_managed_skill_dirs "$path")
  fi

  if [ "$format" = "tsv" ]; then
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$target" "$scope" "$status" "$path" "$exists" "$skills" "$managed"
  else
    printf '%-12s %-8s %-10s %-6s %-6s %-7s %s\n' "$target" "$scope" "$status" "$exists" "$skills" "$managed" "$path"
  fi
}

detect_targets() {
  format="$1"
  project="$2"

  if [ "$format" = "tsv" ]; then
    printf 'target\tscope\tstatus\tpath\texists\tskills\tmanaged\n'
  else
    printf '%-12s %-8s %-10s %-6s %-6s %-7s %s\n' "target" "scope" "status" "exists" "skills" "managed" "path"
    printf '%-12s %-8s %-10s %-6s %-6s %-7s %s\n' "------------" "--------" "----------" "------" "------" "-------" "----"
  fi

  while IFS='	' read -r id label status adapter description extra; do
    case "$id" in
      ''|'#'*|'id')
        continue
        ;;
    esac

    if [ "$id" = "directory" ]; then
      detect_row "$id" "custom" "$status" "-" "$format"
      continue
    fi

    if [ "$status" = "supported" ] && [ "$adapter" = "skill-dir" ]; then
      detect_row "$id" "global" "$status" "$(skillhub_target_root "$id" global "" "" 0)" "$format"
      detect_row "$id" "project" "$status" "$(skillhub_target_root "$id" project "$project" "" 0)" "$format"
    fi
  done < "$(skillhub_targets_file)"
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
    shift
    format="table"
    project=""
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --tsv)
          format="tsv"
          shift
          ;;
        --project)
          if [ "$#" -lt 2 ]; then
            printf '%s\n' '--project requires a value' >&2
            exit 1
          fi
          project="$2"
          shift 2
          ;;
        *)
          usage >&2
          exit 1
          ;;
      esac
    done
    detect_targets "$format" "$project"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
