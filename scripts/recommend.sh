#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$repo_root/scripts/lib.sh"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh scripts/recommend.sh [--project <path>] [--tsv]'
}

no_sources_message() {
  printf 'No sources configured. Run: skillhub sources defaults list\n' >&2
}

shell_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

project=""
format="table"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --project)
      if [ "$#" -lt 2 ]; then
        printf '%s\n' '--project requires a value' >&2
        exit 1
      fi
      project="$2"
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

project_dir=$(skillhub_project_dir "$project")
if [ ! -d "$project_dir" ]; then
  printf 'Project directory does not exist: %s\n' "$project_dir" >&2
  exit 1
fi

skillhub_validate_sources_file
sources_file=$(skillhub_active_sources_file)
catalog_rows=$(mktemp "${TMPDIR:-/tmp}/skillhub-catalog.XXXXXX")
recommend_rows=$(mktemp "${TMPDIR:-/tmp}/skillhub-recommend.XXXXXX")
trap 'rm -f "$sources_file" "$catalog_rows" "$recommend_rows"' EXIT HUP INT TERM

printf 'source\tname\tcategory\ttriggers\tdescription\n' > "$catalog_rows"
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
    printf '%s\t%s\t%s\t%s\t%s\n' "$source_name" "$skill_name" "$category" "$triggers" "$description" >> "$catalog_rows"
  done < "$catalog_file"
done < "$sources_file"

if [ "$source_count" -eq 0 ]; then
  no_sources_message
  exit 1
fi

catalog_row_for_skill() {
  wanted="$1"
  awk -F '	' -v wanted="$wanted" 'NR > 1 && $2 == wanted { print; exit }' "$catalog_rows"
}

add_recommendation() {
  skill_name="$1"
  reason="$2"
  row=$(catalog_row_for_skill "$skill_name")
  if [ -z "$row" ]; then
    return 0
  fi
  source_name=$(printf '%s\n' "$row" | awk -F '	' '{ print $1 }')
  install_arg="$source_name/$skill_name"
  if awk -F '	' -v install_arg="$install_arg" '$4 == install_arg { found = 1 } END { exit(found ? 0 : 1) }' "$recommend_rows"; then
    return 0
  fi
  printf '%s\t%s\t%s\t%s\n' "$source_name" "$skill_name" "$reason" "$install_arg" >> "$recommend_rows"
}

has_any_signal=0
has_docs_signal=0
has_go_signal=0
has_reusable_signal=0
has_contract_signal=0

for signal_file in AGENTS.md README.md README.rst README.txt Taskfile.yml Makefile package.json composer.json pyproject.toml Cargo.toml go.mod go.work; do
  if [ -f "$project_dir/$signal_file" ]; then
    has_any_signal=1
  fi
done

if [ -f "$project_dir/AGENTS.md" ] || [ -f "$project_dir/README.md" ] || [ -d "$project_dir/docs" ]; then
  has_docs_signal=1
  has_any_signal=1
fi

if [ -f "$project_dir/go.mod" ] || [ -f "$project_dir/go.work" ]; then
  has_go_signal=1
  has_any_signal=1
else
  first_go=$(find "$project_dir" -maxdepth 4 -type f -name '*.go' 2>/dev/null | sed -n '1p')
  if [ -n "$first_go" ]; then
    has_go_signal=1
    has_any_signal=1
  fi
fi

first_contract=$(find "$project_dir" -maxdepth 5 -type f \( -name '*.proto' -o -iname '*openapi*' -o -iname '*swagger*' \) 2>/dev/null | sed -n '1p')
if [ -n "$first_contract" ]; then
  has_contract_signal=1
  has_any_signal=1
fi

if [ -f "$project_dir/go.work" ] || [ -d "$project_dir/pkg" ]; then
  has_reusable_signal=1
fi
if [ -f "$project_dir/go.mod" ] && grep -q '^replace[[:space:]]' "$project_dir/go.mod" 2>/dev/null; then
  has_reusable_signal=1
fi
if [ -f "$project_dir/package.json" ] && grep -E -q '"(main|exports|types)"' "$project_dir/package.json" 2>/dev/null; then
  has_reusable_signal=1
fi
if [ -f "$project_dir/composer.json" ] || [ -f "$project_dir/pyproject.toml" ] || [ -f "$project_dir/Cargo.toml" ]; then
  has_reusable_signal=1
fi

if [ "$has_any_signal" -eq 1 ]; then
  add_recommendation "project-workflow-rules" "Repository workflow signals detected; use source-of-truth, scoped-change, dirty-worktree, and verification rules."
fi
if [ "$has_go_signal" -eq 1 ]; then
  add_recommendation "go-project-rules" "Go source/module/workspace signals detected."
fi
if [ "$has_reusable_signal" -eq 1 ]; then
  add_recommendation "reusable-module-rules" "Reusable module or public package surface signals detected."
fi
if [ "$has_docs_signal" -eq 1 ] || [ "$has_contract_signal" -eq 1 ]; then
  add_recommendation "docs-project-rules" "Documentation, project overlay, or public contract signals detected."
fi

if [ "$format" = "tsv" ]; then
  printf 'source\tskill\treason\tinstall_arg\n'
  cat "$recommend_rows"
  exit 0
fi

recommend_count=$(awk 'END { print NR + 0 }' "$recommend_rows")
if [ "$recommend_count" -eq 0 ]; then
  printf 'No shared skills recommended for %s.\n' "$project_dir"
  printf 'Reason: no supported project signals matched the active source catalog.\n'
  exit 0
fi

printf 'Recommended shared skills for %s:\n' "$project_dir"
awk -F '	' '{ printf "- %s/%s: %s\n", $1, $2, $3 }' "$recommend_rows"

install_args=$(awk -F '	' '{ printf " %s", $4 }' "$recommend_rows")
printf '\nInstall:\n'
printf 'skillhub install --target codex --scope project --project %s%s\n' "$(shell_quote "$project_dir")" "$install_args"
