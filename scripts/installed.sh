#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/installed.sh list [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--tsv]' \
    '  sh scripts/installed.sh update [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [-v|--verbose]' \
    '  sh scripts/installed.sh uninstall <skill> [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--force]' \
    '  sh scripts/installed.sh usage [[<source>/]<skill>] [--tsv]' \
    '  sh scripts/installed.sh usage update --projects [--target <target>] [--project <path>] [[<source>/]<skill>...] [-v|--verbose]'
}

update_sources_file=""
usage_snapshot_file=""
cleanup_update_temp_files() {
  if [ -n "$update_sources_file" ]; then
    rm -f "$update_sources_file"
  fi
  if [ -n "$usage_snapshot_file" ]; then
    rm -f "$usage_snapshot_file"
  fi
}
trap cleanup_update_temp_files EXIT HUP INT TERM

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

is_valid_installed_skill_name() {
  case "$1" in
    ''|*[!a-z0-9_-]*)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
}

validate_usage_filter() {
  filter="$1"
  case "$filter" in
    */*)
      source_name=${filter%%/*}
      skill_name=${filter#*/}
      if ! skillhub_is_valid_source_name "$source_name" || ! is_valid_installed_skill_name "$skill_name"; then
        printf 'Invalid usage filter: %s\n' "$filter" >&2
        exit 1
      fi
      ;;
    *)
      if ! is_valid_installed_skill_name "$filter"; then
        printf 'Invalid usage filter: %s\n' "$filter" >&2
        exit 1
      fi
      ;;
  esac
}

usage_filter_list_matches() {
  source_name="$1"
  skill_name="$2"
  filters="$3"

  if [ -z "$filters" ]; then
    return 0
  fi

  for filter in $filters; do
    if skillhub_usage_filter_matches "$filter" "$source_name" "$skill_name"; then
      return 0
    fi
  done
  return 1
}

update_sources_path() {
  if [ -z "$update_sources_file" ]; then
    skillhub_validate_sources_file
    update_sources_file=$(skillhub_active_sources_file)
  fi
  printf '%s\n' "$update_sources_file"
}

find_update_source_row() {
  wanted="$1"
  sources_file=$(update_sources_path)

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
      -h|--help|help)
        usage
        exit 0
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

