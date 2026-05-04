#!/bin/sh
set -eu

skillhub_repo_root() {
  CDPATH= cd -- "$(dirname -- "$0")/.." && pwd
}

skillhub_defaults_sources_file() {
  printf '%s/defaults/sources.tsv\n' "$(skillhub_repo_root)"
}

skillhub_targets_file() {
  printf '%s/targets/targets.tsv\n' "$(skillhub_repo_root)"
}

skillhub_config_root() {
  if [ -n "${SKILLHUB_CONFIG_DIR:-}" ]; then
    printf '%s\n' "$SKILLHUB_CONFIG_DIR"
    return
  fi

  if [ -n "${XDG_CONFIG_HOME:-}" ]; then
    printf '%s/skillhub\n' "$XDG_CONFIG_HOME"
    return
  fi

  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; set SKILLHUB_CONFIG_DIR explicitly.\n' >&2
    exit 1
  fi

  printf '%s/.config/skillhub\n' "$HOME"
}

skillhub_user_sources_file() {
  printf '%s/sources.tsv\n' "$(skillhub_config_root)"
}

skillhub_sources_file() {
  skillhub_user_sources_file
}

skillhub_sources_header() {
  printf 'name\ttype\tlocation\tref\tcatalog\n'
}

skillhub_targets_header() {
  printf 'id\tlabel\tstatus\tadapter\tdescription\n'
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

skillhub_caller_cwd() {
  if [ -n "${SKILLHUB_CALLER_CWD:-}" ]; then
    printf '%s\n' "$SKILLHUB_CALLER_CWD"
    return
  fi

  pwd
}

skillhub_abs_path() {
  path="$1"
  base_dir="${2:-$(skillhub_caller_cwd)}"

  case "$path" in
    /*)
      raw_path="$path"
      ;;
    *)
      raw_path="$base_dir/$path"
      ;;
  esac

  parent=$(dirname -- "$raw_path")
  name=$(basename -- "$raw_path")
  if [ -d "$parent" ]; then
    printf '%s/%s\n' "$(CDPATH= cd -- "$parent" && pwd)" "$name"
  else
    printf '%s\n' "$raw_path"
  fi
}

skillhub_default_codex_skills_dir() {
  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; set AGENT_SKILLS_DIR or use --target directory --dir <path>.\n' >&2
    exit 1
  fi

  printf '%s/.agents/skills\n' "$HOME"
}

skillhub_project_dir() {
  project="$1"
  if [ -n "$project" ]; then
    skillhub_abs_path "$project" "$(skillhub_caller_cwd)"
    return
  fi

  skillhub_abs_path "$(skillhub_caller_cwd)" /
}

skillhub_is_valid_target_id() {
  case "$1" in
    ''|*[!a-z0-9_-]*)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

skillhub_validate_targets_file() {
  file=$(skillhub_targets_file)

  if [ ! -f "$file" ]; then
    printf 'Missing targets registry: %s\n' "$file" >&2
    exit 1
  fi

  if ! awk 'NR == 1 { exit ($0 == "id\tlabel\tstatus\tadapter\tdescription" ? 0 : 1) }' "$file"; then
    printf 'Targets header must be: id<TAB>label<TAB>status<TAB>adapter<TAB>description in %s\n' "$file" >&2
    exit 1
  fi

  awk -F '	' '
    NR == 1 { next }
    $1 == "" || $1 ~ /^#/ { next }
    {
      if ($1 ~ /[^a-z0-9_-]/) {
        printf "Invalid target id: %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if ($3 != "supported" && $3 != "planned") {
        printf "Unsupported target status for %s: %s\n", $1, $3 > "/dev/stderr"
        exit 1
      }
      if ($4 == "") {
        printf "Target adapter is empty for %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if (NF != 5) {
        printf "Target row has wrong column count for %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if (seen[$1]++) {
        printf "Duplicate target: %s\n", $1 > "/dev/stderr"
        exit 1
      }
      count++
    }
    END {
      if (count == 0) {
        printf "No targets configured.\n" > "/dev/stderr"
        exit 1
      }
    }
  ' "$file"
}

skillhub_find_target_row() {
  wanted="$1"
  skillhub_validate_targets_file

  while IFS='	' read -r id label status adapter description extra; do
    case "$id" in
      ''|'#'*|'id')
        continue
        ;;
    esac

    if [ "$id" = "$wanted" ]; then
      printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$label" "$status" "$adapter" "$description"
      return 0
    fi
  done < "$(skillhub_targets_file)"

  return 1
}

skillhub_target_root() {
  target="${1:-codex}"
  scope="${2:-global}"
  project="${3:-}"
  dir="${4:-}"
  legacy_allowed="${5:-1}"
  scope_was_set="${6:-0}"

  if [ "$legacy_allowed" = "1" ] && [ -n "${AGENT_SKILLS_DIR:-}" ]; then
    printf '%s\n' "$AGENT_SKILLS_DIR"
    return
  fi

  case "$scope" in
    global|project)
      ;;
    *)
      printf 'Unsupported scope: %s\n' "$scope" >&2
      exit 1
      ;;
  esac

  if ! skillhub_is_valid_target_id "$target"; then
    printf 'Invalid target id: %s\n' "$target" >&2
    exit 1
  fi

  target_row=$(skillhub_find_target_row "$target" || true)
  if [ -z "$target_row" ]; then
    printf 'Unknown target: %s\n' "$target" >&2
    exit 1
  fi

  target_status=$(printf '%s\n' "$target_row" | awk -F '	' '{ print $3 }')
  target_adapter=$(printf '%s\n' "$target_row" | awk -F '	' '{ print $4 }')
  if [ "$target_status" != "supported" ]; then
    printf 'Target %s is planned but not supported yet: adapter not supported yet.\n' "$target" >&2
    exit 1
  fi
  if [ "$target_adapter" != "skill-dir" ]; then
    printf 'Target %s uses unsupported adapter: %s\n' "$target" "$target_adapter" >&2
    exit 1
  fi

  case "$target" in
    codex)
      case "$scope" in
        global)
          skillhub_default_codex_skills_dir
          ;;
        project)
          printf '%s/.agents/skills\n' "$(skillhub_project_dir "$project")"
          ;;
      esac
      ;;
    directory)
      if [ "$scope_was_set" = "1" ]; then
        printf 'Target directory does not support --scope.\n' >&2
        exit 1
      fi
      if [ -z "$dir" ]; then
        printf 'Target directory requires --dir <path>.\n' >&2
        exit 1
      fi
      skillhub_abs_path "$dir" "$(skillhub_caller_cwd)"
      ;;
    *)
      printf 'Target %s is not implemented by the skill-dir adapter.\n' "$target" >&2
      exit 1
      ;;
  esac
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

skillhub_is_valid_source_name() {
  case "$1" in
    ''|*[!a-z0-9_-]*)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

skillhub_validate_source_header() {
  file="$1"

  if [ ! -f "$file" ]; then
    printf 'Missing sources registry: %s\n' "$file" >&2
    exit 1
  fi

  if ! awk 'NR == 1 { exit ($0 == "name\ttype\tlocation\tref\tcatalog" ? 0 : 1) }' "$file"; then
    printf 'Sources header must be: name<TAB>type<TAB>location<TAB>ref<TAB>catalog in %s\n' "$file" >&2
    exit 1
  fi
}

skillhub_validate_defaults_sources_file() {
  skillhub_validate_source_header "$(skillhub_defaults_sources_file)"
}

skillhub_emit_source_rows() {
  file="$1"
  awk 'NR > 1 && $0 != "" && $0 !~ /^#/ { print }' "$file"
}

skillhub_write_active_sources() {
  user_file=$(skillhub_user_sources_file)

  if [ -f "$user_file" ]; then
    skillhub_validate_source_header "$user_file"
  fi

  skillhub_sources_header
  if [ -f "$user_file" ]; then
    skillhub_emit_source_rows "$user_file"
  fi | awk -F '	' '
    {
      if ($1 == "" || $1 ~ /[^a-z0-9_-]/) {
        printf "Invalid source name: %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if ($2 != "git" && $2 != "path") {
        printf "Unsupported source type for %s: %s\n", $1, $2 > "/dev/stderr"
        exit 1
      }
      if ($3 == "" || $4 == "" || $5 == "") {
        printf "Source row has empty fields for %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if (NF != 5) {
        printf "Source row has wrong column count for %s\n", $1 > "/dev/stderr"
        exit 1
      }
      if (seen[$1]++) {
        printf "Duplicate source: %s\n", $1 > "/dev/stderr"
        exit 1
      }
      print
    }
  '
}

skillhub_active_sources_file() {
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-sources.XXXXXX")
  if ! skillhub_write_active_sources > "$tmp_file"; then
    rm -f "$tmp_file"
    exit 1
  fi
  printf '%s\n' "$tmp_file"
}

skillhub_ensure_user_sources_file() {
  user_file=$(skillhub_user_sources_file)
  mkdir -p "$(dirname -- "$user_file")"
  if [ ! -f "$user_file" ]; then
    skillhub_sources_header > "$user_file"
  fi
}

skillhub_source_exists() {
  wanted="$1"
  sources_file=$(skillhub_active_sources_file)
  trap 'rm -f "$sources_file"' EXIT HUP INT TERM
  awk -F '	' -v wanted="$wanted" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$sources_file"
}

skillhub_default_source_exists() {
  wanted="$1"
  skillhub_validate_defaults_sources_file
  awk -F '	' -v wanted="$wanted" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$(skillhub_defaults_sources_file)"
}

skillhub_user_source_exists() {
  wanted="$1"
  user_file=$(skillhub_user_sources_file)
  [ -f "$user_file" ] || return 1
  awk -F '	' -v wanted="$wanted" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$user_file"
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
  sources_file=$(skillhub_active_sources_file)
  trap 'rm -f "$sources_file"' EXIT HUP INT TERM

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

skillhub_find_default_source_row() {
  wanted="$1"
  skillhub_validate_defaults_sources_file

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
  done < "$(skillhub_defaults_sources_file)"

  return 1
}

skillhub_validate_sources_file() {
  user_file=$(skillhub_user_sources_file)
  if [ -f "$user_file" ]; then
    skillhub_validate_source_header "$user_file"
  fi
}
