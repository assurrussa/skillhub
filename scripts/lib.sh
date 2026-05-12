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

skillhub_installed_header() {
  printf 'source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\tsource_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at\n'
}

skillhub_installed_registry_file() {
  printf '%s/installed.tsv\n' "$(skillhub_config_root)"
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

skillhub_source_state_dir() {
  printf '%s/source-state\n' "$(skillhub_cache_root)"
}

skillhub_source_sync_state_file() {
  source_name="$1"
  printf '%s/%s.synced_at\n' "$(skillhub_source_state_dir)" "$source_name"
}

skillhub_generated_source_path() {
  source_name="$1"
  printf '%s/generated-sources/%s\n' "$(skillhub_cache_root)" "$source_name"
}

skillhub_source_ttl_seconds() {
  printf '600\n'
}

skillhub_now_seconds() {
  date +%s
}

skillhub_mark_source_synced() {
  source_name="$1"
  state_file=$(skillhub_source_sync_state_file "$source_name")
  mkdir -p "$(dirname -- "$state_file")"
  skillhub_now_seconds > "$state_file"
}

skillhub_clear_source_cache() {
  source_name="$1"
  cache_sources_dir="$(skillhub_cache_root)/sources"
  source_path="$cache_sources_dir/$source_name"
  generated_path=$(skillhub_generated_source_path "$source_name")

  case "$source_path" in
    "$cache_sources_dir"/*)
      rm -rf "$source_path"
      ;;
  esac
  rm -rf "$generated_path"
  rm -f "$(skillhub_source_sync_state_file "$source_name")"
}

skillhub_source_has_discoverable_skills() {
  source_path="$1"

  if [ -d "$source_path/skills" ] && find "$source_path/skills" -type f -name SKILL.md -print 2>/dev/null | sed -n '1p' | grep . >/dev/null 2>&1; then
    return 0
  fi

  find "$source_path" -mindepth 2 -maxdepth 2 -type f -name SKILL.md -print 2>/dev/null | sed -n '1p' | grep . >/dev/null 2>&1
}

skillhub_write_discoverable_skill_files() {
  source_path="$1"
  output_file="$2"

  : > "$output_file"
  if [ -d "$source_path/skills" ]; then
    find "$source_path/skills" -type f -name SKILL.md -print 2>/dev/null >> "$output_file"
  fi
  find "$source_path" -mindepth 2 -maxdepth 2 -type f -name SKILL.md -print 2>/dev/null >> "$output_file"
  LC_ALL=C sort -u "$output_file" -o "$output_file"
}

skillhub_catalog_field() {
  printf '%s' "$1" | tr '\t\r\n' '   ' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//; s/[[:space:]][[:space:]]*/ /g'
}

skillhub_skill_frontmatter_value() {
  skill_file="$1"
  wanted_key="$2"
  first_line=$(sed -n '1p' "$skill_file")

  case "$first_line" in
    "--- "*)
      case "$wanted_key" in
        name)
          printf '%s\n' "$first_line" | sed -n 's/^---[[:space:]]*name:[[:space:]]*\([^[:space:]]*\).*/\1/p' | sed -n '1p'
          return
          ;;
        description)
          printf '%s\n' "$first_line" | sed -n 's/.*[[:space:]]description:[[:space:]]*\(.*\)[[:space:]]---.*/\1/p' | sed -n '1p'
          return
          ;;
      esac
      ;;
  esac

  awk -v wanted_key="$wanted_key:" '
    NR == 1 && $0 == "---" { front = 1; next }
    front && $0 == "---" { exit }
    front && index($0, wanted_key) == 1 {
      value = substr($0, length(wanted_key) + 1)
      gsub(/^[[:space:]]+/, "", value)
      gsub(/[[:space:]]+$/, "", value)
      print value
      exit
    }
  ' "$skill_file"
}

