#!/bin/sh
set -eu

default_repo_url="https://github.com/assurrussa/skillhub.git"
default_ref="main"

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh install.sh [--bin-dir <dir>] [--update]' \
    '' \
    'Environment:' \
    '  SKILLHUB_BIN_DIR   Install directory for the skillhub command.' \
    '                     Defaults to $HOME/.local/bin.' \
    '  SKILLHUB_HOME      Checkout/cache directory for curl installs.' \
    '                     Defaults to $HOME/.local/share/skillhub.' \
    '  SKILLHUB_REPO_URL  Git repository URL for curl installs.' \
    '                     Defaults to https://github.com/assurrussa/skillhub.git.' \
    '  SKILLHUB_REF       Git branch/tag/ref for curl installs.' \
    '                     Defaults to main.'
}

bin_dir="${SKILLHUB_BIN_DIR:-}"
update=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --bin-dir)
      if [ "$#" -lt 2 ]; then
        printf '%s\n' '--bin-dir requires a value' >&2
        exit 1
      fi
      bin_dir="$2"
      shift 2
      ;;
    --update)
      update=1
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

require_home() {
  if [ -z "${HOME:-}" ]; then
    printf '%s\n' 'HOME is not set; pass --bin-dir and set SKILLHUB_HOME explicitly.' >&2
    exit 1
  fi
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '%s is required.\n' "$1" >&2
    exit 1
  fi
}

is_checkout() {
  dir="$1"
  [ -f "$dir/go.mod" ] &&
    [ -f "$dir/cmd/skillhub/main.go" ] &&
    [ -f "$dir/defaults/sources.tsv" ] &&
    [ -f "$dir/targets/targets.tsv" ]
}

script_dir() {
  CDPATH= cd -- "$(dirname -- "$0")" 2>/dev/null && pwd
}

find_local_checkout() {
  dir=$(script_dir || printf '')
  if [ -n "$dir" ] && is_checkout "$dir"; then
    printf '%s\n' "$dir"
    return 0
  fi
  return 1
}

sync_repo() {
  repo_root="$1"
  ref="$2"

  require_cmd git

  if [ ! -d "$repo_root/.git" ]; then
    printf 'Cannot update non-git checkout: %s\n' "$repo_root" >&2
    exit 1
  fi

  git -C "$repo_root" fetch --quiet --prune origin
  git -C "$repo_root" checkout --quiet "$ref"
  git -C "$repo_root" pull --quiet --ff-only origin "$ref"
}

bootstrap_repo() {
  require_home
  require_cmd git

  repo_url="${SKILLHUB_REPO_URL:-$default_repo_url}"
  ref="${SKILLHUB_REF:-$default_ref}"
  repo_root="${SKILLHUB_HOME:-$HOME/.local/share/skillhub}"

  if [ -d "$repo_root/.git" ]; then
    sync_repo "$repo_root" "$ref"
  elif [ -e "$repo_root" ]; then
    printf 'SKILLHUB_HOME exists but is not a git checkout: %s\n' "$repo_root" >&2
    exit 1
  else
    mkdir -p "$(dirname -- "$repo_root")"
    git clone --quiet --branch "$ref" "$repo_url" "$repo_root"
  fi

  if ! is_checkout "$repo_root"; then
    printf 'Installed checkout is missing skillhub files: %s\n' "$repo_root" >&2
    exit 1
  fi

  printf '%s\n' "$repo_root"
}

if [ -z "$bin_dir" ]; then
  require_home
  bin_dir="$HOME/.local/bin"
fi

if repo_root=$(find_local_checkout); then
  ref="${SKILLHUB_REF:-$default_ref}"
  if [ "$update" -eq 1 ]; then
    sync_repo "$repo_root" "$ref"
  fi
else
  repo_root=$(bootstrap_repo)
fi

require_cmd go

mkdir -p "$bin_dir"

command_path="$bin_dir/skillhub"
binary_path="$bin_dir/.skillhub-bin"
binary_tmp_path="$binary_path.tmp.$$"
tmp_path="$command_path.tmp.$$"

version="${SKILLHUB_VERSION:-dev}"
commit="unknown"
if command -v git >/dev/null 2>&1 && [ -d "$repo_root/.git" ]; then
  commit=$(git -C "$repo_root" rev-parse --short HEAD 2>/dev/null || printf 'unknown')
  if [ -z "${SKILLHUB_VERSION:-}" ]; then
    version=$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || printf '%s' "$version")
  fi
fi
built=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
install_script="$repo_root/install.sh"

ldflags="-X github.com/assurrussa/skillhub/internal/cli.version=$version"
ldflags="$ldflags -X github.com/assurrussa/skillhub/internal/cli.commit=$commit"
ldflags="$ldflags -X github.com/assurrussa/skillhub/internal/cli.built=$built"
ldflags="$ldflags -X github.com/assurrussa/skillhub/internal/cli.repoPath=$repo_root"
ldflags="$ldflags -X github.com/assurrussa/skillhub/internal/cli.installScript=$install_script"

(cd "$repo_root" && go build -ldflags "$ldflags" -o "$binary_tmp_path" ./cmd/skillhub)
chmod 755 "$binary_tmp_path"
mv "$binary_tmp_path" "$binary_path"

{
  printf '#!/bin/sh\n'
  printf 'SKILLHUB_REPO=%s\n' "$(printf '%s\n' "$repo_root" | sed "s/'/'\\\\''/g; s/^/'/; s/$/'/")"
  printf 'export SKILLHUB_REPO\n'
  printf 'exec %s "$@"\n' "$(printf '%s\n' "$binary_path" | sed "s/'/'\\\\''/g; s/^/'/; s/$/'/")"
} > "$tmp_path"

chmod 755 "$tmp_path"
mv "$tmp_path" "$command_path"

printf 'Installed skillhub command to %s\n' "$command_path"
printf 'Source checkout: %s\n' "$repo_root"
case ":${PATH:-}:" in
  *":$bin_dir:"*)
    ;;
  *)
    printf 'Note: %s is not in PATH. Add it before using skillhub by name.\n' "$bin_dir"
    ;;
esac
printf 'Try: skillhub version\n'
