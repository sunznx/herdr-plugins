#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/normal" "$tmp/other" "$tmp/bin"

cat >"$tmp/herdr" <<'SH'
#!/usr/bin/env bash
set -eu
printf '%s\n' "$*" >>"$CALLS"
case "$1 $2" in
  "plugin pane") printf '{"result":{"opened":true}}\n' ;;
  "workspace list")
    if [ "${SCRATCH_EXISTS:-}" = 1 ]; then
      printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w2","label":"scratch","active_tab_id":"w2:t1"},{"workspace_id":"w1","label":"main","active_tab_id":"w1:t1"},{"workspace_id":"w3","label":"other","active_tab_id":"w3:t1"}]}}'
    else
      printf '%s\n' '{"result":{"workspaces":[{"workspace_id":"w1","label":"main","active_tab_id":"w1:t1"},{"workspace_id":"w3","label":"other","active_tab_id":"w3:t1"}]}}'
    fi
    ;;
  "pane list")
    case "$4" in
      w1) printf '{"result":{"panes":[{"tab_id":"w1:t1","cwd":"%s"}]}}\n' "$NORMAL_DIR" ;;
      w3) printf '{"result":{"panes":[{"tab_id":"w3:t1","cwd":"%s"}]}}\n' "$OTHER_DIR" ;;
      *) exit 1 ;;
    esac
    ;;
  "workspace create") printf '{"result":{"root_pane":{"pane_id":"w4:p1"}}}\n' ;;
  "tab create")
    workspace=""
    while [ "$#" -gt 0 ]; do
      [ "$1" = "--workspace" ] && workspace="$2"
      shift
    done
    printf '{"result":{"root_pane":{"pane_id":"%s:p2"}}}\n' "$workspace"
    ;;
  "pane run") ;;
  "pane send-keys") ;;
  "pane get")
    case "$3" in
      pclose)
        if [ -f "$ARCHIVED" ]; then
          agent="" status=""
        elif [ -f "$FAILED" ]; then
          agent='codex' status='idle'
        elif [ -f "$BLOCKED" ]; then
          agent='codex' status='working'
        else
          agent='codex' status='idle'
        fi
        printf '{"result":{"pane":{"pane_id":"pclose","tab_id":"tclose","agent":"%s","agent_status":"%s"}}}\n' "$agent" "$status"
        ;;
      pother) printf '%s\n' '{"result":{"pane":{"pane_id":"pother","tab_id":"tother","agent":"claude"}}}' ;;
      *) exit 1 ;;
    esac
    ;;
  "pane read")
    case "$3" in
      w2:p2|w4:p1) printf 'Do you trust the contents of this directory?\n' ;;
      pclose)
        [ -f "$BLOCKED" ] && printf 'Archive this session?\n'
        [ -f "$FAILED" ] && printf 'Failed to archive current thread: failed to archive session\n'
        ;;
    esac
    ;;
  "pane current") printf '%s\n' '{"result":{"pane":{"pane_id":"pclose","tab_id":"tclose","agent":"codex"}}}' ;;
  "agent prompt") [ "$3 $4" = "pclose /archive" ] && touch "$BLOCKED" ;;
  "agent send-keys")
    [ "$3 $4 $5" = "pclose down enter" ]
    if [ "${ARCHIVE_FAIL:-}" = 1 ]; then touch "$FAILED"; else touch "$ARCHIVED"; fi
    ;;
  "tab close") printf '%s\n' '{"error":{"code":"tab_not_found","message":"tab not found"}}' >&2; exit 1 ;;
  *) printf 'unexpected command: %s\n' "$*" >&2; exit 2 ;;
esac
SH
chmod +x "$tmp/herdr"

export HERDR_BIN_PATH="$tmp/herdr"
export CALLS="$tmp/calls"
export NORMAL_DIR="$tmp/normal"
export OTHER_DIR="$tmp/other"
export ARCHIVED="$tmp/archived"
export BLOCKED="$tmp/blocked"
export FAILED="$tmp/failed"

: >"$CALLS"
"$root/open.sh" >/dev/null
grep -Fqx 'plugin pane open --plugin sunznx.herdr-new-codex --entrypoint picker --focus' "$CALLS"

: >"$CALLS"
HERDR_NEW_CODEX_CHOICE=w1 "$root/picker.sh"
grep -Fqx "tab create --workspace w1 --cwd $NORMAL_DIR --focus" "$CALLS"
grep -Fqx 'pane run w1:p2 exec codex' "$CALLS"

: >"$CALLS"
SCRATCH_EXISTS=1 HERDR_NEW_CODEX_CHOICE=w2 TMPDIR="$tmp" "$root/picker.sh"
grep -Eq "^tab create --workspace w2 --cwd $tmp/herdr-scratch-.{6} --focus$" "$CALLS"
grep -Fqx 'pane run w2:p2 exec codex' "$CALLS"
grep -Fqx 'pane read w2:p2 --source visible --lines 60' "$CALLS"
grep -Fqx 'pane send-keys w2:p2 enter' "$CALLS"

: >"$CALLS"
HERDR_NEW_CODEX_CHOICE=__scratch__ TMPDIR="$tmp" "$root/picker.sh"
grep -Eq "^workspace create --label scratch --cwd $tmp/herdr-scratch-.{6} --focus$" "$CALLS"
grep -Fqx 'pane run w4:p1 exec codex' "$CALLS"
grep -Fqx 'pane read w4:p1 --source visible --lines 60' "$CALLS"
grep -Fqx 'pane send-keys w4:p1 enter' "$CALLS"
! grep -Fq 'trust_level' "$CALLS"

cat >"$tmp/bin/fzf" <<'SH'
#!/usr/bin/env bash
cat >"$FZF_INPUT"
exit 130
SH
chmod +x "$tmp/bin/fzf"
: >"$CALLS"
PATH="$tmp/bin:$PATH" FZF_INPUT="$tmp/fzf-input" SCRATCH_EXISTS=1 "$root/picker.sh"
[ "$(tail -n1 "$tmp/fzf-input")" = $'w2\tscratch' ]
! grep -Eq '^(workspace|tab) create ' "$CALLS"

: >"$CALLS"
rm -f "$ARCHIVED"
rm -f "$BLOCKED"
rm -f "$FAILED"
env -u LIVE_PANE_ID HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"pclose"}' "$root/close.sh" >/dev/null
grep -Fqx 'agent prompt pclose /archive' "$CALLS"
grep -Fqx 'agent send-keys pclose down enter' "$CALLS"
grep -Fqx 'tab close tclose' "$CALLS"

: >"$CALLS"
rm -f "$ARCHIVED" "$BLOCKED" "$FAILED"
ARCHIVE_FAIL=1 env -u LIVE_PANE_ID HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"pclose"}' "$root/close.sh" >/dev/null
grep -Fqx 'tab close tclose' "$CALLS"

: >"$CALLS"
set +e
env -u LIVE_PANE_ID HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"pother"}' "$root/close.sh" >/dev/null 2>&1
status=$?
set -e
[ "$status" -ne 0 ]
! grep -Fq 'agent prompt' "$CALLS"
! grep -Fq 'tab close' "$CALLS"

echo ok