skillhub_materialize_source() {
  source_name="$1"
  source_path="$2"
  source_catalog="$3"
  force="${4:-0}"

  if [ -f "$source_path/$source_catalog" ]; then
    rm -rf "$(skillhub_generated_source_path "$source_name")"
    printf '%s\n' "$source_path"
    return
  fi

  if ! skillhub_source_has_discoverable_skills "$source_path"; then
    printf 'Source %s is missing catalog: %s and has no discoverable skills under %s/skills or %s/*/SKILL.md\n' "$source_name" "$source_path/$source_catalog" "$source_path" "$source_path" >&2
    return 1
  fi

  generated_path=$(skillhub_generated_source_path "$source_name")
  if [ "$force" -ne 1 ] && [ -f "$generated_path/$source_catalog" ]; then
    printf '%s\n' "$generated_path"
    return
  fi

  tmp_path="${generated_path}.$$"
  seen_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-generated-seen.XXXXXX")
  skill_files=$(mktemp "${TMPDIR:-/tmp}/skillhub-generated-files.XXXXXX")
  rm -rf "$tmp_path"
  mkdir -p "$tmp_path/catalog" "$tmp_path/skills"

  skillhub_write_discoverable_skill_files "$source_path" "$skill_files"
  while IFS= read -r skill_file; do
    skill_dir=$(dirname -- "$skill_file")
    case "$skill_dir" in
      "$source_path/skills/"*)
        rel=${skill_dir#"$source_path/skills/"}
        ;;
      "$source_path/"*)
        rel=${skill_dir#"$source_path/"}
        ;;
      *)
        rel=$(basename -- "$skill_dir")
        ;;
    esac
    case "$rel" in
      .*|*/.*)
        continue
        ;;
    esac
    flat_name=$(printf '%s' "$rel" | tr '/' '_')
    case "$flat_name" in
      ''|*[!a-z0-9_-]*)
        printf 'Invalid generated skill name from %s: %s\n' "$rel" "$flat_name" >&2
        return 1
        ;;
    esac

    if awk -F '	' -v wanted="$flat_name" '$1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$seen_file"; then
      previous=$(awk -F '	' -v wanted="$flat_name" '$1 == wanted { print $2; exit }' "$seen_file")
      printf 'Duplicate generated skill name %s in source %s: %s and %s\n' "$flat_name" "$source_name" "$previous" "$rel" >&2
      return 1
    fi
    printf '%s\t%s\n' "$flat_name" "$rel" >> "$seen_file"

    category=${rel%%/*}
    triggers=$(printf '%s' "$rel" | tr '/' ',')
    fm_name=$(skillhub_skill_frontmatter_value "$skill_file" name)
    if [ -n "$fm_name" ]; then
      case ",$triggers," in
        *",$fm_name,"*) ;;
        *) triggers="${triggers},$fm_name" ;;
      esac
    fi
    description=$(skillhub_skill_frontmatter_value "$skill_file" description)
    if [ -z "$description" ]; then
      description="Skill from $rel"
    fi

    cp -R "$skill_dir" "$tmp_path/skills/$flat_name"
    printf '%s\t%s\t%s\t%s\n' \
      "$(skillhub_catalog_field "$flat_name")" \
      "$(skillhub_catalog_field "$category")" \
      "$(skillhub_catalog_field "$triggers")" \
      "$(skillhub_catalog_field "$description")" >> "$tmp_path/catalog/skills.tsv.rows"
  done < "$skill_files"

  printf 'name\tcategory\ttriggers\tdescription\n' > "$tmp_path/catalog/skills.tsv"
  if [ -f "$tmp_path/catalog/skills.tsv.rows" ]; then
    cat "$tmp_path/catalog/skills.tsv.rows" >> "$tmp_path/catalog/skills.tsv"
    rm -f "$tmp_path/catalog/skills.tsv.rows"
  fi
  rm -f "$seen_file" "$skill_files"
  rm -rf "$generated_path"
  mv "$tmp_path" "$generated_path"
  printf '%s\n' "$generated_path"
}

skillhub_source_sync_is_fresh() {
  source_name="$1"
  state_file=$(skillhub_source_sync_state_file "$source_name")
  [ -f "$state_file" ] || return 1

  synced_at=$(sed -n '1p' "$state_file" 2>/dev/null || true)
  case "$synced_at" in
    ''|*[!0-9]*)
      return 1
      ;;
  esac

  now=$(skillhub_now_seconds)
  ttl=$(skillhub_source_ttl_seconds)
  if [ "$now" -lt "$synced_at" ]; then
    return 0
  fi

  age=$((now - synced_at))
  [ "$age" -lt "$ttl" ]
}

skillhub_cached_git_origin_matches() {
  source_path="$1"
  source_location="$2"

  [ -d "$source_path/.git" ] || return 0
  current_origin=$(git -C "$source_path" remote get-url origin 2>/dev/null || true)
  [ "$current_origin" = "$source_location" ]
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

skillhub_default_claude_skills_dir() {
  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; use --target directory --dir <path>.\n' >&2
    exit 1
  fi

  printf '%s/.claude/skills\n' "$HOME"
}

skillhub_default_gemini_skills_dir() {
  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; use --target directory --dir <path>.\n' >&2
    exit 1
  fi

  printf '%s/.gemini/skills\n' "$HOME"
}

skillhub_default_opencode_skills_dir() {
  if [ -n "${OPENCODE_CONFIG_DIR:-}" ]; then
    printf '%s/skills\n' "$OPENCODE_CONFIG_DIR"
    return
  fi

  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; set OPENCODE_CONFIG_DIR or use --target directory --dir <path>.\n' >&2
    exit 1
  fi

  printf '%s/.config/opencode/skills\n' "$HOME"
}

skillhub_project_dir() {
  project="$1"
  if [ -n "$project" ]; then
    resolved_project=$(skillhub_abs_path "$project" "$(skillhub_caller_cwd)")
  else
    resolved_project=$(skillhub_abs_path "$(skillhub_caller_cwd)" /)
  fi

  if [ -d "$resolved_project" ]; then
    CDPATH= cd -- "$resolved_project" && pwd
  else
    printf '%s\n' "$resolved_project"
  fi
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
      if ($3 == "planned") {
        seen_planned = 1
      }
      if ($3 == "supported" && seen_planned) {
        printf "Supported target must be listed before planned targets: %s\n", $1 > "/dev/stderr"
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
  if [ -n "$dir" ] && [ "$target" != "directory" ]; then
    printf '%s\n' '--dir can only be used with --target directory.' >&2
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
    claude)
      case "$scope" in
        global)
          skillhub_default_claude_skills_dir
          ;;
        project)
          printf '%s/.claude/skills\n' "$(skillhub_project_dir "$project")"
          ;;
      esac
      ;;
    gemini)
      case "$scope" in
        global)
          skillhub_default_gemini_skills_dir
          ;;
        project)
          printf '%s/.gemini/skills\n' "$(skillhub_project_dir "$project")"
          ;;
      esac
      ;;
    opencode)
      case "$scope" in
        global)
          skillhub_default_opencode_skills_dir
          ;;
        project)
          printf '%s/.opencode/skills\n' "$(skillhub_project_dir "$project")"
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

skillhub_json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

skillhub_now_utc() {
  date -u '+%Y-%m-%dT%H:%M:%SZ'
}

skillhub_validate_installed_registry_file() {
  file=$(skillhub_installed_registry_file)
  if [ ! -f "$file" ]; then
    return 0
  fi

  if ! awk 'NR == 1 { exit ($0 == "source\tskill\ttarget\tscope\tproject_path\ttarget_root\tinstalled_path\tsource_ref\tsource_location\tcatalog\tcontent_hash\tinstalled_at\tupdated_at" ? 0 : 1) }' "$file"; then
    printf 'Installed registry header must be: source<TAB>skill<TAB>target<TAB>scope<TAB>project_path<TAB>target_root<TAB>installed_path<TAB>source_ref<TAB>source_location<TAB>catalog<TAB>content_hash<TAB>installed_at<TAB>updated_at in %s\n' "$file" >&2
    exit 1
  fi
}

skillhub_ensure_installed_registry_file() {
  file=$(skillhub_installed_registry_file)
  mkdir -p "$(dirname -- "$file")"
  if [ ! -f "$file" ]; then
    skillhub_installed_header > "$file"
  fi
  skillhub_validate_installed_registry_file
}

skillhub_usage_filter_matches() {
  filter="$1"
  source_name="$2"
  skill_name="$3"

  if [ -z "$filter" ]; then
    return 0
  fi

  case "$filter" in
    */*)
      [ "$filter" = "$source_name/$skill_name" ]
      ;;
    *)
      [ "$filter" = "$skill_name" ]
      ;;
  esac
}

