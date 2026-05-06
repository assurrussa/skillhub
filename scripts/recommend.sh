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
signal_rows=$(mktemp "${TMPDIR:-/tmp}/skillhub-signals.XXXXXX")
trap 'rm -f "$sources_file" "$catalog_rows" "$recommend_rows" "$signal_rows"' EXIT HUP INT TERM

printf 'source\tname\tcategory\ttriggers\tdescription\n' > "$catalog_rows"
source_count=0
while IFS='	' read -r source_name source_type source_location source_ref source_catalog extra; do
  case "$source_name" in
    ''|'#'*|'name')
      continue
      ;;
  esac
  source_count=$((source_count + 1))

  source_path=$(skillhub_catalog_source "$source_name" "$source_type" "$source_location" "$source_ref" "$source_catalog")
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

normalize_signal_token() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//; s/[[:space:]][[:space:]]*/ /g'
}

add_signal() {
  token=$(normalize_signal_token "$1")
  evidence="$2"
  weight="$3"
  if [ -z "$token" ]; then
    return 0
  fi
  if awk -F '	' -v token="$token" '$1 == token { found = 1 } END { exit(found ? 0 : 1) }' "$signal_rows"; then
    return 0
  fi
  printf '%s\t%s\t%s\n' "$token" "$evidence" "$weight" >> "$signal_rows"
}

touch "$signal_rows"

for signal_file in AGENTS.md CLAUDE.md GEMINI.md README.md README.rst README.txt Taskfile.yml Makefile package.json composer.json pyproject.toml Cargo.toml go.mod go.work; do
  if [ -f "$project_dir/$signal_file" ]; then
    add_signal "workflow" "$signal_file" 12
    add_signal "repo orientation" "$signal_file" 10
  fi
done

if [ -f "$project_dir/AGENTS.md" ] || [ -f "$project_dir/CLAUDE.md" ] || [ -f "$project_dir/GEMINI.md" ] || [ -d "$project_dir/.cursor/rules" ]; then
  add_signal "verification" "local agent rules" 10
  add_signal "scope" "local agent rules" 8
  add_signal "review" "local agent rules" 6
fi

if [ -f "$project_dir/README.md" ] || [ -f "$project_dir/README.rst" ] || [ -f "$project_dir/README.txt" ]; then
  add_signal "documentation" "README" 22
  add_signal "readme" "README" 24
fi
if [ -d "$project_dir/docs" ]; then
  add_signal "docs" "docs/" 24
  add_signal "documentation" "docs/" 18
fi
if find "$project_dir" -maxdepth 4 -type f \( -iname '*architecture*' -o -iname '*runbook*' -o -iname '*audit*' -o -iname '*report*' \) -print 2>/dev/null | sed -n '1p' | grep . >/dev/null 2>&1; then
  add_signal "architecture" "architecture/runbook/audit/report docs" 18
  add_signal "runbook" "architecture/runbook/audit/report docs" 10
  add_signal "audit" "architecture/runbook/audit/report docs" 8
  add_signal "report" "architecture/runbook/audit/report docs" 8
fi

if [ -f "$project_dir/go.mod" ]; then
  add_signal "go" "go.mod" 28
  add_signal "golang" "go.mod" 12
  add_signal "go.mod" "go.mod" 48
fi
if [ -f "$project_dir/go.work" ]; then
  add_signal "go" "go.work" 18
  add_signal "go.work" "go.work" 44
  add_signal "reusable module" "go.work" 32
  add_signal "library" "go.work" 20
fi
if find "$project_dir" -maxdepth 4 -type f -name '*.go' 2>/dev/null | sed -n '1p' | grep . >/dev/null 2>&1; then
  add_signal "go" "Go source files" 18
  add_signal "golang" "Go source files" 8
fi

