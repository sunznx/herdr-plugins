#!/usr/bin/env bash
set -uo pipefail

mode="${1:-}"
herdr_bin="${HERDR_BIN_PATH:-herdr}"
clipboard_bin="${HERDR_COPY_CLIPBOARD_BIN:-/usr/bin/pbcopy}"

fail() {
  printf 'herdr-copy-last-command-output: %s\n' "$*" >&2
  exit 1
}

[ "$mode" = "output" ] || [ "$mode" = "command-and-output" ] || fail "unknown action."
command -v "$clipboard_bin" >/dev/null 2>&1 || fail "pbcopy is not installed or not on PATH."

context_json="${HERDR_PLUGIN_CONTEXT_JSON:-}"
pane_id_pattern='"focused_pane_id"[[:space:]]*:[[:space:]]*"([[:alnum:]_.:-]+)"'
pane_id=""
if [[ "$context_json" =~ $pane_id_pattern ]]; then
  pane_id="${BASH_REMATCH[1]}"
fi
if [ -z "$pane_id" ]; then
  pane_id="${HERDR_PANE_ID:-${HERDR_ACTIVE_PANE_ID:-}}"
fi
[ -n "$pane_id" ] || fail "could not resolve the triggering pane."

scrollback="$("$herdr_bin" pane read "$pane_id" --source recent-unwrapped --lines 10000 --format text)" ||
  fail "could not read the triggering pane."

if ! content="$(
  printf '%s\n' "$scrollback" |
    /usr/bin/awk -v mode="$mode" '
      function trim_left(value) {
        sub(/^[[:space:]]+/, "", value)
        return value
      }

      function trim_right(value) {
        sub(/[[:space:]]+$/, "", value)
        return value
      }

      function prompt_marker(line, trimmed, last) {
        trimmed = trim_right(line)
        if (trimmed ~ /❯$/) return "❯"
        if (trimmed ~ /➜$/) return "➜"

        last = substr(trimmed, length(trimmed), 1)
        if (last == "$" || last == "%" || last == "#" || last == ">") return last
        return ""
      }

      function command_after_prefix(line, prefix, rest) {
        if (prefix == "" || index(line, prefix) != 1) return ""

        rest = substr(line, length(prefix) + 1)
        if (rest !~ /^[[:space:]]/) return ""

        rest = trim_left(rest)
        return rest
      }

      function command_after_prompt(line, marker, remaining, offset, relative, rest) {
        remaining = line
        offset = 0

        while ((relative = index(remaining, marker)) > 0) {
          rest = substr(line, offset + relative + length(marker))
          if (rest ~ /^[[:space:]]/) {
            rest = trim_left(rest)
            if (rest != "") return rest
          }
          offset += relative + length(marker) - 1
          remaining = substr(line, offset + 1)
        }
        return ""
      }

      function is_prompt_header(line, trimmed) {
        trimmed = trim_left(line)
        return index(trimmed, "╭") == 1 || index(trimmed, "┌") == 1 || index(trimmed, "┏") == 1
      }

      { lines[++count] = $0 }

      END {
        prompt_line = count
        while (prompt_line > 0 && lines[prompt_line] ~ /^[[:space:]]*$/) prompt_line--

        prompt_prefix = trim_right(lines[prompt_line])
        command_line = 0
        command = ""
        for (line_number = prompt_line - 1; line_number > 0; line_number--) {
          command = command_after_prefix(lines[line_number], prompt_prefix)
          if (command != "") {
            command_line = line_number
            break
          }
        }

        if (command_line == 0) {
          marker = prompt_marker(lines[prompt_line])
          if (marker == "") exit 2

          for (line_number = prompt_line - 1; line_number > 0; line_number--) {
            command = command_after_prompt(lines[line_number], marker)
            if (command != "") {
              command_line = line_number
              break
            }
          }
        }
        if (command_line == 0) exit 2

        output_end = prompt_line - 1
        if (output_end > command_line && is_prompt_header(lines[output_end])) output_end--

        output_start = command_line + 1
        while (output_start <= output_end && lines[output_start] ~ /^[[:space:]]*$/) output_start++
        while (output_end >= output_start && lines[output_end] ~ /^[[:space:]]*$/) output_end--
        if (output_start > output_end) exit 3

        if (mode == "command-and-output") print command
        for (line_number = output_start; line_number <= output_end; line_number++) print lines[line_number]
      }
    '
)"; then
  fail "could not identify a completed command with non-empty output."
fi

[ -n "$content" ] || fail "could not identify a completed command with non-empty output."
printf '%s' "$content" | "$clipboard_bin" || fail "could not write to the clipboard."
