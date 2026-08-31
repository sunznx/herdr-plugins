#!/usr/bin/env bash
# Pane `sunznx.command-palette-popup.palette`: the interactive fzf command palette.
#
# Runs inside a popup pane with a real TTY. It combines Herdr's native
# tab/pane/workspace/worktree/agent commands, live navigation targets, and every
# action exposed by installed plugins in one searchable list.
#
# Each row also shows the keybinding herdr itself has for that action, resolved
# live from `herdr --default-config` + the user's config.toml (see keys_table
# below) — so the picker teaches the shortcut instead of hiding it.
#
# Ranking: native commands and plugin actions are sorted by how often (and how
# recently) you've picked them. Live jump targets are never reordered because
# their identities change across sessions.
#
# Debug env vars: CPP_LIST_ONLY=1 prints the generated rows (TSV) and exits
# without fzf; CPP_CHOICE="<kind><TAB><payload>" preselects a row;
# CPP_PICK_VALUE supplies a nested picker result; CPP_DRY_RUN=1 prints the
# `herdr` command a selection would run instead of running it.
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
herdr_bin="${HERDR_BIN_PATH:-herdr}"
self_plugin="${HERDR_PLUGIN_ID:-sunznx.command-palette-popup}"

if [ "${1:-}" = "--self-test" ]; then
  exec "$script_dir/test.sh" palette
fi

die() {
  printf '%s\n' "$*" >&2
  if [ -t 0 ]; then
    printf 'Press any key to close…' >&2
    read -r -n1 _ 2>/dev/null || sleep 2
  fi
  exit 1
}

command -v fzf >/dev/null 2>&1 || die "command-palette-popup: fzf is not installed or not on PATH."
command -v jq  >/dev/null 2>&1 || die "command-palette-popup: jq is not installed or not on PATH."

# --- Origin context, forwarded by open.sh as a single JSON blob. It is a
# fallback for general commands; pane-to-workspace moves require the popup's
# authoritative invocation context below. ---
ctx="${CPP_CONTEXT_JSON:-}"
[ -n "$ctx" ] || ctx='{}'
pane="$(printf '%s' "$ctx" | jq -r '.pane // empty' 2>/dev/null)"
tab="$(printf '%s' "$ctx" | jq -r '.tab // empty' 2>/dev/null)"
workspace="$(printf '%s' "$ctx" | jq -r '.workspace // empty' 2>/dev/null)"
cwd="$(printf '%s' "$ctx" | jq -r '.cwd // empty' 2>/dev/null)"
# A real popup is not a Herdr pane, so Herdr keeps the tiled pane underneath it
# in the popup's own invocation context. Prefer that authoritative snapshot to
# the forwarded fallback prepared by open.sh.
popup_ctx="${HERDR_PLUGIN_CONTEXT_JSON:-}"
popup_pane=""
if [ -n "$popup_ctx" ]; then
  popup_pane="$(printf '%s' "$popup_ctx" | jq -r '.focused_pane_id // empty' 2>/dev/null)"
  if [ -n "$popup_pane" ]; then
    pane="$popup_pane"
    popup_tab="$(printf '%s' "$popup_ctx" | jq -r '.tab_id // empty' 2>/dev/null)"
    popup_workspace="$(printf '%s' "$popup_ctx" | jq -r '.workspace_id // empty' 2>/dev/null)"
    popup_cwd="$(printf '%s' "$popup_ctx" | jq -r '.focused_pane_cwd // .workspace_cwd // empty' 2>/dev/null)"
    [ -n "$popup_tab" ] && tab="$popup_tab"
    [ -n "$popup_workspace" ] && workspace="$popup_workspace"
    [ -n "$popup_cwd" ] && cwd="$popup_cwd"
  fi
fi

# --- Usage tracking (drives the ranking) ---
# Stored as {"<action-id>": {"count": N, "last": <epoch>}}. Plain integers
# written by earlier versions are still read (treated as count with last=0).
usage_dir="$("$herdr_bin" plugin config-dir sunznx.command-palette-popup 2>/dev/null | tr -d '\n')"
usage_file=""
if [ -n "$usage_dir" ]; then
  mkdir -p "$usage_dir" 2>/dev/null && usage_file="$usage_dir/usage.json"
  [ -n "$usage_file" ] && [ ! -f "$usage_file" ] && echo '{}' > "$usage_file" 2>/dev/null
fi
counts_json='{}'
if [ -n "$usage_file" ] && [ -r "$usage_file" ]; then
  counts_json="$(jq -c '.' "$usage_file" 2>/dev/null)"
  [ -n "$counts_json" ] || counts_json='{}'
fi

