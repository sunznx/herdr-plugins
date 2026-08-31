#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
runtime_bin="$tmp/runtime-bin"
mkdir "$runtime_bin"
ln -s /bin/bash "$runtime_bin/bash"
ln -s /bin/cat "$runtime_bin/cat"

cat > "$tmp/herdr" <<'FAKE'
#!/usr/bin/env bash
set -u
printf '%s\n' "$*" >> "$HERDR_COPY_TEST_HERDR_LOG"
case "$1 $2" in
  "pane read")
    cat "$HERDR_COPY_TEST_FIXTURE"
    ;;
  "feedback show")
    printf 'feedback\n' >> "$HERDR_COPY_TEST_EVENT_LOG"
    [ "${HERDR_COPY_TEST_FEEDBACK_FAIL:-0}" != 1 ]
    ;;
  *)
    exit 1
    ;;
esac
FAKE

cat > "$tmp/clipboard" <<'FAKE'
#!/usr/bin/env bash
set -u
printf 'clipboard\n' >> "$HERDR_COPY_TEST_EVENT_LOG"
printf 'called\n' >> "$HERDR_COPY_TEST_CLIPBOARD_LOG"
[ "${HERDR_COPY_TEST_CLIPBOARD_FAIL:-0}" != 1 ] || exit 1
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
export HERDR_COPY_TEST_EVENT_LOG="$tmp/events.log"

assert_clipboard() {
  expected="$1"
  actual="$(cat "$HERDR_COPY_TEST_CLIPBOARD")"
  [ "$actual" = "$expected" ] || {
    printf 'expected clipboard <%s>, got <%s>\n' "$expected" "$actual" >&2
    return 1
  }
}

run_action() {
  expected_pane_id="${2:-w1:p2}"
  : > "$HERDR_COPY_TEST_HERDR_LOG"
  : > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
  : > "$HERDR_COPY_TEST_EVENT_LOG"
  PATH="$runtime_bin" "$script_dir/copy.sh" "$1"
  grep -Fq "pane read $expected_pane_id --source recent-unwrapped --lines 10000 --format text" "$HERDR_COPY_TEST_HERDR_LOG"
  grep -Fxq "feedback show Copied --position center" "$HERDR_COPY_TEST_HERDR_LOG"
  [ "$(cat "$HERDR_COPY_TEST_EVENT_LOG")" = $'clipboard\nfeedback' ]
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

export HERDR_COPY_TEST_FEEDBACK_FAIL=1
run_action output
assert_clipboard 'hello'
unset HERDR_COPY_TEST_FEEDBACK_FAIL

printf 'unchanged' > "$HERDR_COPY_TEST_CLIPBOARD"
: > "$HERDR_COPY_TEST_HERDR_LOG"
: > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
: > "$HERDR_COPY_TEST_EVENT_LOG"
export HERDR_COPY_TEST_CLIPBOARD_FAIL=1
if "$script_dir/copy.sh" output >/dev/null 2>&1; then
  printf 'expected clipboard failure to fail the action\n' >&2
  exit 1
fi
unset HERDR_COPY_TEST_CLIPBOARD_FAIL
assert_clipboard 'unchanged'
grep -Fxq "pane read w1:p2 --source recent-unwrapped --lines 10000 --format text" "$HERDR_COPY_TEST_HERDR_LOG"
if grep -Fq "feedback show" "$HERDR_COPY_TEST_HERDR_LOG"; then
  printf 'expected clipboard failure not to show feedback\n' >&2
  exit 1
fi
[ "$(cat "$HERDR_COPY_TEST_EVENT_LOG")" = 'clipboard' ]

export HERDR_PLUGIN_CONTEXT_JSON='{}'
export HERDR_PANE_ID='w2:p3'
run_action output 'w2:p3'
assert_clipboard 'hello'
unset HERDR_PANE_ID
export HERDR_ACTIVE_PANE_ID='w3:p4'
run_action output 'w3:p4'
assert_clipboard 'hello'
unset HERDR_ACTIVE_PANE_ID
export HERDR_PLUGIN_CONTEXT_JSON='{"focused_pane_id":"w1:p2"}'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ repo git:(main) printf hello
hello
➜ repo git:(main)
FIXTURE
run_action output
assert_clipboard 'hello'
run_action command-and-output
assert_clipboard $'printf hello\nhello'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ repo git:(main) touch dirty-file && printf dirty
dirty
➜ repo git:(main) ✗
FIXTURE
run_action output
assert_clipboard 'dirty'
run_action command-and-output
assert_clipboard $'touch dirty-file && printf dirty\ndirty'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ repo git:(main) ✗ git restore . && printf clean
clean
➜ repo git:(main)
FIXTURE
run_action output
assert_clipboard 'clean'
run_action command-and-output
assert_clipboard $'git restore . && printf clean\nclean'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ old-dir git:(main) cd ../new-dir && printf moved
moved
➜ new-dir git:(main)
FIXTURE
run_action output
assert_clipboard 'moved'
run_action command-and-output
assert_clipboard $'cd ../new-dir && printf moved\nmoved'

cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ repo git:(main) git switch feature && printf switched
switched
➜ repo git:(feature)
FIXTURE
run_action output
assert_clipboard 'switched'
run_action command-and-output
assert_clipboard $'git switch feature && printf switched\nswitched'

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
: > "$HERDR_COPY_TEST_HERDR_LOG"
: > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
: > "$HERDR_COPY_TEST_EVENT_LOG"
cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
no recognizable prompt here
FIXTURE
if "$script_dir/copy.sh" output >/dev/null 2>&1; then
  printf 'expected an unrecognized prompt to fail\n' >&2
  exit 1
fi
assert_clipboard 'unchanged'
[ ! -s "$HERDR_COPY_TEST_CLIPBOARD_LOG" ]
if grep -Fq "feedback show" "$HERDR_COPY_TEST_HERDR_LOG"; then
  printf 'expected parse failure not to show feedback\n' >&2
  exit 1
fi
[ ! -s "$HERDR_COPY_TEST_EVENT_LOG" ]

printf 'unchanged' > "$HERDR_COPY_TEST_CLIPBOARD"
: > "$HERDR_COPY_TEST_CLIPBOARD_LOG"
cat > "$HERDR_COPY_TEST_FIXTURE" <<'FIXTURE'
➜ repo git:(main) printf old
old
➜ repo git:(main) echo pending ➜
FIXTURE
if "$script_dir/copy.sh" output >/dev/null 2>&1; then
  printf 'expected a non-empty robbyrussell prompt to fail\n' >&2
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
