#!/bin/sh
# End-to-end test of the plugin's PostToolUse hook: feeds hooks/tuilint-edit.sh the JSON
# Claude Code sends after an edit and checks its exit status and stderr. The first case
# builds the tools into a fresh cache, as a user's first Go edit does.
#
#   hooks/test.sh            from anywhere (CI runs it); POSIX sh, bash or dash
set -u

root=$(cd "$(dirname "$0")/.." && pwd)
hook=$root/hooks/tuilint-edit.sh
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
# the fixtures live under testdata, which the hook skips: lint copies of them
cp -R "$root/skills/bubbletea-tui/tools/internal/hook/testdata/." "$tmp/"
export TUITOOLS_CACHE="$tmp/cache" CLAUDE_PLUGIN_ROOT="$root"
shell=$(command -v sh)

# A PATH with what the hook script needs and no Go: a link to each of its utilities.
nogo=$tmp/nogo-bin
mkdir "$nogo"
for u in cat dirname find head mkdir rm mv; do
	ln -s "$(command -v "$u")" "$nogo/$u"
done

# now_ms prints a millisecond clock: date's %N where it has one, whole seconds otherwise.
now_ms() {
	t=$(date +%s%N)
	case $t in
	*[!0-9]*) echo $(($(date +%s) * 1000)) ;;
	*) echo $((t / 1000000)) ;;
	esac
}

# check NAME FILE WANT_STATUS [WANT_STDERR_SUBSTRING]: an Edit of FILE (relative to $tmp).
# HOOK_PATH and HOOK_CACHE, when set, replace PATH and the tools cache for the hook only;
# the harness keeps its own PATH.
check() {
	name=$1 file=$2 want=$3 grep=${4:-}
	start=$(now_ms)
	printf '{"session_id":"test","hook_event_name":"PostToolUse","cwd":"%s","tool_name":"Edit","tool_input":{"file_path":"%s/%s","old_string":"a","new_string":"b"},"tool_response":{"filePath":"%s/%s"}}' \
		"$tmp" "$tmp" "$file" "$tmp" "$file" |
		PATH=${HOOK_PATH:-$PATH} TUITOOLS_CACHE=${HOOK_CACHE:-$TUITOOLS_CACHE} "$shell" "$hook" 2>"$tmp/stderr"
	got=$?
	ms=$(($(now_ms) - start))
	if [ "$got" != "$want" ]; then
		echo "FAIL $name: exit $got, want $want"
	elif [ -n "$grep" ] && ! grep -qF -- "$grep" "$tmp/stderr"; then
		echo "FAIL $name: stderr lacks \"$grep\""
	elif [ -z "$grep" ] && [ -s "$tmp/stderr" ]; then
		echo "FAIL $name: want silence"
	else
		echo "ok   $name (${ms} ms)"
		return 0
	fi
	sed 's/^/     | /' "$tmp/stderr"
	return 1
}

fail=0
check "unthemed huh.NewForm reports C-HUHTHEME (first run builds the tools)" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
check "unthemed huh.NewForm again, tools cached" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
check "clean file is silent" tui/clean/clean.go 0 || fail=1
check "non-Go file is silent" tui/notes.md 0 || fail=1
check "non-TUI Go module is silent" plain/main.go 0 || fail=1
# The hook's own failures never block the edit. No Go and no tools: a note, exit 0.
HOOK_PATH=$nogo HOOK_CACHE=$tmp/empty
check "no Go toolchain: a note, exit 0" tui/form/form.go 0 "go is not on PATH" || fail=1
unset HOOK_PATH HOOK_CACHE

[ "$fail" = 0 ] && echo "hook: all cases pass"
exit $fail
