#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/herdr" <<'FAKE'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$HERDR_COPY_TEST_HERDR_LOG"
cat "$HERDR_COPY_TEST_FIXTURE"
FAKE

cat > "$tmp/clipboard" <<'FAKE'
#!/usr/bin/env bash
set -u
printf 'called\n' >> "$HERDR_COPY_TEST_CLIPBOARD_LOG"
cat > "$HERDR_COPY_TEST_CLIPBOARD"
FAKE

chmod +x "$tmp/herdr" "$tmp/clipboard"
export HERDR_BIN_PATH="$tmp/herdr"
export HERDR_COPY_CLIPBOARD_BIN="$tmp/clipboard"
export HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p2"}'
export HERDR_COPY_TEST_FIXTURE="$tmp/fixture"
export HERDR_COPY_TEST_HERDR_LOG="$tmp/herdr.log"
export HERDR_COPY_TEST_CLIPBOARD_LOG="$tmp/clipboard.log"
export HERDR_COPY_TEST_CLIPBOARD="$tmp/clipboard.txt"

assert_clipboard() {
  expected="$1"
  actual="$(cat "$HERDR_COPY_TEST_CLIPBOARD")"
  [ "$actual" = "$expected" ] || {
    printf 'expected clipboard <%s>, got <%s>\n' "$expected" "$actual" >&2
    return 1
  }
}

run_action() {
  : > "$HERDR_COPY_TEST_HERDR_LOG"
  : > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
  "$script_dir/copy.sh" "$1"
  grep -Fq 'pane read w1:p2 --source recent-unwrapped --lines 10000 --format text' "$HERDR_COPY_TEST_HERDR_LOG"
}

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
$ printf hello
hello
$ 
FIXTURE
run_action output
assert_clipboard 'hello'
run_action command-and-output
assert_clipboard $'printf hello\nhello'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
> printf hello > /tmp/result
wrote result
> 
FIXTURE
run_action command-and-output
assert_clipboard $'printf hello > /tmp/result\nwrote result'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
╭─ ~/repo
╰─❯ printf hello
hello
╭─ ~/repo
╰─❯ 
FIXTURE
run_action output
assert_clipboard 'hello'
run_action command-and-output
assert_clipboard $'printf hello\nhello'

printf 'unchanged' > "$HERDR_COPY_TEST_CLIPBOARD"
: > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
no recognizable prompt here
FIXTURE
if "$script_dir/copy.sh" output >/dev/null 2>&1; then
  printf 'expected an unrecognized prompt to fail\n' >&2
  exit 1
fi
assert_clipboard 'unchanged'
[ ! -s "$HERDR_COPY_TEST_CLIPBOARD_LOG" ]

printf 'unchanged' > "$HERDR_COPY_TEST_CLIPBOARD"
: > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
$ true
$ 
FIXTURE
if "$script_dir/copy.sh" output >/dev/null 2>&1; then
  printf 'expected empty output to fail\n' >&2
  exit 1
fi
assert_clipboard 'unchanged'
[ ! -s "$HERDR_COPY_TEST_CLIPBOARD_LOG" ]

printf 'ok\n'
