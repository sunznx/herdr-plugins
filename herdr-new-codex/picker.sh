#!/usr/bin/env bash
set -uo pipefail

herdr_bin="${HERDR_BIN_PATH:-herdr}"

fail() {
  printf '\nherdr-new-codex: %s\n' "$*" >&2
  exit 1
}

command -v jq >/dev/null 2>&1 || fail "jq is not installed or not on PATH."

workspaces="$("$herdr_bin" workspace list 2>/dev/null)" || fail "could not list workspaces."
scratch_id="$(printf '%s' "$workspaces" | jq -r 'first(.result.workspaces[]? | select(.label == "scratch") | .workspace_id) // "__scratch__"')"
candidates="$({
  printf '%s' "$workspaces" | jq -r '
    .result.workspaces[]?
    | select(.label != "scratch")
    | [.workspace_id, (.label // .workspace_id)]
    | @tsv
  '
  printf '%s\tscratch\n' "$scratch_id"
})"

if [ -n "${HERDR_NEW_CODEX_CHOICE:-}" ]; then
  destination="$HERDR_NEW_CODEX_CHOICE"
else
  command -v fzf >/dev/null 2>&1 || fail "fzf is not installed or not on PATH."
  selected="$(
    printf '%s\n' "$candidates" \
      | fzf --delimiter=$'\t' --with-nth=2 --prompt='new Codex tab ▸ ' --reverse --cycle --no-multi --no-sort
  )"
  fzf_status=$?
  case "$fzf_status" in
    0) ;;
    1|130) exit 0 ;;
    *) fail "fzf exited with status $fzf_status." ;;
  esac
  [ -n "$selected" ] || exit 0
  destination="${selected%%$'\t'*}"
fi

printf '%s\n' "$candidates" | awk -F $'\t' -v id="$destination" '$1 == id { found = 1 } END { exit !found }' \
  || fail "workspace '$destination' is unavailable."

scratch=false
if [ "$destination" = "$scratch_id" ]; then
  scratch=true
  cwd="$(mktemp -d "${TMPDIR:-/tmp}/herdr-scratch-XXXXXX")"
  if [ "$scratch_id" = "__scratch__" ]; then
    created="$("$herdr_bin" workspace create --label scratch --cwd "$cwd" --focus 2>/dev/null)" \
      || fail "could not create the scratch workspace."
  else
    created="$("$herdr_bin" tab create --workspace "$scratch_id" --cwd "$cwd" --focus 2>/dev/null)" \
      || fail "could not create a scratch tab."
  fi
else
  active_tab="$(printf '%s' "$workspaces" | jq -r --arg id "$destination" 'first(.result.workspaces[]? | select(.workspace_id == $id) | .active_tab_id) // empty')"
  panes="$("$herdr_bin" pane list --workspace "$destination" 2>/dev/null)" || fail "could not inspect workspace '$destination'."
  cwd="$(printf '%s' "$panes" | jq -r --arg tab "$active_tab" 'first(.result.panes[]? | select(.tab_id == $tab) | .cwd) // first(.result.panes[]? | .cwd) // empty')"
  [ -n "$cwd" ] && [ -d "$cwd" ] || fail "workspace '$destination' directory is unavailable."
  created="$("$herdr_bin" tab create --workspace "$destination" --cwd "$cwd" --focus 2>/dev/null)" \
    || fail "could not create a tab in workspace '$destination'."
fi

pane_id="$(printf '%s' "$created" | jq -r '.result.root_pane.pane_id // empty')"
[ -n "$pane_id" ] || fail "Herdr did not return a root pane."

if [ "$scratch" = false ]; then
  exec "$herdr_bin" pane run "$pane_id" "exec codex"
fi

"$herdr_bin" pane run "$pane_id" "exec codex" || fail "could not start Codex."
for _ in {1..100}; do
  screen="$("$herdr_bin" pane read "$pane_id" --source visible --lines 60 2>/dev/null || true)"
  case "$screen" in
    *"Do you trust the contents of this directory?"*)
      exec "$herdr_bin" pane send-keys "$pane_id" enter
      ;;
  esac
  sleep 0.1
done

fail "Codex trust prompt did not appear within 10 seconds."