update_result="skipped"
update_managed_skill() {
  skill_dir="$1"
  target_root="$2"
  target="$3"
  scope="$4"
  verbose="$5"
  project_path="${6:-}"
  registry_source="${7:-}"
  registry_skill="${8:-}"
  registry_installed_at="${9:-}"
  skill_basename=$(basename -- "$skill_dir")
  metadata_file="$skill_dir/.skillhub.json"

  update_result="skipped"

  if [ -n "$registry_source" ]; then
    source_name="$registry_source"
  else
    source_name=$(skillhub_metadata_value "$metadata_file" source)
  fi
  if [ -n "$registry_skill" ]; then
    skill_name="$registry_skill"
  else
    skill_name=$(skillhub_metadata_value "$metadata_file" skill)
  fi
  if [ -n "$registry_installed_at" ]; then
    installed_at="$registry_installed_at"
  else
    installed_at=$(skillhub_metadata_value "$metadata_file" installed_at)
  fi
  if [ "$installed_at" = "-" ]; then
    installed_at=""
  fi
  if [ -z "$project_path" ]; then
    project_path=$(skillhub_metadata_value "$metadata_file" project_path)
    if [ "$project_path" = "-" ]; then
      project_path="-"
    fi
  fi
  if [ "$skill_name" = "-" ]; then
    skill_name="$skill_basename"
  fi

  if ! skillhub_is_valid_source_name "$source_name"; then
    printf 'Skipped %s/%s %s: invalid metadata source %s\n' "$target" "$scope" "$skill_basename" "$source_name"
    return 0
  fi
  if ! is_valid_installed_skill_name "$skill_name"; then
    printf 'Skipped %s/%s %s: invalid metadata skill %s\n' "$target" "$scope" "$skill_basename" "$skill_name"
    return 0
  fi

  source_row=$(find_update_source_row "$source_name" || true)
  if [ -z "$source_row" ]; then
    printf 'Skipped %s/%s %s: source not configured: %s\n' "$target" "$scope" "$skill_name" "$source_name"
    return 0
  fi

  source_type=$(printf '%s\n' "$source_row" | awk -F '	' '{ print $2 }')
  source_location=$(printf '%s\n' "$source_row" | awk -F '	' '{ print $3 }')
  source_ref=$(printf '%s\n' "$source_row" | awk -F '	' '{ print $4 }')
  source_catalog=$(printf '%s\n' "$source_row" | awk -F '	' '{ print $5 }')

  if [ "$verbose" -eq 1 ]; then
    printf 'Checking %s/%s %s from %s\n' "$target" "$scope" "$skill_name" "$source_name"
  fi

  source_path=$(skillhub_sync_catalog_source "$source_name" "$source_type" "$source_location" "$source_ref" "$source_catalog")
  catalog_file="$source_path/$source_catalog"
  if [ ! -f "$catalog_file" ]; then
    printf 'Skipped %s/%s %s: source catalog missing: %s\n' "$target" "$scope" "$skill_name" "$catalog_file"
    return 0
  fi
  if ! awk -F '	' -v wanted="$skill_name" 'NR > 1 && $1 == wanted { found = 1 } END { exit(found ? 0 : 1) }' "$catalog_file"; then
    printf 'Skipped %s/%s %s: skill no longer cataloged in source %s\n' "$target" "$scope" "$skill_name" "$source_name"
    return 0
  fi

  source_skill_dir="$source_path/skills/$skill_name"
  if [ ! -f "$source_skill_dir/SKILL.md" ]; then
    printf 'Skipped %s/%s %s: source SKILL.md missing: %s\n' "$target" "$scope" "$skill_name" "$source_skill_dir/SKILL.md"
    return 0
  fi

  old_hash=$(skillhub_hash_skill_dir "$skill_dir")
  tmp_dir="$target_root/.skillhub-update-$skill_name.$$"
  backup_dir="$target_root/.skillhub-backup-$skill_name.$$"
  rm -rf "$tmp_dir" "$backup_dir"
  write_sidecar=1
  if ! skillhub_install_uses_sidecar_metadata "$target" "$scope"; then
    write_sidecar=0
  fi

  if ! cp -R "$source_skill_dir" "$tmp_dir"; then
    rm -rf "$tmp_dir"
    printf 'Failed %s/%s %s: could not copy source skill\n' "$target" "$scope" "$skill_name"
    update_result="failed"
    return 0
  fi

  new_hash=$(skillhub_hash_skill_dir "$tmp_dir")
  if [ "$old_hash" != "-" ] && [ "$new_hash" != "-" ] && [ "$old_hash" = "$new_hash" ]; then
    rm -rf "$tmp_dir"
    skillhub_write_install_metadata \
      "$skill_dir" \
      "$source_name" \
      "$skill_name" \
      "$source_name/$skill_name" \
      "$target" \
      "$scope" \
      "$target_root" \
      "$source_ref" \
      "$source_location" \
      "$source_catalog" \
      "$old_hash" \
      "$project_path" \
      "$installed_at" \
      "$write_sidecar"
    if [ "$scope" = "project" ]; then
      skillhub_ensure_project_gitignore_metadata "$project_path"
    fi
    if [ "$verbose" -eq 1 ]; then
      printf 'Unchanged %s/%s %s (%s)\n' "$target" "$scope" "$skill_name" "$old_hash"
    fi
    update_result="unchanged"
    return 0
  fi

  if ! mv "$skill_dir" "$backup_dir"; then
    rm -rf "$tmp_dir" "$backup_dir"
    printf 'Failed %s/%s %s: could not move current install aside\n' "$target" "$scope" "$skill_name"
    update_result="failed"
    return 0
  fi
  if ! mv "$tmp_dir" "$skill_dir"; then
    mv "$backup_dir" "$skill_dir" 2>/dev/null || true
    rm -rf "$tmp_dir" "$backup_dir"
    printf 'Failed %s/%s %s: could not replace installed skill\n' "$target" "$scope" "$skill_name"
    update_result="failed"
    return 0
  fi
  rm -rf "$backup_dir"

  content_hash=$(skillhub_hash_skill_dir "$skill_dir")
  skillhub_write_install_metadata \
    "$skill_dir" \
    "$source_name" \
    "$skill_name" \
    "$source_name/$skill_name" \
    "$target" \
    "$scope" \
    "$target_root" \
    "$source_ref" \
    "$source_location" \
    "$source_catalog" \
    "$content_hash" \
    "$project_path" \
    "$installed_at" \
    "$write_sidecar"
  if [ "$scope" = "project" ]; then
    skillhub_ensure_project_gitignore_metadata "$project_path"
  fi

  if [ "$verbose" -eq 1 ]; then
    printf 'Updated %s/%s %s %s -> %s\n' "$target" "$scope" "$skill_name" "$old_hash" "$content_hash"
  else
    printf 'Updated %s/%s %s\n' "$target" "$scope" "$skill_name"
  fi
  update_result="updated"
}

