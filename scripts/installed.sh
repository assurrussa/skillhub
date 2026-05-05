#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/installed.sh list [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--tsv]'
}

list_installed() {
  shift

  target="codex"
  scope="global"
  project=""
  dir=""
  format="table"
  legacy_allowed=1
  scope_was_set=0

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --target)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--target requires a value' >&2
          exit 1
        fi
        target="$2"
        legacy_allowed=0
        shift 2
        ;;
      --scope)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--scope requires a value' >&2
          exit 1
        fi
        scope="$2"
        legacy_allowed=0
        scope_was_set=1
        shift 2
        ;;
      --project)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--project requires a value' >&2
          exit 1
        fi
        project="$2"
        legacy_allowed=0
        shift 2
        ;;
      --dir)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--dir requires a value' >&2
          exit 1
        fi
        dir="$2"
        legacy_allowed=0
        shift 2
        ;;
      --tsv)
        format="tsv"
        shift
        ;;
      *)
        usage >&2
        exit 1
        ;;
    esac
  done

  target_root=$(skillhub_target_root "$target" "$scope" "$project" "$dir" "$legacy_allowed" "$scope_was_set")
  metadata_scope="$scope"
  if [ "$target" = "directory" ]; then
    metadata_scope="custom"
  fi

  if [ "$format" = "tsv" ]; then
    skillhub_emit_installed_tsv "$target_root" "$target" "$metadata_scope"
  else
    skillhub_emit_installed_table "$target_root" "$target" "$metadata_scope"
  fi
}

cmd="${1:-list}"
case "$cmd" in
  list)
    list_installed "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
