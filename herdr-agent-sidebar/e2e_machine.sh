#!/bin/sh
set -eu

machine=${1:?usage: sh e2e_machine.sh local|MACHINE}
if [ "$machine" = local ]; then
  workspace_list=$(herdr workspace list)
  agent_list=$(herdr agent list)
else
  workspace_list=$(herdr --machine "$machine" workspace list)
  agent_list=$(herdr --machine "$machine" agent list)
fi
printf '%s\n' "$agent_list" | jq -e --arg machine "$machine" --argjson workspaces "$workspace_list" '
  ($workspaces.result.workspaces | map({key: .workspace_id, value: .label}) | from_entries) as $labels |
  .result.agents as $agents |
  ($agents | length > 0) and
  ([$agents[] | select(.tokens.group != null)] | length > 0) and
  all($agents[];
    .tokens.agent_index != null and
    .tokens.ws_key != null and
    (.tokens | keys | any(startswith("agent_logo_"))) and
    (if .tokens.group != null then
      ($labels[.workspace_id] // .workspace_id) as $label |
      if $machine == "local" then
        .tokens.group == $label
      else
        (.tokens.group | startswith($label + " (") and endswith(")") and (length > ($label | length) + 3))
      end
    else true end)
  )
' >/dev/null
