#!/bin/sh
set -eu

skillhub_repo_root() {
  CDPATH= cd -- "$(dirname -- "$0")/.." && pwd
}

skillhub_sources_file() {
  printf '%s/sources/sources.tsv\n' "$(skillhub_repo_root)"
}

skillhub_cache_root() {
  if [ -n "${SKILLHUB_CACHE_DIR:-}" ]; then
    printf '%s\n' "$SKILLHUB_CACHE_DIR"
    return
  fi

  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; set SKILLHUB_CACHE_DIR explicitly.\n' >&2
    exit 1
  fi

  printf '%s/.cache/skillhub\n' "$HOME"
}

skillhub_target_root() {
  if [ -n "${AGENT_SKILLS_DIR:-}" ]; then
    printf '%s\n' "$AGENT_SKILLS_DIR"
    return
  fi

  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; set AGENT_SKILLS_DIR explicitly.\n' >&2
    exit 1
  fi

  printf '%s/.agents/skills\n' "$HOME"
}

skillhub_source_path() {
  source_name="$1"
  source_type="$2"
  source_location="$3"

  if [ "$source_name" = "agent-rules" ] && [ -n "${SKILLHUB_AGENT_RULES_PATH:-}" ]; then
    printf '%s\n' "$SKILLHUB_AGENT_RULES_PATH"
    return
  fi

  case "$source_type" in
    path)
      case "$source_location" in
        /*)
          printf '%s\n' "$source_location"
          ;;
        *)
          printf '%s/%s\n' "$(skillhub_repo_root)" "$source_location"
          ;;
      esac
      ;;
    git)
      printf '%s/sources/%s\n' "$(skillhub_cache_root)" "$source_name"
      ;;
    *)
      printf 'Unsupported source type for %s: %s\n' "$source_name" "$source_type" >&2
      exit 1
      ;;
  esac
}

skillhub_sync_source() {
  source_name="$1"
  source_type="$2"
  source_location="$3"
  source_ref="$4"

  source_path=$(skillhub_source_path "$source_name" "$source_type" "$source_location")

  if [ "$source_name" = "agent-rules" ] && [ -n "${SKILLHUB_AGENT_RULES_PATH:-}" ]; then
    if [ ! -d "$source_path" ]; then
      printf 'Local source override does not exist: %s\n' "$source_path" >&2
      exit 1
    fi
    printf '%s\n' "$source_path"
    return
  fi

  case "$source_type" in
    path)
      if [ ! -d "$source_path" ]; then
        printf 'Path source does not exist: %s\n' "$source_path" >&2
        exit 1
      fi
      ;;
    git)
      mkdir -p "$(dirname -- "$source_path")"
      if [ -d "$source_path/.git" ]; then
        git -C "$source_path" fetch --quiet --prune >&2
      else
        git clone --quiet "$source_location" "$source_path" >&2
      fi
      git -C "$source_path" checkout --quiet "$source_ref" >&2
      git -C "$source_path" pull --quiet --ff-only >&2
      ;;
  esac

  printf '%s\n' "$source_path"
}

skillhub_find_source_row() {
  wanted="$1"
  sources_file=$(skillhub_sources_file)

  while IFS='	' read -r name type location ref catalog extra; do
    case "$name" in
      ''|'#'*|'name')
        continue
        ;;
    esac

    if [ "$name" = "$wanted" ]; then
      printf '%s\t%s\t%s\t%s\t%s\n' "$name" "$type" "$location" "$ref" "$catalog"
      return 0
    fi
  done < "$sources_file"

  return 1
}

skillhub_validate_sources_file() {
  sources_file=$(skillhub_sources_file)

  if [ ! -f "$sources_file" ]; then
    printf 'Missing sources registry: %s\n' "$sources_file" >&2
    exit 1
  fi

  if ! awk 'NR == 1 { exit ($0 == "name\ttype\tlocation\tref\tcatalog" ? 0 : 1) }' "$sources_file"; then
    printf 'Sources header must be: name<TAB>type<TAB>location<TAB>ref<TAB>catalog\n' >&2
    exit 1
  fi
}