skillhub_emit_installed_usage_tsv() {
  filter="${1:-}"
  file=$(skillhub_installed_registry_file)

  skillhub_installed_header
  if [ ! -f "$file" ]; then
    return
  fi
  skillhub_validate_installed_registry_file

  awk -F '	' -v filter="$filter" '
    NR == 1 { next }
    $0 == "" || $0 ~ /^#/ { next }
    {
      if (NF != 13) {
        printf "Installed registry row has wrong column count for %s/%s\n", $1, $2 > "/dev/stderr"
        exit 1
      }
      if (filter == "" || $2 == filter || ($1 "/" $2) == filter) {
        print
      }
    }
  ' "$file"
}

skillhub_upsert_installed_usage() {
  source_name="$1"
  skill_name="$2"
  target="$3"
  scope="$4"
  project_path="$5"
  target_root="$6"
  installed_path="$7"
  source_ref="$8"
  source_location="$9"
  source_catalog="${10}"
  content_hash="${11}"
  installed_at="${12}"
  updated_at="${13}"

  skillhub_ensure_installed_registry_file
  file=$(skillhub_installed_registry_file)
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-installed.XXXXXX")

  {
    skillhub_installed_header
    awk -F '	' \
      -v source_name="$source_name" \
      -v skill_name="$skill_name" \
      -v target="$target" \
      -v scope="$scope" \
      -v project_path="$project_path" \
      -v installed_path="$installed_path" '
      NR == 1 { next }
      $0 == "" || $0 ~ /^#/ { next }
      $7 != installed_path {
        print
      }
    ' "$file"
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
      "$source_name" \
      "$skill_name" \
      "$target" \
      "$scope" \
      "$project_path" \
      "$target_root" \
      "$installed_path" \
      "$source_ref" \
      "$source_location" \
      "$source_catalog" \
      "$content_hash" \
      "$installed_at" \
      "$updated_at"
  } > "$tmp_file"

  mv "$tmp_file" "$file"
}

