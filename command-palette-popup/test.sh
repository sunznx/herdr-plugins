#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mode="${1:-all}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/herdr" <<'EOF'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$CPP_TEST_LOG"

if [ "$1" = "--default-config" ]; then
  printf '%s\n' '[keys]' '# prefix = "ctrl+b"'
  exit 0
fi

case "$1 $2" in
  "pane current")
    printf '%s\n' '{"result":{"pane":{"pane_id":"w9:p9","tab_id":"w9:t1","workspace_id":"w9"}}}'
    ;;
  "pane get")
    if [ "${3:-}" = "w5:p5" ] || [ "${3:-}" = "w7:p7" ]; then
      workspace="${3%%:*}"
      printf '{"result":{"pane":{"pane_id":"%s","tab_id":"%s:t1","workspace_id":"%s"}}}\n' "$3" "$workspace" "$workspace"
    else
      exit 1
    fi
    ;;
  "pane move")
    printf '{"result":{"move_result":{"changed":true,"previous_pane_id":"%s","pane":{"pane_id":"w3:p10","tab_id":"w3:t2","workspace_id":"w3"}}}}\n' "${3:-}"
    ;;
  "pane rename")
    printf '%s\n' '{"result":{"pane":{"pane_id":"w5:p5"}}}'
    ;;
  "plugin pane")
    printf '%s\n' '{"result":{"opened":true}}'
    ;;
  "plugin config-dir")
    printf '%s\n' "$CPP_TEST_TMP/config"
    ;;
  "plugin action")
    printf '%s\n' '{"result":{"actions":[{"plugin_id":"sunznx.herdr-move","action_id":"open","title":"Move pane to workspace…"},{"plugin_id":"sunznx.herdr-move","action_id":"tab","title":"Move pane to tab…"},{"plugin_id":"sunznx.herdr-rename","action_id":"open","title":"Rename pane and agent…"}]}}'
    ;;
  "tab list")
    printf '%s\n' '{"result":{"tabs":[{"tab_id":"w5:t1","workspace_id":"w5","number":1,"label":"source","pane_count":1},{"tab_id":"w3:t2","workspace_id":"w3","number":2,"label":"agents","pane_count":2}]}}'
    ;;
  "tab get")
    [ "${3:-}" = "w3:t2" ] || exit 1
    printf '%s\n' '{"result":{"tab":{"tab_id":"w3:t2"}}}'
    ;;
  "workspace list")
    printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w9","label":"source"},{"workspace_id":"w3","label":"herdr-plugin"}]}}'
    ;;
  "agent list")
    if [ "${FAKE_HAS_AGENT:-0}" = "1" ]; then
      printf '%s\n' '{"result":{"agents":[{"name":"old","agent":"codex","pane_id":"w5:p5"}]}}'
    else
      printf '%s\n' '{"result":{"agents":[]}}'
    fi
    ;;
  "agent rename")
    printf '%s\n' '{"result":{"agent":{"pane_id":"w5:p5"}}}'
    ;;
  "worktree list")
    printf '%s\n' '{"result":{"worktrees":[]}}'
    ;;
  *)
    printf 'unexpected fake herdr command: %s\n' "$*" >&2
    exit 2
    ;;
esac
EOF
chmod +x "$tmp/herdr"

export HERDR_BIN_PATH="$tmp/herdr"
export CPP_TEST_LOG="$tmp/herdr.log"
export CPP_TEST_TMP="$tmp"
export HERDR_CONFIG_PATH="$tmp/config.toml"

test_open() {
  : > "$CPP_TEST_LOG"
  grep -Fq 'placement = "popup"' "$script_dir/herdr-plugin.toml"

  LIVE_PANE_ID='w7:p7' \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5","focused_pane_cwd":"/tmp"}' \
    "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w7:p7' "$CPP_TEST_LOG"
  ! grep -Fq 'pane get w5:p5' "$CPP_TEST_LOG"
  ! grep -Fq 'pane current --current' "$CPP_TEST_LOG"
  grep -Fq -- '--env CPP_CONTEXT_JSON={"pane":"w7:p7","tab":"w7:t1","workspace":"w7","cwd":"/tmp"}' "$CPP_TEST_LOG"
  grep -Fq -- '--placement popup' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  env -u LIVE_PANE_ID \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5","focused_pane_cwd":"/tmp"}' \
    "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w5:p5' "$CPP_TEST_LOG"
  ! grep -Fq 'pane current --current' "$CPP_TEST_LOG"
  grep -Fq -- '--env CPP_CONTEXT_JSON={"pane":"w5:p5","tab":"w5:t1","workspace":"w5","cwd":"/tmp"}' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  env -u LIVE_PANE_ID \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:pAK","tab_id":"w5:t1","workspace_id":"w5","focused_pane_cwd":"/tmp"}' \
    "$script_dir/open.sh" >/dev/null
  grep -Fq 'pane get w5:pAK' "$CPP_TEST_LOG"
  grep -Fq 'pane current --current' "$CPP_TEST_LOG"
  grep -Fq -- '--env CPP_CONTEXT_JSON={"pane":"w9:p9","tab":"w9:t1","workspace":"w9","cwd":"/tmp"}' "$CPP_TEST_LOG"
  ! grep -Fq -- '"pane":"w5:pAK"' "$CPP_TEST_LOG"
}

