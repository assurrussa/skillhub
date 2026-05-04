#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/skills.sh list [--tsv]' \
    '  sh scripts/skills.sh search [--tsv] <query>' \
    '  sh scripts/skills.sh install [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [<source>/]<skill-name>...' \
    '  sh scripts/skills.sh install [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] --all'
}

skillhub_validate_sources_file
sources_file=$(skillhub_active_sources_file)
trap 'rm -f "$sources_file"' EXIT HUP INT TERM

no_sources_message() {
  printf 'No sources configured. Run: skillhub sources defaults list\n' >&2
}

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

skill_install_name() {
  wanted="$1"
  case "$wanted" in
    */*)
      wanted_source=${wanted%%/*}
      wanted_skill=${wanted#*/}
      case "$wanted_source" in
        ''|*[!a-z0-9_-]*)
          printf 'Invalid source name in qualified skill: %s\n' "$wanted" >&2
          exit 1
          ;;
      esac
      case "$wanted_skill" in
        ''|*/*)
          printf 'Invalid qualified skill name: %s\n' "$wanted" >&2
          exit 1
          ;;
      esac
      ;;
    *)
      wanted_skill="$wanted"
      ;;
  esac

  if ! is_valid_skill_name "$wanted_skill"; then
    printf 'Invalid skill name: %s\n' "$wanted_skill" >&2
    exit 1
  fi

  printf '%s\n' "$wanted_skill"
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
  source_count=0
  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac
    source_count=$((source_count + 1))

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
  done < "$sources_file"

  if [ "$source_count" -eq 0 ]; then
    no_sources_message
    exit 1
  fi

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
  target_root="$2"
  wanted_source=""
  wanted_skill=$(skill_install_name "$wanted")
  case "$wanted" in
    */*)
      wanted_source=${wanted%%/*}
      ;;
  esac

  match_count=0
  match_source=""
  match_path=""
  source_count=0
  considered_source_count=0

  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac
    source_count=$((source_count + 1))

    if [ -n "$wanted_source" ] && [ "$source_name" != "$wanted_source" ]; then
      continue
    fi
    considered_source_count=$((considered_source_count + 1))

    source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
    catalog_file="$source_path/$source_catalog"

    if awk -F '	' -v wanted="$wanted_skill" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$catalog_file"; then
      match_count=$((match_count + 1))
      match_source="$source_name"
      match_path="$source_path"
    fi
  done < "$sources_file"

  if [ "$source_count" -eq 0 ]; then
    no_sources_message
    exit 1
  fi

  if [ -n "$wanted_source" ] && [ "$considered_source_count" -eq 0 ]; then
    printf 'Unknown source: %s\n' "$wanted_source" >&2
    exit 1
  fi

  if [ "$match_count" -eq 0 ]; then
    printf 'Unknown skill: %s\n' "$wanted" >&2
    exit 1
  fi

  if [ "$match_count" -gt 1 ]; then
    printf 'Skill name is ambiguous across sources: %s\n' "$wanted" >&2
    exit 1
  fi

  skill_dir="$match_path/skills/$wanted_skill"
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    printf 'Source %s cataloged %s but SKILL.md is missing: %s\n' "$match_source" "$wanted_skill" "$skill_dir" >&2
    exit 1
  fi

  mkdir -p "$target_root"
  rm -rf "$target_root/$wanted_skill"
  cp -R "$skill_dir" "$target_root/$wanted_skill"
  printf 'Installed %s from %s to %s\n' "$wanted_skill" "$match_source" "$target_root/$wanted_skill"
}

install_all() {
  target_root="$1"
  source_count=0
  while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
    case "$source_name" in
      ''|'#'*|'name')
        continue
        ;;
    esac
    source_count=$((source_count + 1))

    source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
    catalog_file="$source_path/$source_catalog"
    while IFS='	' read -r skill_name category triggers description catalog_extra; do
      case "$skill_name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      install_skill "$skill_name" "$target_root"
    done < "$catalog_file"
  done < "$sources_file"

  if [ "$source_count" -eq 0 ]; then
    no_sources_message
    exit 1
  fi
}

install_requested() {
  shift

  target="codex"
  scope="global"
  project=""
  dir=""
  legacy_allowed=1
  scope_was_set=0
  all=0
  skill_names=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --all)
        all=1
        shift
        ;;
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
      -*)
        usage >&2
        exit 1
        ;;
      *)
        skill_names="${skill_names}${skill_names:+ }$1"
        shift
        ;;
    esac
  done

  if [ "$all" -eq 1 ] && [ -n "$skill_names" ]; then
    printf '%s\n' '--all cannot be combined with skill names' >&2
    exit 1
  fi

  if [ "$all" -eq 0 ] && [ -z "$skill_names" ]; then
    usage >&2
    exit 1
  fi

  target_root=$(skillhub_target_root "$target" "$scope" "$project" "$dir" "$legacy_allowed" "$scope_was_set")
  if [ "$all" -eq 1 ]; then
    install_all "$target_root"
    return
  fi

  seen_install_names=""
  for skill_name in $skill_names; do
    install_name=$(skill_install_name "$skill_name")
    case " $seen_install_names " in
      *" $install_name "*)
        printf 'Multiple requested skills install to the same target name: %s\n' "$install_name" >&2
        exit 1
        ;;
    esac
    seen_install_names="${seen_install_names}${seen_install_names:+ }$install_name"
  done

  for skill_name in $skill_names; do
    install_skill "$skill_name" "$target_root"
  done
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
    install_requested "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