skillhub_remove_installed_usage_by_path() {
  installed_path="$1"
  file=$(skillhub_installed_registry_file)
  if [ ! -f "$file" ]; then
    return 0
  fi
  skillhub_validate_installed_registry_file
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-installed.XXXXXX")

  {
    skillhub_installed_header
    awk -F '	' -v installed_path="$installed_path" '
      NR == 1 { next }
      $0 == "" || $0 ~ /^#/ { next }
      $7 != installed_path { print }
    ' "$file"
  } > "$tmp_file"

  mv "$tmp_file" "$file"
}

skillhub_installed_usage_row_by_path() {
  installed_path="$1"
  file=$(skillhub_installed_registry_file)
  if [ ! -f "$file" ]; then
    return 1
  fi
  skillhub_validate_installed_registry_file

  awk -F '	' -v installed_path="$installed_path" '
    NR == 1 { next }
    $0 == "" || $0 ~ /^#/ { next }
    $7 == installed_path {
      print
      found = 1
      exit
    }
    END { exit(found ? 0 : 1) }
  ' "$file"
}

skillhub_install_uses_sidecar_metadata() {
  target="$1"
  scope="$2"

  case "$scope" in
    project)
      return 1
      ;;
    global|custom)
      return 0
      ;;
    *)
      printf 'Unsupported metadata scope for %s: %s\n' "$target" "$scope" >&2
      exit 1
      ;;
  esac
}

skillhub_project_gitignore_block() {
  cat <<'EOF'
# >>> skillhub local metadata >>>
skills/**/.skillhub.json
skills/.skillhub-*/
.*/skills/**/.skillhub.json
.*/skills/.skillhub-*/
# <<< skillhub local metadata <<<
EOF
}