if find "$project_dir" -maxdepth 5 -type f \( -name '*.proto' -o -iname '*openapi*' -o -iname '*swagger*' \) -print 2>/dev/null | sed -n '1p' | grep . >/dev/null 2>&1; then
  add_signal "documentation" "OpenAPI/protobuf contracts" 28
  add_signal "architecture" "OpenAPI/protobuf contracts" 16
  add_signal "docs" "OpenAPI/protobuf contracts" 12
  add_signal "contract change" "OpenAPI/protobuf contracts" 10
fi

if [ -d "$project_dir/pkg" ]; then
  add_signal "public surface" "pkg/" 26
  add_signal "library" "pkg/" 20
  add_signal "reusable module" "pkg/" 18
fi
if [ -f "$project_dir/go.mod" ] && grep -q '^replace[[:space:]]' "$project_dir/go.mod" 2>/dev/null; then
  add_signal "replace" "go.mod replace" 28
  add_signal "external consumer" "go.mod replace" 12
fi
if [ -f "$project_dir/package.json" ] && grep -E -q '"(main|exports|types)"' "$project_dir/package.json" 2>/dev/null; then
  add_signal "public surface" "package.json exports" 26
  add_signal "library" "package.json exports" 18
  add_signal "reusable module" "package.json exports" 16
fi
if [ -f "$project_dir/composer.json" ] || [ -f "$project_dir/pyproject.toml" ] || [ -f "$project_dir/Cargo.toml" ]; then
  add_signal "library" "package manifest" 18
  add_signal "reusable module" "package manifest" 14
fi

tab=$(printf '\t')
awk -F '	' '
  FILENAME == ARGV[1] {
    signal[$1] = $3 + 0
    evidence[$1] = $2
    next
  }
  FNR == 1 { next }
  $0 == "" || $0 ~ /^#/ { next }
  function trim(value) {
    gsub(/^[[:space:]]+/, "", value)
    gsub(/[[:space:]]+$/, "", value)
    gsub(/[[:space:]][[:space:]]+/, " ", value)
    return value
  }
  function add_match(token,   key) {
    key = tolower(trim(token))
    if (key == "" || !(key in signal)) {
      return
    }
    score += signal[key]
    if (seen[evidence[key]] != row_id) {
      seen[evidence[key]] = row_id
      reasons = reasons (reasons == "" ? "" : ", ") evidence[key]
    }
  }
  {
    row_id++
    source_name = $1
    skill_name = $2
    category = $3
    triggers = $4
    score = 0
    reasons = ""
    add_match(category)
    trigger_count = split(triggers, trigger_list, ",")
    for (i = 1; i <= trigger_count; i++) {
      add_match(trigger_list[i])
    }
    if (score > 0) {
      printf "%06d\t%s\t%s\tmatched %s\t%s/%s\n", score, source_name, skill_name, reasons, source_name, skill_name
    }
  }
' "$signal_rows" "$catalog_rows" | LC_ALL=C sort -t "$tab" -k1,1nr -k2,2 -k3,3 > "$recommend_rows"

if [ "$format" = "tsv" ]; then
  printf 'source\tskill\treason\tinstall_arg\n'
  awk -F '	' '{ printf "%s\t%s\t%s\t%s\n", $2, $3, $4, $5 }' "$recommend_rows"
  exit 0
fi

recommend_count=$(awk 'END { print NR + 0 }' "$recommend_rows")
if [ "$recommend_count" -eq 0 ]; then
  printf 'No shared skills recommended for %s.\n' "$project_dir"
  printf 'Reason: no supported project signals matched the active source catalog.\n'
  exit 0
fi

printf 'Recommended shared skills for %s:\n' "$project_dir"
awk -F '	' '{ printf "- %s/%s (score %d): %s\n", $2, $3, $1 + 0, $4 }' "$recommend_rows"

install_args=$(awk -F '	' '{ printf " %s", $5 }' "$recommend_rows")
printf '\nInstall:\n'
printf 'skillhub install --target codex --scope project --project %s%s\n' "$(shell_quote "$project_dir")" "$install_args"
