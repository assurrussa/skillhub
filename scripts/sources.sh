#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/sources.sh list [--tsv]' \
    '  sh scripts/sources.sh sync [source-name]' \
    '  sh scripts/sources.sh defaults list [--tsv]' \
    '  sh scripts/sources.sh defaults add <source-name>' \
    '  sh scripts/sources.sh add <path-or-git-url> [--name <name>] [--type path|git] [--ref <ref>] [--catalog <path>]' \
    '  sh scripts/sources.sh remove <name>'
}

skillhub_validate_sources_file
skillhub_validate_defaults_sources_file
cmd="${1:-list}"

derive_source_name() {
  location="$1"
  clean_location=${location%/}
  base=$(basename -- "$clean_location")
  base=${base%.git}
  printf '%s\n' "$base"
}

validate_source_name_or_exit() {
  name="$1"
  if ! skillhub_is_valid_source_name "$name"; then
    printf 'Invalid source name: %s\n' "$name" >&2
    exit 1
  fi
}

validate_source_type_or_exit() {
  type="$1"
  case "$type" in
    git|path)
      ;;
    *)
      printf 'Unsupported source type: %s\n' "$type" >&2
      exit 1
      ;;
  esac
}

validate_path_source_or_exit() {
  name="$1"
  location="$2"
  catalog="$3"

  if [ ! -d "$location" ]; then
    printf 'Path source does not exist: %s\n' "$location" >&2
    exit 1
  fi

  abs_location=$(CDPATH= cd -- "$location" && pwd)
  if [ ! -f "$abs_location/$catalog" ]; then
    printf 'Path source %s is missing catalog: %s\n' "$name" "$abs_location/$catalog" >&2
    exit 1
  fi

  printf '%s\n' "$abs_location"
}

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
    sources_file=$(skillhub_active_sources_file)
    trap 'rm -f "$sources_file"' EXIT HUP INT TERM
    if [ "$format" = "tsv" ]; then
      printf 'name\ttype\tlocation\tref\tcatalog\n'
    else
      printf '%-20s %-8s %-48s %-12s %s\n' "name" "type" "location" "ref" "catalog"
      printf '%-20s %-8s %-48s %-12s %s\n' "--------------------" "--------" "------------------------------------------------" "------------" "-------"
    fi
    listed=0
    while IFS='	' read -r name type location ref catalog extra; do
      case "$name" in
        ''|'#'*|'name')
          continue
          ;;
      esac
      if [ "$format" = "tsv" ]; then
        printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$type" "$location" "$ref" "$catalog"
      else
        printf '%-20s %-8s %-48s %-12s %s\n' "$name" "$type" "$location" "$ref" "$catalog"
      fi
      listed=$((listed + 1))
    done < "$sources_file"
    if [ "$listed" -eq 0 ] && [ "$format" != "tsv" ]; then
      printf 'No sources configured. Run: skillhub sources defaults list\n'
    fi
    ;;
  sync)
    wanted="${2:-}"
    sources_file=$(skillhub_active_sources_file)
    trap 'rm -f "$sources_file"' EXIT HUP INT TERM
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
    done < "$sources_file"

    if [ "$synced" -eq 0 ]; then
      if [ -n "$wanted" ]; then
        printf 'No source matched: %s\n' "$wanted" >&2
      else
        printf 'No sources configured. Run: skillhub sources defaults list\n' >&2
      fi
      exit 1
    fi
    ;;
  defaults)
    subcmd="${2:-list}"
    case "$subcmd" in
      list)
        if [ "$#" -ne 1 ] && [ "$#" -ne 2 ] && [ "$#" -ne 3 ]; then
          usage >&2
          exit 1
        fi
        format="table"
        if [ "${3:-}" = "--tsv" ]; then
          format="tsv"
        elif [ "$#" -eq 3 ]; then
          usage >&2
          exit 1
        fi
        if [ "$format" = "tsv" ]; then
          printf 'name\ttype\tlocation\tref\tcatalog\n'
        else
          printf '%-20s %-8s %-48s %-12s %s\n' "name" "type" "location" "ref" "catalog"
          printf '%-20s %-8s %-48s %-12s %s\n' "--------------------" "--------" "------------------------------------------------" "------------" "-------"
        fi
        while IFS='	' read -r name type location ref catalog extra; do
          case "$name" in
            ''|'#'*|'name')
              continue
              ;;
          esac
          if [ "$format" = "tsv" ]; then
            printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$type" "$location" "$ref" "$catalog"
          else
            printf '%-20s %-8s %-48s %-12s %s\n' "$name" "$type" "$location" "$ref" "$catalog"
          fi
        done < "$(skillhub_defaults_sources_file)"
        ;;
      add)
        if [ "$#" -ne 3 ]; then
          usage >&2
          exit 1
        fi
        name="$3"
        validate_source_name_or_exit "$name"
        source_row=$(skillhub_find_default_source_row "$name" || true)
        if [ -z "$source_row" ]; then
          printf 'Default source not found: %s\n' "$name" >&2
          exit 1
        fi
        if skillhub_source_exists "$name"; then
          printf 'Source already exists: %s\n' "$name" >&2
          exit 1
        fi
        skillhub_ensure_user_sources_file
        user_file=$(skillhub_user_sources_file)
        printf '%s\n' "$source_row" >> "$user_file"
        printf 'Added default source %s to %s\n' "$name" "$user_file"
        ;;
      -h|--help|help)
        usage
        ;;
      *)
        usage >&2
        exit 1
        ;;
    esac
    ;;
  add)
    if [ "$#" -lt 2 ]; then
      usage >&2
      exit 1
    fi
    location="$2"
    raw_location="$location"
    path_location=$(skillhub_abs_path "$location" "$(skillhub_caller_cwd)")
    shift 2
    name=""
    type=""
    ref=""
    catalog="catalog/skills.tsv"

    while [ "$#" -gt 0 ]; do
      case "$1" in
        --name)
          if [ "$#" -lt 2 ]; then
            printf '%s\n' '--name requires a value' >&2
            exit 1
          fi
          name="$2"
          shift 2
          ;;
        --type)
          if [ "$#" -lt 2 ]; then
            printf '%s\n' '--type requires a value' >&2
            exit 1
          fi
          type="$2"
          shift 2
          ;;
        --ref)
          if [ "$#" -lt 2 ]; then
            printf '%s\n' '--ref requires a value' >&2
            exit 1
          fi
          ref="$2"
          shift 2
          ;;
        --catalog)
          if [ "$#" -lt 2 ]; then
            printf '%s\n' '--catalog requires a value' >&2
            exit 1
          fi
          catalog="$2"
          shift 2
          ;;
        *)
          usage >&2
          exit 1
          ;;
      esac
    done

    if [ -z "$type" ]; then
      if [ -d "$path_location" ]; then
        type="path"
      else
        type="git"
      fi
    fi
    validate_source_type_or_exit "$type"

    if [ -z "$name" ]; then
      if [ "$type" = "path" ]; then
        name=$(derive_source_name "$path_location")
      else
        name=$(derive_source_name "$raw_location")
      fi
    fi
    validate_source_name_or_exit "$name"

    if [ -z "$catalog" ]; then
      printf 'Catalog path is required.\n' >&2
      exit 1
    fi

    case "$type" in
      path)
        location=$(validate_path_source_or_exit "$name" "$path_location" "$catalog")
        if [ -z "$ref" ]; then
          ref="-"
        fi
        ;;
      git)
        if [ -z "$ref" ]; then
          ref="main"
        fi
        ;;
    esac

    if skillhub_source_exists "$name"; then
      printf 'Source already exists: %s\n' "$name" >&2
      exit 1
    fi

    skillhub_ensure_user_sources_file
    user_file=$(skillhub_user_sources_file)
    printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$type" "$location" "$ref" "$catalog" >> "$user_file"
    printf 'Added source %s to %s\n' "$name" "$user_file"
    ;;
  remove)
    if [ "$#" -ne 2 ]; then
      usage >&2
      exit 1
    fi
    name="$2"
    validate_source_name_or_exit "$name"

    if ! skillhub_user_source_exists "$name"; then
      printf 'User source not found: %s\n' "$name" >&2
      exit 1
    fi

    user_file=$(skillhub_user_sources_file)
    tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-user-sources.XXXXXX")
    awk -F '	' -v wanted="$name" 'NR == 1 || $1 != wanted { print }' "$user_file" > "$tmp_file"
    mv "$tmp_file" "$user_file"
    printf 'Removed source %s from %s\n' "$name" "$user_file"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