skillhub_ensure_project_gitignore_metadata() {
  project_path="$1"

  if [ -z "$project_path" ] || [ "$project_path" = "-" ]; then
    return 0
  fi
  if ! command -v git >/dev/null 2>&1; then
    printf 'Warning: git not found; could not update project .gitignore for Skillhub metadata.\n' >&2
    return 0
  fi

  git_root=$(git -C "$project_path" rev-parse --show-toplevel 2>/dev/null || true)
  if [ -z "$git_root" ]; then
    printf 'Warning: project is not a git worktree; could not update .gitignore for Skillhub metadata: %s\n' "$project_path" >&2
    return 0
  fi

  gitignore="$git_root/.gitignore"
  tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-gitignore.XXXXXX")
  clean_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-gitignore.XXXXXX")

  if [ -f "$gitignore" ]; then
    awk '
      $0 == "# >>> skillhub local metadata >>>" { skip = 1; next }
      $0 == "# <<< skillhub local metadata <<<" { skip = 0; next }
      skip != 1 { print }
    ' "$gitignore" > "$tmp_file"
  else
    : > "$tmp_file"
  fi

  awk '
    { lines[NR] = $0 }
    END {
      n = NR
      while (n > 0 && lines[n] == "") {
        n--
      }
      for (i = 1; i <= n; i++) {
        print lines[i]
      }
    }
  ' "$tmp_file" > "$clean_file"

  {
    if [ -s "$clean_file" ]; then
      cat "$clean_file"
      printf '\n\n'
    fi
    skillhub_project_gitignore_block
  } > "$gitignore"

  rm -f "$tmp_file" "$clean_file"
}

skillhub_sha256_stream() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{ print $1 }'
    return
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{ print $1 }'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 | awk '{ print $NF }'
    return
  fi
  cat >/dev/null
  printf '%s\n' '-'
}

skillhub_hash_skill_dir() {
  skill_dir="$1"
  if [ ! -d "$skill_dir" ]; then
    printf '%s\n' '-'
    return
  fi

  (
    CDPATH= cd -- "$skill_dir"
    find . -type f ! -name '.skillhub.json' -print | LC_ALL=C sort | while IFS= read -r file; do
      printf '%s\n' "$file"
      cat -- "$file"
      printf '\n'
    done
  ) | skillhub_sha256_stream
}

skillhub_write_install_metadata() {
  installed_path="$1"
  source_name="$2"
  skill_name="$3"
  qualified_skill="$4"
  target="$5"
  scope="$6"
  target_root="$7"
  source_ref="$8"
  source_location="$9"
  source_catalog="${10}"
  content_hash="${11}"
  project_path="${12:--}"
  previous_installed_at="${13:-}"
  write_sidecar="${14:-1}"
  now=$(skillhub_now_utc)
  installed_at="$previous_installed_at"
  if [ -z "$installed_at" ] || [ "$installed_at" = "-" ]; then
    installed_at="$now"
  fi
  updated_at="$now"

  if [ "$write_sidecar" = "1" ]; then
    metadata_file="$installed_path/.skillhub.json"
    cat > "$metadata_file" <<EOF
{
  "schema_version": "2",
  "source": "$(skillhub_json_escape "$source_name")",
  "skill": "$(skillhub_json_escape "$skill_name")",
  "qualified_skill": "$(skillhub_json_escape "$qualified_skill")",
  "target": "$(skillhub_json_escape "$target")",
  "scope": "$(skillhub_json_escape "$scope")",
  "project_path": "$(skillhub_json_escape "$project_path")",
  "target_root": "$(skillhub_json_escape "$target_root")",
  "installed_path": "$(skillhub_json_escape "$installed_path")",
  "source_ref": "$(skillhub_json_escape "$source_ref")",
  "source_location": "$(skillhub_json_escape "$source_location")",
  "catalog": "$(skillhub_json_escape "$source_catalog")",
  "content_hash": "$(skillhub_json_escape "$content_hash")",
  "installed_at": "$(skillhub_json_escape "$installed_at")",
  "updated_at": "$(skillhub_json_escape "$updated_at")"
}
EOF
    skillhub_upsert_installed_usage \
      "$source_name" \
      "$skill_name" \
      "$target" \
      "$scope" \
      "$project_path" \
      "$target_root" \
      "$installed_path" \
      "$source_ref" \
      "$source_location" \
      "$source_catalog" \
      "$content_hash" \
      "$installed_at" \
      "$updated_at"
    return
  fi

  skillhub_upsert_installed_usage \
    "$source_name" \
    "$skill_name" \
    "$target" \
    "$scope" \
    "$project_path" \
    "$target_root" \
    "$installed_path" \
    "$source_ref" \
    "$source_location" \
    "$source_catalog" \
    "$content_hash" \
    "$installed_at" \
    "$updated_at"
  rm -f "$installed_path/.skillhub.json"
}

