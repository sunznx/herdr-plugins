#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mode="${1:-all}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/herdr" <<'EOF'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$HERDR_MOVE_TEST_LOG"

case "$1 $2" in
  "pane get")
    if [ "${3:-}" = "w7:p7" ] || [ "${3:-}" = "w9:p9" ]; then
      workspace="${3%%:*}"
      printf '{"result":{"pane":{"pane_id":"%s","workspace_id":"%s","tab_id":"%s:t1"}}}\n' "$3" "$workspace" "$workspace"
      exit 0
    fi
    exit 1
    ;;
  "pane current")
    [ "${FAKE_CURRENT_FAIL:-0}" != "1" ] || exit 1
    printf '{"result":{"pane":{"pane_id":"w9:p9","workspace_id":"w9","tab_id":"w9:t1"}}}\n'
    ;;
  "plugin pane")
    printf '{"result":{"opened":true}}\n'
    ;;
  "workspace list")
    printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w9","label":"source"},{"workspace_id":"w3","label":"herdr-plugin"}]}}'
    ;;
  "pane move")
    printf '%s\n' '{"result":{"move_result":{"changed":true,"previous_pane_id":"w9:p9","pane":{"pane_id":"w3:p10","workspace_id":"w3","tab_id":"w3:t4"}}}}'
    ;;
  *)
    printf 'unexpected fake herdr command: %s\n' "$*" >&2
    exit 2
    ;;
esac
EOF
chmod +x "$tmp/herdr"

export HERDR_BIN_PATH="$tmp/herdr"
export HERDR_MOVE_TEST_LOG="$tmp/herdr.log"

test_open() {
  : > "$HERDR_MOVE_TEST_LOG"
  env -u LIVE_PANE_ID "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane current --current' "$HERDR_MOVE_TEST_LOG"
  grep -Fq -- '--env HERDR_MOVE_PANE_ID=w9:p9 --env HERDR_MOVE_WORKSPACE_ID=w9' "$HERDR_MOVE_TEST_LOG"

  : > "$HERDR_MOVE_TEST_LOG"
  LIVE_PANE_ID="w5:pAK" "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w5:pAK' "$HERDR_MOVE_TEST_LOG"
  grep -Fq 'pane current --current' "$HERDR_MOVE_TEST_LOG"
  grep -Fq -- '--env HERDR_MOVE_PANE_ID=w9:p9 --env HERDR_MOVE_WORKSPACE_ID=w9' "$HERDR_MOVE_TEST_LOG"
  ! grep -Fq -- '--env HERDR_MOVE_PANE_ID=w5:pAK' "$HERDR_MOVE_TEST_LOG"

  : > "$HERDR_MOVE_TEST_LOG"
  LIVE_PANE_ID="w7:p7" "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w7:p7' "$HERDR_MOVE_TEST_LOG"
  ! grep -Fq 'pane current --current' "$HERDR_MOVE_TEST_LOG"
  grep -Fq -- '--env HERDR_MOVE_PANE_ID=w7:p7 --env HERDR_MOVE_WORKSPACE_ID=w7' "$HERDR_MOVE_TEST_LOG"

  : > "$HERDR_MOVE_TEST_LOG"
  if env -u LIVE_PANE_ID FAKE_CURRENT_FAIL=1 "$script_dir/open.sh" >/dev/null 2>&1; then
    printf 'expected missing caller context to fail\n' >&2
    return 1
  fi
  ! grep -Fq 'plugin pane open' "$HERDR_MOVE_TEST_LOG"
}

test_move() {
  : > "$HERDR_MOVE_TEST_LOG"
  output="$(
    HERDR_MOVE_PANE_ID="w9:p9" \
    HERDR_MOVE_WORKSPACE_ID="w9" \
    HERDR_MOVE_CHOICE="w3" \
    "$script_dir/move.sh"
  )"
  grep -Fq 'pane move w9:p9 --new-tab --workspace w3 --focus' "$HERDR_MOVE_TEST_LOG"
  printf '%s' "$output" | grep -Fq 'w3:p10'

  mkdir -p "$tmp/bin"
  cat > "$tmp/bin/fzf" <<'EOF'
#!/usr/bin/env bash
exit 130
EOF
  chmod +x "$tmp/bin/fzf"
  : > "$HERDR_MOVE_TEST_LOG"
  PATH="$tmp/bin:$PATH" \
    HERDR_MOVE_PANE_ID="w9:p9" \
    HERDR_MOVE_WORKSPACE_ID="w9" \
    "$script_dir/move.sh"
  ! grep -Fq 'pane move ' "$HERDR_MOVE_TEST_LOG"
}

case "$mode" in
  open) test_open ;;
  move) test_move ;;
  all) test_open; test_move ;;
  *) printf 'unknown test mode: %s\n' "$mode" >&2; exit 2 ;;
esac

printf 'ok\n'