record_usage() {
  local id="$1" tmp now
  [ -n "$usage_file" ] || return 0
  [ "${CPP_DRY_RUN:-0}" = "1" ] && return 0
  now="$(date +%s 2>/dev/null || echo 0)"
  tmp="$(mktemp "${usage_file}.XXXXXX" 2>/dev/null)" || return 0
  if jq --arg id "$id" --argjson now "$now" '
        (.[$id] // 0) as $prev
        | (if ($prev | type) == "object" then ($prev.count // 0) else $prev end) as $count
        | .[$id] = {count: ($count + 1), last: $now}
      ' "$usage_file" > "$tmp" 2>/dev/null; then
    mv "$tmp" "$usage_file"
  else
    rm -f "$tmp"
  fi
}

# --- Keybinding resolution -------------------------------------------------
# herdr has no "list my effective keybindings" API (checked: nothing in
# `herdr api schema`), so we reconstruct it from the two files that define it:
#
#   1. `herdr --default-config` — the installed binary's own defaults, which
#      ship as COMMENTED assignments under [keys] (so it tracks herdr versions
#      instead of us hardcoding a table that silently rots).
#   2. the user's config.toml — uncommented [keys] entries win; an explicit ""
#      means the user unbound it.
#
# A key claimed by a [[keys.command]] block (custom command / plugin action)
# shadows the built-in that shipped with it, so we report those as unbound —
# showing a shortcut that no longer fires is worse than showing none.

config_path="${HERDR_CONFIG_PATH:-$HOME/.config/herdr/config.toml}"

# Emit `name<TAB>value` for every scalar in the [keys] table on stdin.
# $1=1 first strips one leading comment marker (for --default-config).
keys_table() {
  awk -v uncomment="${1:-0}" '
    { line = $0 }
    uncomment == 1 { sub(/^[[:space:]]*#[[:space:]]?/, "", line) }
    # Any table header ends the [keys] scalar run — including [keys.indexed]
    # and [[keys.command]], whose own `key =` must not be read as a built-in.
    line ~ /^[[:space:]]*\[/ {
      inkeys = (line ~ /^[[:space:]]*\[keys\][[:space:]]*$/)
      next
    }
    !inkeys { next }
    match(line, /^[[:space:]]*[a-z_][a-z0-9_]*[[:space:]]*=[[:space:]]*"[^"]*"/) {
      s = substr(line, RSTART, RLENGTH)
      eq = index(s, "=")
      name = substr(s, 1, eq - 1); gsub(/[[:space:]]/, "", name)
      q = index(s, "\"")
      printf "%s\t%s\n", name, substr(s, q + 1, length(s) - q - 1)
    }
  '
}

# Keys claimed by [[keys.command]] blocks in the user's config.
cmd_keys() {
  awk '
    /^[[:space:]]*\[/ { incmd = ($0 ~ /^[[:space:]]*\[\[keys\.command\]\][[:space:]]*$/); next }
    !incmd { next }
    match($0, /^[[:space:]]*key[[:space:]]*=[[:space:]]*"[^"]*"/) {
      s = substr($0, RSTART, RLENGTH)
      q = index(s, "\"")
      print substr(s, q + 1, length(s) - q - 1)
    }
  '
}

tsv_to_object() { jq -R -s 'split("\n") | map(select(index("\t"))) | map(split("\t")) | map({key: .[0], value: .[1]}) | from_entries'; }

default_keys_json="$("$herdr_bin" --default-config 2>/dev/null | keys_table 1 | tsv_to_object)"
[ -n "$default_keys_json" ] || default_keys_json='{}'
user_keys_json='{}'
shadow_json='[]'
if [ -r "$config_path" ]; then
  user_keys_json="$(keys_table 0 < "$config_path" | tsv_to_object)"
  [ -n "$user_keys_json" ] || user_keys_json='{}'
  shadow_json="$(cmd_keys < "$config_path" | jq -R -s 'split("\n") | map(select(length > 0))')"
  [ -n "$shadow_json" ] || shadow_json='[]'
fi
keys_json="$(jq -nc --argjson d "$default_keys_json" --argjson u "$user_keys_json" '$d * $u')"
prefix_key="$(printf '%s' "$keys_json" | jq -r '.prefix // "ctrl+b"')"

# --- Native actions -------------------------------------------------------
# id | title | config key name (for the shortcut column) | extra search keywords | preview hint
#
# The id is appended to the searchable keywords automatically, so it stays
# typeable without cluttering the visible row.
STATIC_ACTIONS=(
  "new_tab|New tab|new_tab|create window|herdr tab create --focus"
  "new_tab_named|New tab (named)…|:|create window label prompt|herdr tab create --label <name> --focus"
  "rename_tab|Rename tab|rename_tab|label title|herdr tab rename <tab> <name>"
  "close_tab|Close tab|close_tab|kill remove quit delete|herdr tab close <tab>"
  "split_vertical|Split pane right (vertical)|split_vertical|vsplit beside column new|herdr pane split <pane> --direction right --focus"
  "split_horizontal|Split pane down (horizontal)|split_horizontal|hsplit below row new|herdr pane split <pane> --direction down --focus"
  "zoom_pane|Toggle zoom (fullscreen pane)|zoom|maximize fullscreen big toggle|herdr pane zoom <pane> --toggle"
  "close_pane|Close pane|close_pane|kill remove quit delete|herdr pane close <pane>"
  "rename_pane|Rename pane|rename_pane|label title|herdr pane rename <pane> <name>"
  "focus_left|Focus pane left|focus_pane_left|go move navigate h|herdr pane focus --direction left --pane <pane>"
  "focus_right|Focus pane right|focus_pane_right|go move navigate l|herdr pane focus --direction right --pane <pane>"
  "focus_up|Focus pane up|focus_pane_up|go move navigate k|herdr pane focus --direction up --pane <pane>"
  "focus_down|Focus pane down|focus_pane_down|go move navigate j|herdr pane focus --direction down --pane <pane>"
  "resize_left|Resize pane left|resize_mode|grow shrink wider narrower border|herdr pane resize --direction left --pane <pane>"
  "resize_right|Resize pane right|resize_mode|grow shrink wider narrower border|herdr pane resize --direction right --pane <pane>"
  "resize_up|Resize pane up|resize_mode|grow shrink taller shorter border|herdr pane resize --direction up --pane <pane>"
  "resize_down|Resize pane down|resize_mode|grow shrink taller shorter border|herdr pane resize --direction down --pane <pane>"
  "swap_left|Swap pane with the one left|:|exchange switch rotate reorder|herdr pane swap --direction left --pane <pane>"
  "swap_right|Swap pane with the one right|:|exchange switch rotate reorder|herdr pane swap --direction right --pane <pane>"
  "swap_up|Swap pane with the one above|:|exchange switch rotate reorder|herdr pane swap --direction up --pane <pane>"
  "swap_down|Swap pane with the one below|:|exchange switch rotate reorder|herdr pane swap --direction down --pane <pane>"
  "move_pane_tab|Move pane to tab…|:|send relocate join merge existing|herdr pane move <pane> --tab <tab> --split right --focus"
  "move_pane_workspace|Move pane to workspace…|:|send relocate project existing|herdr pane move <pane> --new-tab --workspace <workspace> --focus"
  "move_pane_new_tab|Move pane out to a new tab|:|send relocate extract break out|herdr pane move <pane> --new-tab --focus"
  "move_pane_new_workspace|Move pane out to a new workspace|:|send relocate extract break out|herdr pane move <pane> --new-workspace --focus"
  "start_agent|Start an agent in a new split…|:|claude codex gemini ai launch spawn run new|herdr pane split + herdr agent start <name> --kind <kind>"
  "prompt_agent|Send a prompt to an agent…|:|ask message text tell claude ai|herdr agent prompt <agent> <text>"
  "interrupt_agent|Interrupt an agent (esc)…|:|stop cancel escape abort key|herdr agent send-keys <agent> esc"
  "rename_agent|Rename an agent…|:|label name target|herdr agent rename <agent> <name>"
  "rename_pane_agent|Rename pane and agent…|:|label name title current together|herdr pane rename <pane> <name> + herdr agent rename <pane> <name>"
  "new_workspace|New workspace|new_workspace|create project|herdr workspace create --focus"
  "new_workspace_here|New workspace here (named)…|:|create project cwd label prompt directory|herdr workspace create --cwd <cwd> --label <name> --focus"
  "rename_workspace|Rename workspace|rename_workspace|label title project|herdr workspace rename <workspace> <name>"
  "close_workspace|Close workspace|close_workspace|kill remove quit delete project|herdr workspace close <workspace>"
  "new_worktree|New worktree here|new_worktree|git branch checkout create|herdr worktree create --workspace <workspace> --focus"
  "new_worktree_branch|New worktree on a branch…|:|git checkout create base prompt|herdr worktree create --branch <name> [--base <ref>] --focus"
  "remove_worktree|Remove this worktree checkout|remove_worktree|git delete prune rm|herdr worktree remove --workspace <workspace>"
  "reload_config|Reload herdr config|reload_config|settings keys keybindings toml refresh|herdr server reload-config"
)

actions_json="$(
  printf '%s\n' "${STATIC_ACTIONS[@]}" | jq -R -s '
    split("\n") | map(select(length > 0))
    | map(split("|") as $f | {
        id: $f[0], title: $f[1],
        key: (if $f[2] == ":" then "" else $f[2] end),
        keywords: $f[3], hint: $f[4]
      } )
    | to_entries | map(.value + {rank: .key})
  '
)"

# --- Installed plugin actions --------------------------------------------
plugin_actions_json="$(
  "$herdr_bin" plugin action list 2>/dev/null \
    | jq -c --arg self "$self_plugin" '
        [ .result.actions[]?
          | select(.plugin_id != $self)
          | (.plugin_id + "." + .action_id) as $qid
          | select($qid != "sunznx.herdr-move.open")
          | select($qid != "sunznx.herdr-move.tab")
          | select($qid != "sunznx.herdr-rename.open")
          | {
              usage: ("plugin:" + $qid),
              kind: "plugin",
              payload: $qid,
              title: ("Plugin: " + .title),
              key: "",
              keywords: ("plugin action " + $qid),
              hint: ("herdr plugin action invoke " + $qid)
            }
        ]
        | to_entries
        | map(.value + {rank: (1000 + .key)})
      ' 2>/dev/null
)"
[ -n "$plugin_actions_json" ] || plugin_actions_json='[]'

# --- Live jump targets ----------------------------------------------------
list_json() { # $1 = jq path into .result, rest = herdr args
  local path="$1"; shift
  local out
  out="$("$herdr_bin" "$@" 2>/dev/null | jq -c "$path" 2>/dev/null)"
  [ -n "$out" ] && printf '%s' "$out" || printf '[]'
}

tabs_json="$(list_json '.result.tabs // []' tab list --workspace "$workspace")"
all_tabs_json="$(list_json '.result.tabs // []' tab list)"
workspaces_json="$(list_json '.result.workspaces // []' workspace list)"
agents_json="$(list_json '.result.agents // []' agent list)"
if [ -n "$cwd" ]; then
  worktrees_json="$(list_json '.result.worktrees // []' worktree list --cwd "$cwd")"
else
  worktrees_json='[]'
fi

cols="$(tput cols 2>/dev/null || echo 80)"
case "$cols" in ''|*[!0-9]*) cols=80 ;; esac

lines="$(
  jq -nr \
    --argjson actions "$actions_json" \
    --argjson plugin_actions "$plugin_actions_json" \
    --argjson counts "$counts_json" \
    --argjson keys "$keys_json" \
    --argjson shadow "$shadow_json" \
    --argjson tabs "$tabs_json" \
    --argjson workspaces "$workspaces_json" \
    --argjson agents "$agents_json" \
    --argjson worktrees "$worktrees_json" \
    --arg prefix "$prefix_key" \
    --arg curtab "$tab" \
    --arg curws "$workspace" \
    --arg curpane "$pane" \
    --argjson cols "$cols" '

    # Resolve a config key name to the keystroke the user actually types.
    # $index fills in indexed "…1..9" bindings (switch_tab and friends).
    def shortcut($name; $index):
      ($keys[$name] // "") as $b
      | if ($name | length) == 0 or ($b | length) == 0 then ""
        else (if $index == null then $b else ($b | sub("1\\.\\.9$"; ($index | tostring))) end) as $bb
        | if ($bb | test("1\\.\\.9$")) then ""              # indexed, but no index to fill
          elif ($shadow | index($bb)) then ""               # stolen by a [[keys.command]]
          elif ($bb | startswith("prefix+")) then ($prefix + " " + ($bb | ltrimstr("prefix+")))
          else $bb
          end
        end;

    def idx($n): if ($n | type) == "number" and $n >= 1 and $n <= 9 then $n else null end;
    def trunc($n): if (. | length) > $n then (.[0:$n-1] + "…") else . end;

    ( ($actions
        | map(
            (($counts[.id] // 0) | if type == "object" then . else {count: ., last: 0} end) as $u
            | . + {
                kind: "static", payload: .id,
                key: shortcut(.key; null),
                keywords: (.keywords + " " + .id),
                count: ($u.count // 0), last: ($u.last // 0)
              })
      )
      + ($plugin_actions
        | map(
            (($counts[.usage] // 0) | if type == "object" then . else {count: ., last: 0} end) as $u
            | . + {count: ($u.count // 0), last: ($u.last // 0)})
      )
      | sort_by([-.count, -.last, .rank])
    )
    + ( $workspaces | map(select(.workspace_id != $curws))
        | map({
            kind: "goto_workspace", payload: .workspace_id,
            title: ("Go to workspace: " + (.label // .workspace_id)),
            key: shortcut("switch_workspace"; idx(.number)),
            keywords: ("switch jump goto project workspace " + (.label // "") + " " + (.agent_status // "")),
            hint: ("herdr workspace focus " + .workspace_id + "   (" + ((.tab_count // 0) | tostring) + " tabs, " + (.agent_status // "no agent") + ")")
          }) )
    + ( $tabs | map(select(.tab_id != $curtab))
        | map({
            kind: "goto_tab", payload: .tab_id,
            title: ("Go to tab: " + (.label // .tab_id)),
            key: shortcut("switch_tab"; idx(.number)),
            keywords: ("switch jump goto tab " + (.label // "") + " " + (.agent_status // "")),
            hint: ("herdr tab focus " + .tab_id + "   (" + ((.pane_count // 0) | tostring) + " panes, " + (.agent_status // "no agent") + ")")
          }) )
    # `focus_agent` is an indexed binding over the agent rows in the sidebar; we
    # assume `agent list` order matches it (it is unbound by default, so this
    # only ever shows up for someone who opted into indexed agent focus).
    + ( $agents | to_entries | map(.value + {row: (.key + 1)}) | map(select(.pane_id != $curpane))
        | map({
            kind: "focus_agent", payload: .pane_id,
            title: ("Focus agent: " + ((.terminal_title_stripped // .agent // .pane_id) | trunc(38)) + " · " + (.agent_status // "?")),
            key: shortcut("focus_agent"; idx(.row)),
            keywords: ("agent claude codex focus jump " + (.agent // "") + " " + (.agent_status // "") + " " + (.terminal_title_stripped // "") + " " + .pane_id),
            hint: ("herdr agent focus " + .pane_id + "   (" + (.agent // "?") + ", " + (.agent_status // "?") + ", " + (.cwd // "?") + ")")
          }) )
    + ( $worktrees
        | map(select((.open_workspace_id // null) == null and (.is_bare // false) == false and (.is_prunable // false) == false))
        | map({
            kind: "open_worktree", payload: .path,
            # Detached checkouts report no branch, so fall back to the directory name.
            title: ("Open worktree: " + (((if ((.branch // "") | length) > 0 then .branch else (.path | split("/") | last) end)) | trunc(34))),
            keywords: ("git worktree branch checkout open " + (.branch // "") + " " + (.path // "")),
            hint: ("herdr worktree open --path " + .path + " --focus")
          }) )

    # Row layout: "<title>  <shortcut>  <dim synonyms>".
    #
    # fzf only searches what it DISPLAYS (--nth indexes the post---with-nth
    # string, so a hidden TSV column is unreachable), which is why the search
    # synonyms have to be on the row. They go last and dimmed: fzf still
    # matches text that the window truncates, so on a narrow popup they act as
    # invisible search fodder, and on a wide one they hint what else matches.
    #
    # The title column is sized from the NATIVE action titles only — a long
    # agent/tab label should not push every shortcut across the popup. Titles
    # longer than the column are never cut, they just push their own shortcut
    # right; losing the end of a tab label to keep a column tidy is a bad trade.
    | def col($w): if length >= $w then . else . + (" " * ($w - length)) end;
      (map(select(.kind == "static") | .title | length) | max // 24) as $tw
    | (if $cols > 46 then $cols - 24 else 22 end) as $cap
    | (if $tw > $cap then $cap else $tw end) as $w
    | map([
        .kind, .payload,
        ( (.title | col($w)) + "  " + ((.key // "") | col(14)) + "  "
          + (if ((.keywords // "") | length) > 0 then "\u001b[2m" + .keywords + "\u001b[0m" else "" end)
          | sub("[[:space:]]+$"; "") ),
        (.keywords // ""), (.hint // "")
      ] | @tsv)
    | .[]
  '
)"

[ -n "$lines" ] || die "command-palette-popup: nothing to show."

if [ "${CPP_LIST_ONLY:-0}" = "1" ]; then
  printf '%s\n' "$lines"
  exit 0
fi

# fzf relevance:
#   --ansi      the row carries its dim synonym tail; fzf ignores the escapes
#               when matching, so "vsplit", "maximize", "kill" and the old
#               `close_pane`-style ids all hit the right row.
#   sorting is ON (no --no-sort): with an empty query fzf keeps input order,
#               i.e. our usage ranking, and starts scoring as soon as you type.
#   --tiebreak=begin,index  prefer matches near the start of the row (the title,
#               not the synonym tail), then fall back to our ranking instead of
#               fzf's default line-length rule.
if [ -n "${CPP_CHOICE:-}" ]; then
  # Debug: preselect a row as "<kind><TAB><payload>", skipping fzf entirely.
  choice="$(printf '%s\n' "$lines" | awk -F'\t' -v want="$CPP_CHOICE" '$1 "\t" $2 == want' | head -1)"
  [ -n "$choice" ] || die "command-palette-popup: CPP_CHOICE '${CPP_CHOICE}' matched no row."
else
choice="$(
  printf '%s\n' "$lines" \
    | fzf --delimiter=$'\t' \
          --with-nth=3 \
          --ansi \
          --prompt='herdr command ▸ ' \
          --header='↑↓ select · enter run · esc cancel' \
          --reverse \
          --cycle \
          --no-multi \
          --tiebreak=begin,index \
          --info=inline \
          --preview='printf "%s\n" {5}' \
          --preview-window='down,3,wrap,border-top'
)" || true
fi

[ -n "$choice" ] || exit 0

IFS=$'\t' read -r kind payload _display _keywords _hint <<< "$choice"

# --- Interactive helpers (we still own the TTY here) ---
ask() { # $1 = prompt label; prints the answer, non-zero if empty/aborted
  local ans
  printf '%s' "$1" >&2
  read -r ans || return 1
  [ -n "$ans" ] || return 1
  printf '%s' "$ans"
}

pick() { # stdin: "value<TAB>label" lines; $1 = prompt; prints the value
  local sel
  if [ -n "${CPP_PICK_VALUE:-}" ]; then
    cat >/dev/null
    printf '%s' "$CPP_PICK_VALUE"
    return 0
  fi
  sel="$(fzf --delimiter=$'\t' --with-nth=2 --prompt="$1" --reverse --cycle --no-multi --tiebreak=begin,index)" || return 1
  [ -n "$sel" ] || return 1
  printf '%s' "${sel%%$'\t'*}"
}

run() { # run a herdr command, die with its stderr on failure
  local resp rc
  if [ "${CPP_DRY_RUN:-0}" = "1" ]; then
    { printf 'DRY-RUN: %s' "$herdr_bin"; printf ' %q' "$@"; printf '\n'; } >&2
    return 0
  fi
  resp="$("$herdr_bin" "$@" 2>&1)"; rc=$?
  [ $rc -eq 0 ] || die "command-palette-popup: herdr $1 $2 failed (exit ${rc})
${resp}"
  printf '%s' "$resp"
}

invoke_plugin_action() {
  local action_id="$1" resp rc log_id plugin_id i entry status code err
  if [ "${CPP_DRY_RUN:-0}" = "1" ]; then
    printf 'DRY-RUN: %s plugin action invoke %q\n' "$herdr_bin" "$action_id" >&2
    return 0
  fi

  resp="$("$herdr_bin" plugin action invoke "$action_id" 2>&1)"; rc=$?
  [ $rc -eq 0 ] || die "command-palette-popup: failed to invoke ${action_id}
${resp}"

  log_id="$(printf '%s' "$resp" | jq -r '.result.log.log_id // empty' 2>/dev/null)"
  plugin_id="$(printf '%s' "$resp" | jq -r '.result.log.plugin_id // empty' 2>/dev/null)"
  [ -n "$log_id" ] && [ -n "$plugin_id" ] || return 0

  i=0
  while [ "$i" -lt 25 ]; do
    i=$((i + 1))
    entry="$(
      "$herdr_bin" plugin log list --plugin "$plugin_id" --limit 20 2>/dev/null \
        | jq -c --arg id "$log_id" '.result.logs[]? | select(.log_id == $id)' 2>/dev/null
    )"
    status="$(printf '%s' "$entry" | jq -r '.status // empty' 2>/dev/null)"
    case "$status" in
      succeeded) return 0 ;;
      failed)
        code="$(printf '%s' "$entry" | jq -r '.exit_code // "?"' 2>/dev/null)"
        err="$(printf '%s' "$entry" | jq -r '.stderr // empty' 2>/dev/null)"
        die "command-palette-popup: ${action_id} failed (exit ${code})
${err}"
        ;;
    esac
    sleep 0.2
  done
}

pick_agent() { # prints a pane_id hosting a live agent
  local list
  list="$(printf '%s' "$agents_json" | jq -r '.[] | [.pane_id, ((.terminal_title_stripped // .agent // .pane_id)) + " · " + (.agent_status // "?") + " · " + .pane_id] | @tsv')"
  [ -n "$list" ] || die "command-palette-popup: no live agents."
  printf '%s\n' "$list" | pick 'agent ▸ '
}

# --- Dispatch -------------------------------------------------------------
args=()
case "$kind" in
  goto_tab)       args=(tab focus "$payload") ;;
  goto_workspace) args=(workspace focus "$payload") ;;
  focus_agent)    args=(agent focus "$payload") ;;
  plugin)
    record_usage "plugin:$payload"
    invoke_plugin_action "$payload"
    exit 0
    ;;
  open_worktree)
    # --path identifies the checkout on its own; --cwd only passes the source
    # repo along (same role it has in `worktree create`), never a workspace to
    # reuse, so this always opens the worktree in its own workspace.
    args=(worktree open --path "$payload" --focus)
    [ -n "$cwd" ] && args+=(--cwd "$cwd")
    ;;
  static)
    record_usage "$payload"
    case "$payload" in
      new_tab|new_tab_named)
        args=(tab create --focus)
        [ -n "$workspace" ] && args+=(--workspace "$workspace")
        [ -n "$cwd" ] && args+=(--cwd "$cwd")
        if [ "$payload" = "new_tab_named" ]; then
          name="$(ask 'New tab name: ')" || exit 0
          args+=(--label "$name")
        fi
        ;;
      close_tab)
        [ -n "$tab" ] || die "command-palette-popup: no origin tab to close."
        args=(tab close "$tab")
        ;;
      rename_tab)
        [ -n "$tab" ] || die "command-palette-popup: no origin tab to rename."
        name="$(ask 'New tab name: ')" || exit 0
        args=(tab rename "$tab" "$name")
        ;;
      split_vertical|split_horizontal)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to split."
        dir=right; [ "$payload" = "split_horizontal" ] && dir=down
        args=(pane split "$pane" --direction "$dir" --focus)
        [ -n "$cwd" ] && args+=(--cwd "$cwd")
        ;;
      close_pane)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to close."
        args=(pane close "$pane")
        ;;
      zoom_pane)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to zoom."
        args=(pane zoom "$pane" --toggle)
        ;;
      rename_pane)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to rename."
        name="$(ask 'New pane name: ')" || exit 0
        args=(pane rename "$pane" "$name")
        ;;
      focus_left|focus_right|focus_up|focus_down)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to focus from."
        args=(pane focus --direction "${payload#focus_}" --pane "$pane")
        ;;
      resize_left|resize_right|resize_up|resize_down)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to resize."
        args=(pane resize --direction "${payload#resize_}" --pane "$pane")
        ;;
      swap_left|swap_right|swap_up|swap_down)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to swap."
        args=(pane swap --direction "${payload#swap_}" --pane "$pane")
        ;;
      move_pane_tab)
        [ -n "$popup_pane" ] || die "command-palette-popup: popup origin pane is unavailable; refusing to move the forwarded pane."
        source_response="$("$herdr_bin" pane get "$popup_pane" 2>&1)"; source_rc=$?
        [ $source_rc -eq 0 ] || die "command-palette-popup: origin pane '${popup_pane}' is no longer available.
${source_response}"
        source_pane="$(printf '%s' "$source_response" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
        source_tab="$(printf '%s' "$source_response" | jq -r '.result.pane.tab_id // empty' 2>/dev/null)"
        [ -n "$source_pane" ] || die "command-palette-popup: could not read the live origin pane id."
        [ -n "$source_tab" ] || die "command-palette-popup: no origin tab to move from."
        if [ -n "${HERDR_PANE_ID:-}" ] && [ "$source_pane" = "$HERDR_PANE_ID" ]; then
          die "command-palette-popup: refused to move the palette pane."
        fi
        target="$(
          jq -nr --argjson tabs "$all_tabs_json" --argjson workspaces "$workspaces_json" --arg cur "$source_tab" '
            ($workspaces | map({key: .workspace_id, value: (.label // .workspace_id)}) | from_entries) as $labels
            | $tabs[] | select(.tab_id != $cur)
            | [.tab_id, (($labels[.workspace_id] // .workspace_id) + " / #" + (.number | tostring) + " " + (.label // .tab_id))] | @tsv
          ' | pick 'move to tab ▸ '
        )" || exit 0
        [ -n "$target" ] || exit 0
        target_response="$("$herdr_bin" tab get "$target" 2>&1)"; target_rc=$?
        [ $target_rc -eq 0 ] || die "command-palette-popup: destination tab '${target}' is no longer available.
${target_response}"
        target_tab="$(printf '%s' "$target_response" | jq -r '.result.tab.tab_id // empty' 2>/dev/null)"
        [ "$target_tab" = "$target" ] || die "command-palette-popup: destination tab '${target}' is no longer available."
        args=(pane move "$source_pane" --tab "$target" --split right --focus)
        ;;
      move_pane_workspace)
        [ -n "$popup_pane" ] || die "command-palette-popup: popup origin pane is unavailable; refusing to move the forwarded pane."
        source_response="$("$herdr_bin" pane get "$popup_pane" 2>&1)"; source_rc=$?
        [ $source_rc -eq 0 ] || die "command-palette-popup: origin pane '${popup_pane}' is no longer available.
${source_response}"
        source_pane="$(printf '%s' "$source_response" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
        source_workspace="$(printf '%s' "$source_response" | jq -r '.result.pane.workspace_id // empty' 2>/dev/null)"
        [ -n "$source_pane" ] || die "command-palette-popup: could not read the live origin pane id."
        [ -n "$source_workspace" ] || die "command-palette-popup: no origin workspace to move from."
        if [ -n "${HERDR_PANE_ID:-}" ] && [ "$source_pane" = "$HERDR_PANE_ID" ]; then
          die "command-palette-popup: refused to move the palette pane; reopen it after the popup placement update."
        fi
        target="$(
          printf '%s' "$workspaces_json" | jq -r --arg cur "$source_workspace" '
            .[] | select(.workspace_id != $cur)
            | [.workspace_id, (.label // .workspace_id)] | @tsv
          ' | pick 'move to workspace ▸ '
        )" || exit 0
        [ -n "$target" ] || exit 0
        args=(pane move "$source_pane" --new-tab --workspace "$target" --focus)
        ;;
      move_pane_new_tab)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to move."
        args=(pane move "$pane" --new-tab --focus)
        ;;
      move_pane_new_workspace)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to move."
        args=(pane move "$pane" --new-workspace --focus)
        ;;
      start_agent)
        [ -n "$pane" ] || die "command-palette-popup: no origin pane to split for the agent."
        # Kinds come from the installed binary's own completion script, so the
        # list tracks herdr instead of a hardcoded copy going stale.
        kinds="$("$herdr_bin" completion zsh 2>/dev/null \
          | sed -n 's/.*--kind\[Supported agent kind[^(]*(\([^)]*\)).*/\1/p' | head -1 | tr ' ' '\n' | grep -v '^$')"
        [ -n "$kinds" ] || kinds="$(printf 'claude\ncodex\ngemini\ncursor\nopencode\ncopilot\namp\ndroid')"
        kind_choice="$(printf '%s\n' "$kinds" | awk '{print $0 "\t" $0}' | pick 'agent kind ▸ ')" || exit 0
        [ -n "$kind_choice" ] || exit 0
        name="$(ask 'Agent name (a-z0-9_-): ')" || exit 0
        case "$name" in
          [a-z]*) : ;;
          *) die "command-palette-popup: agent names must match [a-z][a-z0-9_-]{0,31} (got '${name}')." ;;
        esac
        split_args=(pane split "$pane" --direction right --focus)
        [ -n "$cwd" ] && split_args+=(--cwd "$cwd")
        if [ "${CPP_DRY_RUN:-0}" = "1" ]; then
          run "${split_args[@]}"
          args=(agent start "$name" --kind "$kind_choice" --pane "<new-pane>")
          run "${args[@]}"
          exit 0
        fi
        split_resp="$("$herdr_bin" "${split_args[@]}" 2>&1)"
        [ $? -eq 0 ] || die "command-palette-popup: pane split failed before starting the agent.
${split_resp}"
        new_pane="$(printf '%s' "$split_resp" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
        [ -n "$new_pane" ] || die "command-palette-popup: could not read the new pane id from pane split.
${split_resp}"
        args=(agent start "$name" --kind "$kind_choice" --pane "$new_pane")
        ;;
      prompt_agent)
        target="$(pick_agent)" || exit 0
        [ -n "$target" ] || exit 0
        text="$(ask 'Prompt: ')" || exit 0
        args=(agent prompt "$target" "$text")
        ;;
      interrupt_agent)
        target="$(pick_agent)" || exit 0
        [ -n "$target" ] || exit 0
        args=(agent send-keys "$target" esc)
        ;;
      rename_agent)
        target="$(pick_agent)" || exit 0
        [ -n "$target" ] || exit 0
        name="$(ask 'New agent name (a-z0-9_-): ')" || exit 0
        args=(agent rename "$target" "$name")
        ;;
      rename_pane_agent)
        [ -n "$popup_pane" ] || die "command-palette-popup: popup origin pane is unavailable; refusing to rename the forwarded pane."
        source_response="$("$herdr_bin" pane get "$popup_pane" 2>&1)"; source_rc=$?
        [ $source_rc -eq 0 ] || die "command-palette-popup: origin pane '${popup_pane}' is no longer available.
${source_response}"
        source_pane="$(printf '%s' "$source_response" | jq -r '.result.pane.pane_id // empty' 2>/dev/null)"
        [ -n "$source_pane" ] || die "command-palette-popup: could not read the live origin pane id."
        name="$(ask 'New pane and agent name: ')" || exit 0
        has_agent="$(printf '%s' "$agents_json" | jq -r --arg pane "$source_pane" 'any(.[]?; .pane_id == $pane)')"
        if [ "$has_agent" = "true" ]; then
          [[ "$name" =~ ^[a-z][a-z0-9_-]{0,31}$ ]] \
            || die "command-palette-popup: agent names must match [a-z][a-z0-9_-]{0,31}."
          run agent rename "$source_pane" "$name" >/dev/null
        fi
        args=(pane rename "$source_pane" "$name")
        ;;
      new_workspace|new_workspace_here)
        args=(workspace create --focus)
        if [ "$payload" = "new_workspace_here" ]; then
          [ -n "$cwd" ] || die "command-palette-popup: no origin cwd for the new workspace."
          name="$(ask 'New workspace name: ')" || exit 0
          args+=(--cwd "$cwd" --label "$name")
        fi
        ;;
      close_workspace)
        [ -n "$workspace" ] || die "command-palette-popup: no origin workspace to close."
        args=(workspace close "$workspace")
        ;;
      rename_workspace)
        [ -n "$workspace" ] || die "command-palette-popup: no origin workspace to rename."
        name="$(ask 'New workspace name: ')" || exit 0
        args=(workspace rename "$workspace" "$name")
        ;;
      new_worktree)
        [ -n "$workspace" ] || die "command-palette-popup: no origin workspace to create a worktree in."
        args=(worktree create --workspace "$workspace" --focus)
        ;;
      new_worktree_branch)
        [ -n "$workspace" ] || die "command-palette-popup: no origin workspace to create a worktree in."
        branch="$(ask 'Branch name: ')" || exit 0
        args=(worktree create --workspace "$workspace" --branch "$branch" --focus)
        base="$(ask 'Base ref (empty = default): ')" || base=""
        [ -n "$base" ] && args+=(--base "$base")
        ;;
      remove_worktree)
        [ -n "$workspace" ] || die "command-palette-popup: no origin workspace to remove."
        confirm="$(ask "Remove the worktree checkout for ${workspace}? [y/N] ")" || exit 0
        case "$confirm" in y|Y|yes|YES) : ;; *) exit 0 ;; esac
        args=(worktree remove --workspace "$workspace")
        ;;
      reload_config)
        args=(server reload-config)
        ;;
      *)
        die "command-palette-popup: unknown action '${payload}'."
        ;;
    esac
    ;;
  *)
    die "command-palette-popup: unrecognized selection kind '${kind}'."
    ;;
esac

[ "${#args[@]}" -gt 0 ] || exit 0
run "${args[@]}" >/dev/null
