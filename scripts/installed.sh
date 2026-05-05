#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/installed.sh list [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--tsv]' \
    '  sh scripts/installed.sh update [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [-v|--verbose]' \
    '  sh scripts/installed.sh uninstall <skill> [--target <target>] [--scope global|project] [--project <path>] [--dir <path>] [--force]'
}

update_sources_file=""
cleanup_update_sources_file() {
  if [ -n "$update_sources_file" ]; then
    rm -f "$update_sources_file"
  fi
}
trap cleanup_update_sources_file EXIT HUP INT TERM

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
  skill_basename=$(basename -- "$skill_dir")
  metadata_file="$skill_dir/.skillhub.json"

  update_result="skipped"

  source_name=$(skillhub_metadata_value "$metadata_file" source)
  skill_name=$(skillhub_metadata_value "$metadata_file" skill)
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

  source_path=$(skillhub_sync_source "$source_name" "$source_type" "$source_location" "$source_ref")
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

  if ! cp -R "$source_skill_dir" "$tmp_dir"; then
    rm -rf "$tmp_dir"
    printf 'Failed %s/%s %s: could not copy source skill\n' "$target" "$scope" "$skill_name"
    update_result="failed"
    return 0
  fi

  new_hash=$(skillhub_hash_skill_dir "$tmp_dir")
  if [ "$old_hash" != "-" ] && [ "$new_hash" != "-" ] && [ "$old_hash" = "$new_hash" ]; then
    rm -rf "$tmp_dir"
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
    "$content_hash"

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
  print_empty="$5"

  managed_count=0
  target_updated=0
  target_unchanged=0
  target_skipped=0
  target_failed=0

  if [ -d "$target_root" ]; then
    for skill_dir in "$target_root"/*; do
      [ -d "$skill_dir" ] || continue
      [ -f "$skill_dir/SKILL.md" ] || continue
      [ -f "$skill_dir/.skillhub.json" ] || continue
      managed_count=$((managed_count + 1))
      update_managed_skill "$skill_dir" "$target_root" "$target" "$scope" "$verbose"
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

  update_supported_root() {
    root_target="$1"
    root_scope="$2"
    root_path="$3"

    if [ "$(skillhub_count_managed_skill_dirs "$root_path")" -eq 0 ]; then
      if [ "$verbose" -eq 1 ]; then
        printf 'No managed skills in %s/%s: %s\n' "$root_target" "$root_scope" "$root_path"
      fi
      return 0
    fi
    touched=1
    update_target_root "$root_path" "$root_target" "$root_scope" "$verbose" 0
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

      if [ "$id" = "codex" ] && [ "$candidate_scope" = "global" ] && [ -n "${AGENT_SKILLS_DIR:-}" ] && [ "$AGENT_SKILLS_DIR" != "$canonical_root" ]; then
        update_supported_root "$id" "$candidate_scope" "$AGENT_SKILLS_DIR"
      fi

      update_supported_root "$id" "$candidate_scope" "$canonical_root"
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

  update_target_root "$target_root" "$target" "$metadata_scope" "$verbose" 1
  printf 'Updated: %s, unchanged: %s, skipped: %s, failed: %s\n' \
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
  update)
    update_installed "$@"
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