test_palette() {
  : > "$CPP_TEST_LOG"
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_LIST_ONLY=1 \
    "$script_dir/palette.sh" >"$tmp/list.out"
  [ "$(grep -Fc 'Move pane to tab…' "$tmp/list.out")" -eq 1 ]
  [ "$(grep -Fc 'Rename pane and agent…' "$tmp/list.out")" -eq 1 ]
  ! grep -Fq 'Plugin: Move pane to tab…' "$tmp/list.out"
  ! grep -Fq 'Plugin: Rename pane and agent…' "$tmp/list.out"

  : > "$CPP_TEST_LOG"
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5","focused_pane_cwd":"/tmp"}' \
    CPP_CONTEXT_JSON='{"pane":"w9:p9","tab":"w9:t1","workspace":"w9","cwd":"/wrong"}' \
    CPP_CHOICE=$'static\tmove_pane_workspace' \
    CPP_PICK_VALUE='w3' \
    "$script_dir/palette.sh" >/dev/null
  if ! grep -Fq 'pane move w5:p5 --new-tab --workspace w3 --focus' "$CPP_TEST_LOG"; then
    cat "$CPP_TEST_LOG" >&2
    return 1
  fi

  : > "$CPP_TEST_LOG"
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\tmove_pane_tab' \
    CPP_PICK_VALUE='w3:t2' \
    "$script_dir/palette.sh" >/dev/null
  grep -Fq 'pane get w5:p5' "$CPP_TEST_LOG"
  grep -Fq 'tab get w3:t2' "$CPP_TEST_LOG"
  grep -Fq 'pane move w5:p5 --tab w3:t2 --split right --focus' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  printf 'reviewer-2\n' | FAKE_HAS_AGENT=1 \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\trename_pane_agent' \
    "$script_dir/palette.sh" >/dev/null
  grep -Fq 'agent rename w5:p5 reviewer-2' "$CPP_TEST_LOG"
  grep -Fq 'pane rename w5:p5 reviewer-2' "$CPP_TEST_LOG"
  agent_line="$(grep -nF 'agent rename w5:p5 reviewer-2' "$CPP_TEST_LOG" | cut -d: -f1)"
  pane_line="$(grep -nF 'pane rename w5:p5 reviewer-2' "$CPP_TEST_LOG" | cut -d: -f1)"
  [ "$agent_line" -lt "$pane_line" ]

  : > "$CPP_TEST_LOG"
  printf 'Build Logs\n' | \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\trename_pane_agent' \
    "$script_dir/palette.sh" >/dev/null
  ! grep -Fq 'agent rename ' "$CPP_TEST_LOG"
  grep -Fq 'pane rename w5:p5 Build Logs' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  if printf 'Invalid Name\n' | FAKE_HAS_AGENT=1 \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\trename_pane_agent' \
    "$script_dir/palette.sh" >/dev/null 2>&1; then
    printf 'expected invalid agent name to fail\n' >&2
    return 1
  fi
  ! grep -Fq 'agent rename ' "$CPP_TEST_LOG"
  ! grep -Fq 'pane rename ' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  if HERDR_PANE_ID='w5:p5' \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\tmove_pane_workspace' \
    CPP_PICK_VALUE='w3' \
    "$script_dir/palette.sh" </dev/null >/dev/null 2>&1; then
    printf 'expected palette self-move guard to fail\n' >&2
    return 1
  fi
  ! grep -Fq 'pane move ' "$CPP_TEST_LOG"

  : > "$CPP_TEST_LOG"
  if CPP_CONTEXT_JSON='{"pane":"w9:p9","tab":"w9:t1","workspace":"w9","cwd":"/private/forwarded"}' \
    CPP_CHOICE=$'static\tmove_pane_workspace' \
    CPP_PICK_VALUE='w3' \
    "$script_dir/palette.sh" </dev/null >"$tmp/missing-popup.out" 2>&1; then
    printf 'expected missing popup context to fail safely\n' >&2
    return 1
  fi
  ! grep -Fq 'pane get ' "$CPP_TEST_LOG"
  ! grep -Fq 'pane move ' "$CPP_TEST_LOG"
  ! grep -Fq '/private/forwarded' "$tmp/missing-popup.out"

  mkdir -p "$tmp/bin"
  cat > "$tmp/bin/fzf" <<'EOF'
#!/usr/bin/env bash
exit 130
EOF
  chmod +x "$tmp/bin/fzf"
  : > "$CPP_TEST_LOG"
  env -u HERDR_PANE_ID -u CPP_PICK_VALUE \
    PATH="$tmp/bin:$PATH" \
    HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w5:p5","tab_id":"w5:t1","workspace_id":"w5"}' \
    CPP_CHOICE=$'static\tmove_pane_workspace' \
    "$script_dir/palette.sh" </dev/null >/dev/null 2>&1
  grep -Fq 'pane get w5:p5' "$CPP_TEST_LOG"
  ! grep -Fq 'pane move ' "$CPP_TEST_LOG"
}

case "$mode" in
  open) test_open ;;
  palette) test_palette ;;
  all) test_open; test_palette ;;
  *) printf 'unknown test mode: %s\n' "$mode" >&2; exit 2 ;;
esac

printf 'ok\n'
