#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

usage() {
  printf '%s\n' \
    'Usage:' \
    '  sh install.sh [--bin-dir <dir>]' \
    '' \
    'Environment:' \
    '  SKILLHUB_BIN_DIR  Install directory for the skillhub command.' \
    '                    Defaults to $HOME/.local/bin.'
}

bin_dir="${SKILLHUB_BIN_DIR:-}"

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

if [ -z "$bin_dir" ]; then
  if [ -z "${HOME:-}" ]; then
    printf 'HOME is not set; pass --bin-dir or set SKILLHUB_BIN_DIR.\n' >&2
    exit 1
  fi
  bin_dir="$HOME/.local/bin"
fi

mkdir -p "$bin_dir"

if ! command -v go >/dev/null 2>&1; then
  printf '%s\n' 'Go is required to build the skillhub command with TUI support.' >&2
  exit 1
fi

command_path="$bin_dir/skillhub"
binary_path="$bin_dir/.skillhub-bin"
binary_tmp_path="$binary_path.tmp.$$"
tmp_path="$command_path.tmp.$$"

(cd "$repo_root" && go build -o "$binary_tmp_path" ./cmd/skillhub)
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
case ":${PATH:-}:" in
  *":$bin_dir:"*)
    ;;
  *)
    printf 'Note: %s is not in PATH. Add it before using skillhub by name.\n' "$bin_dir"
    ;;
esac
printf 'Try: skillhub skills list\n'