updated_total=0
unchanged_total=0
skipped_total=0
failed_total=0

update_target_root() {
  target_root="$1"
  target="$2"
  scope="$3"
  verbose="$4"
  project_path="$5"
  print_empty="$6"

  managed_count=0
  target_updated=0
  target_unchanged=0
  target_skipped=0
  target_failed=0

  if [ -d "$target_root" ]; then
    for skill_dir in "$target_root"/*; do
      [ -d "$skill_dir" ] || continue
      [ -f "$skill_dir/SKILL.md" ] || continue
      registry_row=""
      if [ "$scope" = "project" ]; then
        registry_row=$(skillhub_installed_usage_row_by_path "$skill_dir" 2>/dev/null || true)
        [ -n "$registry_row" ] || continue
      elif [ ! -f "$skill_dir/.skillhub.json" ]; then
        continue
      fi
      managed_count=$((managed_count + 1))
      if [ -n "$registry_row" ]; then
        registry_source=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $1 }')
        registry_skill=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $2 }')
        registry_installed_at=$(printf '%s\n' "$registry_row" | awk -F '	' '{ print $12 }')
        update_managed_skill "$skill_dir" "$target_root" "$target" "$scope" "$verbose" "$project_path" "$registry_source" "$registry_skill" "$registry_installed_at"
      else
        update_managed_skill "$skill_dir" "$target_root" "$target" "$scope" "$verbose" "$project_path"
      fi
      case "$update_result" in
        updated)
          target_updated=$((target_updated + 1))
          updated_total=$((updated_total + 1))
          ;;
        unchanged)
          target_unchanged=$((target_unchanged + 1))
          unchanged_total=$((unchanged_total + 1))
          ;;
        failed)
          target_failed=$((target_failed + 1))
          failed_total=$((failed_total + 1))
          ;;
        *)
          target_skipped=$((target_skipped + 1))
          skipped_total=$((skipped_total + 1))
          ;;
      esac
    done
  fi

  if [ "$managed_count" -eq 0 ]; then
    if [ "$print_empty" -eq 1 ]; then
      printf 'No managed installed skills found in %s\n' "$target_root"
    elif [ "$verbose" -eq 1 ]; then
      printf 'No managed skills in %s/%s: %s\n' "$target" "$scope" "$target_root"
    fi
    return 0
  fi

  printf 'Summary %s/%s: updated=%s unchanged=%s skipped=%s failed=%s\n' \
    "$target" "$scope" "$target_updated" "$target_unchanged" "$target_skipped" "$target_failed"
}

update_all_supported_targets() {
  project="$1"
  verbose="$2"
  touched=0
  resolved_project=$(skillhub_project_dir "$project")

  update_supported_root() {
    root_target="$1"
    root_scope="$2"
    root_path="$3"
    root_project_path="$4"

    if [ "$(skillhub_count_managed_skill_dirs "$root_path" "$root_scope")" -eq 0 ]; then
      if [ "$verbose" -eq 1 ]; then
        printf 'No managed skills in %s/%s: %s\n' "$root_target" "$root_scope" "$root_path"
      fi
      return 0
    fi
    touched=1
    update_target_root "$root_path" "$root_target" "$root_scope" "$verbose" "$root_project_path" 0
  }

  while IFS='	' read -r id label status adapter description extra; do
    case "$id" in
      ''|'#'*|'id'|directory)
        continue
        ;;
    esac
    [ "$status" = "supported" ] || continue
    [ "$adapter" = "skill-dir" ] || continue

    for candidate_scope in global project; do
      canonical_root=$(skillhub_target_root "$id" "$candidate_scope" "$project" "" 0 1)
      candidate_project_path="-"
      if [ "$candidate_scope" = "project" ]; then
        candidate_project_path="$resolved_project"
      fi

      if [ "$id" = "codex" ] && [ "$candidate_scope" = "global" ] && [ -n "${AGENT_SKILLS_DIR:-}" ] && [ "$AGENT_SKILLS_DIR" != "$canonical_root" ]; then
        update_supported_root "$id" "$candidate_scope" "$AGENT_SKILLS_DIR" "-"
      fi

      update_supported_root "$id" "$candidate_scope" "$canonical_root" "$candidate_project_path"
    done
  done < "$(skillhub_targets_file)"

  if [ "$touched" -eq 0 ]; then
    printf 'No managed installed skills found across supported targets.\n'
  fi

  printf 'Updated: %s, unchanged: %s, skipped: %s, failed: %s\n' \
    "$updated_total" "$unchanged_total" "$skipped_total" "$failed_total"

  if [ "$failed_total" -gt 0 ]; then
    exit 1
  fi
}

update_installed() {
  shift

  target="codex"
  scope="global"
  project=""
  dir=""
  legacy_allowed=1
  scope_was_set=0
  verbose=0
  all_supported=0

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
      --all-supported)
        all_supported=1
        legacy_allowed=0
        shift
        ;;
      -v|--verbose)
        verbose=1
        shift
        ;;
      -h|--help|help)
        usage
        exit 0
        ;;
      *)
        usage >&2
        exit 1
        ;;
    esac
  done

  if [ "$all_supported" -eq 1 ]; then
    if [ -n "$dir" ] || [ "$target" != "codex" ] || [ "$scope_was_set" -eq 1 ]; then
      printf '%s\n' '--all-supported cannot be combined with --target, --scope, or --dir' >&2
      exit 1
    fi
    update_all_supported_targets "$project" "$verbose"
    return
  fi

  target_root=$(skillhub_target_root "$target" "$scope" "$project" "$dir" "$legacy_allowed" "$scope_was_set")
  metadata_scope="$scope"
  if [ "$target" = "directory" ]; then
    metadata_scope="custom"
  fi
  project_path="-"
  if [ "$metadata_scope" = "project" ]; then
    project_path=$(skillhub_project_dir "$project")
  fi

  update_target_root "$target_root" "$target" "$metadata_scope" "$verbose" "$project_path" 1
  printf 'Updated: %s, unchanged: %s, skipped: %s, failed: %s\n' \
    "$updated_total" "$unchanged_total" "$skipped_total" "$failed_total"

  if [ "$failed_total" -gt 0 ]; then
    exit 1
  fi
}

usage_lookup() {
  shift

  if [ "${1:-}" = "update" ]; then
    shift
    usage_update_projects "$@"
    return
  fi

  filter=""
  format="table"

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --tsv)
        format="tsv"
        shift
        ;;
      -h|--help|help)
        usage
        exit 0
        ;;
      -*)
        usage >&2
        exit 1
        ;;
      *)
        if [ -n "$filter" ]; then
          usage >&2
          exit 1
        fi
        validate_usage_filter "$1"
        filter="$1"
        shift
        ;;
    esac
  done

  tmp_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-usage.XXXXXX")
  skillhub_emit_installed_usage_tsv "$filter" > "$tmp_file"

  if [ "$format" = "tsv" ]; then
    cat "$tmp_file"
    rm -f "$tmp_file"
    return
  fi

  printf '%-20s %-28s %-10s %-8s %-32s %s\n' "source" "skill" "target" "scope" "project" "installed_path"
  printf '%-20s %-28s %-10s %-8s %-32s %s\n' "--------------------" "----------------------------" "----------" "--------" "--------------------------------" "--------------"
  awk -F '	' 'NR > 1 {
    printf "%-20s %-28s %-10s %-8s %-32s %s\n", $1, $2, $3, $4, $5, $7
    count++
  }
  END {
    if (count == 0) {
      print "No managed skill usage recorded."
    }
  }' "$tmp_file"
  rm -f "$tmp_file"
}

usage_update_projects() {
  projects=0
  verbose=0
  filters=""
  target_filter=""
  project_filter=""

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --projects)
        projects=1
        shift
        ;;
      --target)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--target requires a value' >&2
          exit 1
        fi
        if ! skillhub_is_valid_target_id "$2"; then
          printf 'Invalid target id: %s\n' "$2" >&2
          exit 1
        fi
        target_filter="$2"
        shift 2
        ;;
      --project)
        if [ "$#" -lt 2 ]; then
          printf '%s\n' '--project requires a value' >&2
          exit 1
        fi
        project_filter=$(skillhub_project_dir "$2")
        shift 2
        ;;
      -v|--verbose)
        verbose=1
        shift
        ;;
      -h|--help|help)
        usage
        exit 0
        ;;
      -*)
        usage >&2
        exit 1
        ;;
      *)
        validate_usage_filter "$1"
        filters="${filters}${filters:+ }$1"
        shift
        ;;
    esac
  done

  if [ "$projects" -ne 1 ]; then
    printf '%s\n' 'usage update currently requires --projects.' >&2
    exit 1
  fi

  registry_file=$(skillhub_installed_registry_file)
  if [ ! -f "$registry_file" ]; then
    printf 'No managed project-scope skill usage recorded.\n'
    return
  fi
  skillhub_validate_installed_registry_file

  usage_snapshot_file=$(mktemp "${TMPDIR:-/tmp}/skillhub-project-usage.XXXXXX")
  {
    skillhub_installed_header
    while IFS='	' read -r source_name skill_name target scope project_path target_root installed_path source_ref source_location source_catalog content_hash installed_at updated_at extra; do
      case "$source_name" in
        ''|'#'*|'source')
          continue
          ;;
      esac

      if [ "$scope" != "project" ]; then
        continue
      fi
      if [ -n "$target_filter" ] && [ "$target" != "$target_filter" ]; then
        continue
      fi
      if [ -n "$project_filter" ] && [ "$project_path" != "$project_filter" ]; then
        continue
      fi
      if ! usage_filter_list_matches "$source_name" "$skill_name" "$filters"; then
        continue
      fi

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
    done < "$registry_file"
  } > "$usage_snapshot_file"

  project_usage_count=$(awk 'NR > 1 && $0 != "" { count++ } END { print count + 0 }' "$usage_snapshot_file")
  while IFS='	' read -r source_name skill_name target scope project_path target_root installed_path source_ref source_location source_catalog content_hash installed_at updated_at extra; do
    case "$source_name" in
      ''|'#'*|'source')
        continue
        ;;
    esac
    if [ ! -d "$installed_path" ] || [ ! -f "$installed_path/SKILL.md" ]; then
      printf 'Skipped %s/project %s: managed install missing at %s\n' "$target" "$skill_name" "$installed_path"
      skipped_total=$((skipped_total + 1))
      continue
    fi

    update_managed_skill "$installed_path" "$target_root" "$target" "$scope" "$verbose" "$project_path" "$source_name" "$skill_name" "$installed_at"
    case "$update_result" in
      updated)
        updated_total=$((updated_total + 1))
        ;;
      unchanged)
        unchanged_total=$((unchanged_total + 1))
        ;;
      failed)
        failed_total=$((failed_total + 1))
        ;;
      *)
        skipped_total=$((skipped_total + 1))
        ;;
    esac
  done < "$usage_snapshot_file"
  rm -f "$usage_snapshot_file"
  usage_snapshot_file=""

  if [ "$project_usage_count" -eq 0 ]; then
    printf 'No managed project-scope skill usage matched registry filters.\n'
  fi

  printf 'Updated project usage: updated=%s unchanged=%s skipped=%s failed=%s\n' \
    "$updated_total" "$unchanged_total" "$skipped_total" "$failed_total"

  if [ "$failed_total" -gt 0 ]; then
    exit 1
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
      -h|--help|help)
        usage
        exit 0
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

  registry_row=$(skillhub_installed_usage_row_by_path "$skill_dir" 2>/dev/null || true)
  if [ "$scope" = "project" ]; then
    managed_by_sidecar=0
  elif [ -f "$skill_dir/.skillhub.json" ]; then
    managed_by_sidecar=1
  else
    managed_by_sidecar=0
  fi
  if [ "$managed_by_sidecar" -ne 1 ] && [ -z "$registry_row" ] && [ "$force" -ne 1 ]; then
    printf 'Refusing to uninstall unmanaged skill: %s\n' "$skill_dir" >&2
    printf 'Use --force to remove it anyway.\n' >&2
    exit 1
  fi

  rm -rf "$skill_dir"
  skillhub_remove_installed_usage_by_path "$skill_dir"
  printf 'Uninstalled %s from %s\n' "$skill_name" "$target_root"
}

cmd="${1:-list}"
case "$cmd" in
  list)
    list_installed "$@"
    ;;
  update)
    update_installed "$@"
    ;;
  uninstall)
    uninstall_installed "$@"
    ;;
  usage)
    usage_lookup "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
