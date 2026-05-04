#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/skills.sh list [--tsv]' \
    '  sh scripts/skills.sh search [--tsv] <query>' \
    '  sh scripts/skills.sh install <skill-name>...' \
    '  sh scripts/skills.sh install --all'
}

skillhub_validate_sources_file

is_valid_skill_name() {
  case "$1" in
    ''|*[!a-z0-9_-]*)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

list_catalogs() {
  query="${1:-}"
  format="${2:-table}"

  if [ "$format" = "tsv" ]; then
    printf 'source\tname\tcategory\ttriggers\tdescription\n'
  else
    printf '%-20s %-28s %-16s %s\n' "source" "name" "category" "description"
    printf '%-20s %-28s %-16s %s\n' "--------------------" "----------------------------" "----------------" "-----------"
  fi

  found=0
  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac

    source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
    catalog_file="$source_path/$source_catalog"
    if [ ! -f "$catalog_file" ]; then
      printf 'Source %s is missing catalog: %s\n' "$source_name" "$catalog_file" >&2
      exit 1
    fi

    while IFS='	' read -r skill_name category triggers description catalog_extra; do
      case "$skill_name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      if ! is_valid_skill_name "$skill_name"; then
        printf 'Invalid catalog skill name from %s: %s\n' "$source_name" "$skill_name" >&2
        exit 1
      fi

      row="${source_name} ${skill_name} ${category} ${triggers} ${description}"
      if [ -n "$query" ] && ! printf '%s\n' "$row" | grep -i -e "$query" >/dev/null 2>&1; then
        continue
      fi

      if [ "$format" = "tsv" ]; then
        printf '%s\t%s\t%s\t%s\t%s\n' "$source_name" "$skill_name" "$category" "$triggers" "$description"
      else
        printf '%-20s %-28s %-16s %s\n' "$source_name" "$skill_name" "$category" "$description"
      fi
      found=$((found + 1))
    done < "$catalog_file"
  done < "$(skillhub_sources_file)"

  if [ "$found" -eq 0 ]; then
    if [ -n "$query" ]; then
      printf 'No skills matched query: %s\n' "$query" >&2
    else
      printf 'No skills found.\n' >&2
    fi
    exit 1
  fi
}

install_skill() {
  wanted="$1"
  if ! is_valid_skill_name "$wanted"; then
    printf 'Invalid skill name: %s\n' "$wanted" >&2
    exit 1
  fi

  target_root=$(skillhub_target_root)
  match_count=0
  match_source=""
  match_path=""

  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac

    source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
    catalog_file="$source_path/$source_catalog"

    if awk -F '	' -v wanted="$wanted" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$catalog_file"; then
      match_count=$((match_count + 1))
      match_source="$source_name"
      match_path="$source_path"
    fi
  done < "$(skillhub_sources_file)"

  if [ "$match_count" -eq 0 ]; then
    printf 'Unknown skill: %s\n' "$wanted" >&2
    exit 1
  fi

  if [ "$match_count" -gt 1 ]; then
    printf 'Skill name is ambiguous across sources: %s\n' "$wanted" >&2
    exit 1
  fi

  skill_dir="$match_path/skills/$wanted"
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    printf 'Source %s cataloged %s but SKILL.md is missing: %s\n' "$match_source" "$wanted" "$skill_dir" >&2
    exit 1
  fi

  mkdir -p "$target_root"
  rm -rf "$target_root/$wanted"
  cp -R "$skill_dir" "$target_root/$wanted"
  printf 'Installed %s from %s to %s\n' "$wanted" "$match_source" "$target_root/$wanted"
}

install_all() {
  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac

    source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
    catalog_file="$source_path/$source_catalog"
    while IFS='	' read -r skill_name category triggers description catalog_extra; do
      case "$skill_name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      install_skill "$skill_name"
    done < "$catalog_file"
  done < "$(skillhub_sources_file)"
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
    fi
    list_catalogs "" "$format"
    ;;
  search)
    if [ "$#" -lt 2 ]; then
      usage >&2
      exit 1
    fi
    shift
    format="table"
    if [ "${1:-}" = "--tsv" ]; then
      format="tsv"
      shift
    fi
    if [ "$#" -lt 1 ]; then
      usage >&2
      exit 1
    fi
    list_catalogs "$*" "$format"
    ;;
  install)
    if [ "$#" -lt 2 ]; then
      usage >&2
      exit 1
    fi
    shift
    if [ "$#" -eq 1 ] && [ "$1" = "--all" ]; then
      install_all
    else
      for skill_name in "$@"; do
        install_skill "$skill_name"
      done
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
