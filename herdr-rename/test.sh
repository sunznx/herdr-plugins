#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mode="${1:-all}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/herdr" <<'EOF'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$HERDR_RENAME_TEST_LOG"

case "$1 $2" in
  "pane get")
    if [ "${3:-}" = "w7:p7" ] || [ "${3:-}" = "w9:p9" ]; then
      printf '{"result":{"pane":{"pane_id":"%s","workspace_id":"%s"}}}\n' "$3" "${3%%:*}"
      exit 0
    fi
    exit 1
    ;;
  "pane current")
    [ "${FAKE_CURRENT_FAIL:-0}" != "1" ] || exit 1
    printf '%s\n' '{"result":{"pane":{"pane_id":"w9:p9","workspace_id":"w9"}}}'
    ;;
  "plugin pane")
    printf '%s\n' '{"result":{"opened":true}}'
    ;;
  "agent list")
    if [ "${FAKE_HAS_AGENT:-0}" = "1" ]; then
      printf '%s\n' '{"result":{"agents":[{"name":"old-name","agent":"codex","pane_id":"w9:p9"}]}}'
    else
      printf '%s\n' '{"result":{"agents":[]}}'
    fi
    ;;
  "agent rename")
    if [ "${FAKE_AGENT_RENAME_FAIL:-0}" = "1" ]; then
      printf '%s\n' '{"error":{"code":"agent_name_taken","message":"agent name is already in use"}}' >&2
      exit 1
    fi
    printf '%s\n' '{"result":{"agent":{"pane_id":"w9:p9"}}}'
    ;;
  "pane rename")
    if [ "${FAKE_PANE_RENAME_FAIL:-0}" = "1" ]; then
      printf '%s\n' '{"error":{"code":"pane_not_found","message":"pane not found"}}' >&2
      exit 1
    fi
    printf '%s\n' '{"result":{"pane":{"pane_id":"w9:p9"}}}'
    ;;
  *)
    printf 'unexpected fake herdr command: %s\n' "$*" >&2
    exit 2
    ;;
esac
EOF
chmod +x "$tmp/herdr"

export HERDR_BIN_PATH="$tmp/herdr"
export HERDR_RENAME_TEST_LOG="$tmp/herdr.log"

test_open() {
  : > "$HERDR_RENAME_TEST_LOG"
  env -u LIVE_PANE_ID "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane current --current' "$HERDR_RENAME_TEST_LOG"
  grep -Fq -- '--placement popup' "$HERDR_RENAME_TEST_LOG"
  grep -Fq -- '--env HERDR_RENAME_PANE_ID=w9:p9' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  LIVE_PANE_ID="w5:pAK" "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w5:pAK' "$HERDR_RENAME_TEST_LOG"
  grep -Fq 'pane current --current' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq -- '--env HERDR_RENAME_PANE_ID=w5:pAK' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  LIVE_PANE_ID="w7:p7" "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w7:p7' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'pane current --current' "$HERDR_RENAME_TEST_LOG"
  grep -Fq -- '--env HERDR_RENAME_PANE_ID=w7:p7' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  if env -u LIVE_PANE_ID FAKE_CURRENT_FAIL=1 "$script_dir/open.sh" >/dev/null 2>&1; then
    printf 'expected missing caller context to fail\n' >&2
    return 1
  fi
  ! grep -Fq 'plugin pane open' "$HERDR_RENAME_TEST_LOG"
}

test_rename() {
  : > "$HERDR_RENAME_TEST_LOG"
  HERDR_RENAME_PANE_ID="w9:p9" \
    HERDR_RENAME_NAME="Build Logs" \
    "$script_dir/rename.sh" >/dev/null
  grep -Fq 'pane rename w9:p9 Build Logs' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'agent rename ' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  FAKE_HAS_AGENT=1 \
    HERDR_RENAME_PANE_ID="w9:p9" \
    HERDR_RENAME_NAME="reviewer-2" \
    "$script_dir/rename.sh" >/dev/null
  grep -Fq 'agent rename w9:p9 reviewer-2' "$HERDR_RENAME_TEST_LOG"
  grep -Fq 'pane rename w9:p9 reviewer-2' "$HERDR_RENAME_TEST_LOG"
  agent_line="$(grep -nF 'agent rename w9:p9 reviewer-2' "$HERDR_RENAME_TEST_LOG" | cut -d: -f1)"
  pane_line="$(grep -nF 'pane rename w9:p9 reviewer-2' "$HERDR_RENAME_TEST_LOG" | cut -d: -f1)"
  [ "$agent_line" -lt "$pane_line" ]

  : > "$HERDR_RENAME_TEST_LOG"
  if FAKE_HAS_AGENT=1 \
    HERDR_RENAME_PANE_ID="w9:p9" \
    HERDR_RENAME_NAME="Invalid Name" \
    "$script_dir/rename.sh" >/dev/null 2>&1; then
    printf 'expected invalid agent name to fail\n' >&2
    return 1
  fi
  ! grep -Fq 'agent rename ' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'pane rename ' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  if failure_output="$(FAKE_HAS_AGENT=1 \
    FAKE_AGENT_RENAME_FAIL=1 \
    HERDR_RENAME_PANE_ID="w9:p9" \
    HERDR_RENAME_NAME="reviewer" \
    "$script_dir/rename.sh" 2>&1)"; then
    printf 'expected agent rename failure\n' >&2
    return 1
  fi
  printf '%s' "$failure_output" | grep -Fq 'agent name is already in use'
  grep -Fq 'agent rename w9:p9 reviewer' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'pane rename ' "$HERDR_RENAME_TEST_LOG"

  : > "$HERDR_RENAME_TEST_LOG"
  HERDR_RENAME_PANE_ID="w9:p9" \
    HERDR_RENAME_NAME="" \
    "$script_dir/rename.sh" >/dev/null
  ! grep -Fq 'agent rename ' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'pane rename ' "$HERDR_RENAME_TEST_LOG"

  mkdir -p "$tmp/bin"
  cat > "$tmp/bin/fzf" <<'EOF'
#!/usr/bin/env bash
exit 130
EOF
  chmod +x "$tmp/bin/fzf"
  : > "$HERDR_RENAME_TEST_LOG"
  env -u HERDR_RENAME_NAME \
    PATH="$tmp/bin:$PATH" \
    HERDR_RENAME_PANE_ID="w9:p9" \
    "$script_dir/rename.sh" >/dev/null
  ! grep -Fq 'agent rename ' "$HERDR_RENAME_TEST_LOG"
  ! grep -Fq 'pane rename ' "$HERDR_RENAME_TEST_LOG"
}

case "$mode" in
  open) test_open ;;
  rename) test_rename ;;
  all) test_open; test_rename ;;
  *) printf 'unknown test mode: %s\n' "$mode" >&2; exit 2 ;;
esac

printf 'ok\n'
