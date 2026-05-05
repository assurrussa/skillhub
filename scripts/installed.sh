#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/installed.sh list [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--tsv]' \
    '  sh scripts/installed.sh uninstall <skill> [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--force]'
}

installed_skill_name() {
  wanted="$1"
  case "$wanted" in
    */*)
      wanted=${wanted##*/}
      ;;
  esac

  case "$wanted" in
    ''|*[!a-z0-9_-]*)
      printf 'Invalid installed skill name: %s\n' "$wanted" >&2
      exit 1
      ;;
  esac

  printf '%s\n' "$wanted"
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

uninstall_installed() {
  shift

  if [ "$#" -lt 1 ]; then
    usage >&2
    exit 1
  fi

  skill_name=$(installed_skill_name "$1")
  shift

  target="codex"
  scope="global"
  project=""
  dir=""
  legacy_allowed=1
  scope_was_set=0
  force=0

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
      --force)
        force=1
        shift
        ;;
      *)
        usage >&2
        exit 1
        ;;
    esac
  done

  target_root=$(skillhub_target_root "$target" "$scope" "$project" "$dir" "$legacy_allowed" "$scope_was_set")
  skill_dir="$target_root/$skill_name"

  if [ ! -d "$skill_dir" ]; then
    printf 'Installed skill not found: %s\n' "$skill_dir" >&2
    exit 1
  fi

  if [ ! -f "$skill_dir/SKILL.md" ]; then
    printf 'Refusing to uninstall non-skill directory: %s\n' "$skill_dir" >&2
    exit 1
  fi

  if [ ! -f "$skill_dir/.skillhub.json" ] && [ "$force" -ne 1 ]; then
    printf 'Refusing to uninstall unmanaged skill: %s\n' "$skill_dir" >&2
    printf 'Use --force to remove it anyway.\n' >&2
    exit 1
  fi

  rm -rf "$skill_dir"
  printf 'Uninstalled %s from %s\n' "$skill_name" "$target_root"
}

cmd="${1:-list}"
case "$cmd" in
  list)
    list_installed "$@"
    ;;
  uninstall)
    uninstall_installed "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
