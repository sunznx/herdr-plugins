#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/herdr" <<'EOF'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$AI_RENAME_HERDR_LOG"
case "$1 $2" in
  "pane get"|"pane current")
    printf '%s\n' '{"result":{"pane":{"pane_id":"w1:p1","cwd":"/repo","terminal_title_stripped":"build api"}}}'
    ;;
  "pane list")
    printf '%s\n' '{"result":{"panes":[{"pane_id":"w1:p1","cwd":"/repo","terminal_title_stripped":"build api"},{"pane_id":"w1:p2","cwd":"/docs","terminal_title_stripped":"write docs"}]}}'
    ;;
  "pane read")
    printf 'working in %s\n' "$3"
    ;;
  "pane rename"|"agent rename")
    printf '%s\n' '{"result":{"ok":true}}'
    ;;
  "agent list")
    printf '%s\n' '{"result":{"agents":[{"pane_id":"w1:p1","agent":"codex"}]}}'
    ;;
  *) exit 2 ;;
esac
EOF

cat > "$tmp/codex" <<'EOF'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$AI_RENAME_CODEX_LOG"
input="$(cat)"
output=""
while [ $# -gt 0 ]; do
  [ "$1" != "--output-last-message" ] || { output="$2"; shift; }
  shift
done
[ "${AI_RENAME_CODEX_FAIL:-0}" != "1" ] || exit 1
sleep "${AI_RENAME_CODEX_SLEEP:-0}"
if [ -n "${AI_RENAME_CODEX_OUTPUT:-}" ]; then
  name="$AI_RENAME_CODEX_OUTPUT"
elif [[ "$input" == *"w1:p2"* ]]; then
  name="write-docs"
else
  name="build-api"
fi
printf '%s\n' "$name" > "$output"
EOF
chmod +x "$tmp/herdr" "$tmp/codex"

export HERDR_BIN_PATH="$tmp/herdr"
export CODEX_BIN_PATH="$tmp/codex"
export AI_RENAME_HERDR_LOG="$tmp/herdr.log"
export AI_RENAME_CODEX_LOG="$tmp/codex.log"

grep -Fq 'command = ["bash", "launch.sh", "current"]' "$script_dir/herdr-plugin.toml"
grep -Fq 'command = ["bash", "launch.sh", "all"]' "$script_dir/herdr-plugin.toml"

: > "$AI_RENAME_HERDR_LOG"
: > "$AI_RENAME_CODEX_LOG"
HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p1"}' "$script_dir/rename.sh" current >/dev/null
grep -Fq 'agent rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG"
grep -Fq 'pane rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG"
grep -Fq -- '--model gpt-5.3-codex-spark' "$AI_RENAME_CODEX_LOG"
agent_line="$(grep -nF 'agent rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG" | cut -d: -f1)"
pane_line="$(grep -nF 'pane rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG" | cut -d: -f1)"
[ "$agent_line" -lt "$pane_line" ]

: > "$AI_RENAME_HERDR_LOG"
"$script_dir/rename.sh" all >/dev/null
grep -Fq 'agent rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG"
grep -Fq 'pane rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG"
grep -Fq 'pane rename w1:p2 write-docs' "$AI_RENAME_HERDR_LOG"
if grep -Fq 'agent rename w1:p2' "$AI_RENAME_HERDR_LOG"; then
  printf 'rename-all renamed a pane without an agent\n' >&2
  exit 1
fi

: > "$AI_RENAME_HERDR_LOG"
if AI_RENAME_CODEX_OUTPUT='Invalid Name' \
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p1"}' \
  "$script_dir/rename.sh" current >/dev/null 2>&1; then
  printf 'expected invalid Codex output to fail\n' >&2
  exit 1
fi
if grep -Fq ' rename ' "$AI_RENAME_HERDR_LOG"; then
  printf 'invalid output must not rename anything\n' >&2
  exit 1
fi

: > "$AI_RENAME_HERDR_LOG"
if AI_RENAME_CODEX_FAIL=1 \
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p1"}' \
  "$script_dir/rename.sh" current >/dev/null 2>&1; then
  printf 'expected Codex failure to fail\n' >&2
  exit 1
fi
if grep -Fq ' rename ' "$AI_RENAME_HERDR_LOG"; then
  printf 'Codex failure must not rename anything\n' >&2
  exit 1
fi

: > "$AI_RENAME_HERDR_LOG"
AI_RENAME_CODEX_SLEEP=0.3 \
  HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p1"}' \
  "$script_dir/launch.sh" current >/dev/null
if grep -Fq 'pane rename ' "$AI_RENAME_HERDR_LOG"; then
  printf 'launcher waited for the background worker\n' >&2
  exit 1
fi
for _ in {1..40}; do
  grep -Fq 'pane rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG" && break
  sleep 0.05
done
grep -Fq 'pane rename w1:p1 build-api' "$AI_RENAME_HERDR_LOG"

printf 'ok\n'