skillhub_count_skill_dirs() {
  target_root="$1"
  count=0
  if [ ! -d "$target_root" ]; then
    printf '0\n'
    return
  fi
  for skill_dir in "$target_root"/*; do
    [ -d "$skill_dir" ] || continue
    [ -f "$skill_dir/SKILL.md" ] || continue
    count=$((count + 1))
  done
  printf '%s\n' "$count"
}

skillhub_count_managed_skill_dirs() {
  target_root="$1"
  scope="${2:-}"
  count=0
  if [ ! -d "$target_root" ]; then
    printf '0\n'
    return
  fi
  for skill_dir in "$target_root"/*; do
    [ -d "$skill_dir" ] || continue
    [ -f "$skill_dir/SKILL.md" ] || continue
    if [ "$scope" = "project" ]; then
      if skillhub_installed_usage_row_by_path "$skill_dir" >/dev/null 2>&1; then
        count=$((count + 1))
      fi
      continue
    fi
    if [ -f "$skill_dir/.skillhub.json" ]; then
      count=$((count + 1))
      continue
    fi
  done
  printf '%s\n' "$count"
}

skillhub_dir_exists_label() {
  if [ -d "$1" ]; then
    printf 'yes\n'
  else
    printf 'no\n'
  fi
}

skillhub_metadata_value() {
  metadata_file="$1"
  key="$2"
  if [ ! -f "$metadata_file" ]; then
    printf '%s\n' '-'
    return
  fi
  value=$(sed -n 's/^[[:space:]]*"'"$key"'":[[:space:]]*"\(.*\)",\{0,1\}[[:space:]]*$/\1/p' "$metadata_file" | sed -n '1p')
  if [ -n "$value" ]; then
    printf '%s\n' "$value"
  else
    printf '%s\n' '-'
  fi
}

skillhub_emit_installed_tsv() {
  target_root="$1"
  target="$2"
  scope="$3"

  printf 'target\tscope\tskill\tmanaged\tsource\tqualified_skill\tinstalled_path\tcontent_hash\tinstalled_at\tpath\n'
  if [ ! -d "$target_root" ]; then
    return
  fi

  for skill_dir in "$target_root"/*; do
    [ -d "$skill_dir" ] || continue
    [ -f "$skill_dir/SKILL.md" ] || continue
    skill_name=$(basename -- "$skill_dir")
    metadata_file="$skill_dir/.skillhub.json"
    managed="no"
    registry_row=""
    if [ "$scope" = "project" ]; then
      registry_row=$(skillhub_installed_usage_row_by_path "$skill_dir" 2>/dev/null || true)
    fi

    if [ -n "$registry_row" ]; then
      managed="yes"
      source=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $1 }')
      registry_skill=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $2 }')
      qualified_skill="$source/$registry_skill"
      installed_path=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $7 }')
      content_hash=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $11 }')
      installed_at=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $12 }')
    elif [ "$scope" = "project" ]; then
      source="-"
      qualified_skill="-"
      installed_path="$skill_dir"
      content_hash="-"
      installed_at="-"
    else
      if [ -f "$metadata_file" ]; then
        managed="yes"
      fi
      source=$(skillhub_metadata_value "$metadata_file" source)
      qualified_skill=$(skillhub_metadata_value "$metadata_file" qualified_skill)
      installed_path=$(skillhub_metadata_value "$metadata_file" installed_path)
      content_hash=$(skillhub_metadata_value "$metadata_file" content_hash)
      installed_at=$(skillhub_metadata_value "$metadata_file" installed_at)
      if [ "$installed_path" = "-" ]; then
        installed_path="$skill_dir"
      fi
    fi
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$target" "$scope" "$skill_name" "$managed" "$source" "$qualified_skill" "$installed_path" "$content_hash" "$installed_at" "$skill_dir"
  done
}

skillhub_emit_installed_table() {
  target_root="$1"
  target="$2"
  scope="$3"

  printf '%-12s %-8s %-28s %-8s %-20s %s\n' "target" "scope" "skill" "managed" "source" "path"
  printf '%-12s %-8s %-28s %-8s %-20s %s\n' "------------" "--------" "----------------------------" "--------" "--------------------" "----"
  skillhub_emit_installed_tsv "$target_root" "$target" "$scope" | awk -F '	' 'NR > 1 { printf "%-12s %-8s %-28s %-8s %-20s %s\n", $1, $2, $3, $4, $5, $10 }'
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
        current_origin=$(git -C "$source_path" remote get-url origin 2>/dev/null || true)
        if [ "$current_origin" != "$source_location" ]; then
          cache_sources_dir="$(skillhub_cache_root)/sources"
          case "$source_path" in
            "$cache_sources_dir"/*)
              rm -rf "$source_path"
              ;;
            *)
              printf 'Refusing to replace non-cache source path for %s: %s\n' "$source_name" "$source_path" >&2
              exit 1
              ;;
          esac
        fi
      fi
      if [ -d "$source_path/.git" ]; then
        git -C "$source_path" fetch --quiet --prune >&2
      else
        git clone --quiet "$source_location" "$source_path" >&2
      fi
      git -C "$source_path" checkout --quiet "$source_ref" >&2
      git -C "$source_path" pull --quiet --ff-only >&2
      skillhub_mark_source_synced "$source_name"
      ;;
  esac

  printf '%s\n' "$source_path"
}

skillhub_catalog_source() {
  source_name="$1"
  source_type="$2"
  source_location="$3"
  source_ref="$4"
  source_catalog="$5"

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
      skillhub_materialize_source "$source_name" "$source_path" "$source_catalog" 1
      return
      ;;
    git)
      materialized_path=$(skillhub_materialize_source "$source_name" "$source_path" "$source_catalog" 0 2>/dev/null || true)
      if [ -d "$source_path" ] && [ -n "$materialized_path" ] && [ -f "$materialized_path/$source_catalog" ] && skillhub_source_sync_is_fresh "$source_name" && skillhub_cached_git_origin_matches "$source_path" "$source_location"; then
        printf '%s\n' "$materialized_path"
        return
      fi

      sync_error=$(mktemp "${TMPDIR:-/tmp}/skillhub-source-sync.XXXXXX")
      if synced_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref" 2>"$sync_error"); then
        rm -f "$sync_error"
        skillhub_materialize_source "$source_name" "$synced_path" "$source_catalog" 1
        return
      fi

      materialized_path=$(skillhub_materialize_source "$source_name" "$source_path" "$source_catalog" 0 2>/dev/null || true)
      if [ -d "$source_path" ] && [ -n "$materialized_path" ] && [ -f "$materialized_path/$source_catalog" ]; then
        printf 'Warning: using stale cache for source %s; refresh failed. Run: skillhub sources sync %s\n' "$source_name" "$source_name" >&2
        rm -f "$sync_error"
        printf '%s\n' "$materialized_path"
        return
      fi

      if [ -s "$sync_error" ]; then
        cat "$sync_error" >&2
      fi
      rm -f "$sync_error"
      printf 'Git source cache is missing for %s. Run: skillhub sources sync %s\n' "$source_name" "$source_name" >&2
      exit 1
      ;;
    *)
      printf 'Unsupported source type for %s: %s\n' "$source_name" "$source_type" >&2
      exit 1
      ;;
  esac
}

skillhub_sync_catalog_source() {
  source_name="$1"
  source_type="$2"
  source_location="$3"
  source_ref="$4"
  source_catalog="$5"

  source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
  skillhub_materialize_source "$source_name" "$source_path" "$source_catalog" 1
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
