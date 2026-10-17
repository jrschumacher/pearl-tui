#!/bin/sh
# End-to-end test of the plugin's PostToolUse hook: feeds hooks/tuilint-edit.sh the JSON
# Claude Code sends after an edit and checks its exit status, stdout and stderr. The
# first case builds the tools into a fresh cache, as a user's first Go edit does.
#
#   hooks/test.sh                   the hook under sh, as hooks/hooks.json runs it
#   HOOK_SHELL=dash hooks/test.sh   the hook under another shell (CI: sh, dash, bash)
#
# This script itself is POSIX sh; HOOK_SHELL picks the shell that runs the hook.
set -u

root=$(cd "$(dirname "$0")/.." && pwd)
hook=$root/hooks/tuilint-edit.sh
shell=$(command -v "${HOOK_SHELL:-sh}") || {
	echo "no ${HOOK_SHELL:-sh} on PATH"
	exit 1
}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
# the fixtures live under testdata, which the hook skips: lint copies of them
cp -R "$root/skills/bubbletea-tui/tools/internal/hook/testdata/." "$tmp/"
export TUITOOLS_CACHE="$tmp/cache" CLAUDE_PLUGIN_ROOT="$root"
realgo=$(command -v go) || {
	echo "no go on PATH"
	exit 1
}

# A PATH with what the hook script needs and no Go: a link to each of its utilities.
nogo=$tmp/nogo-bin
mkdir "$nogo"
for u in cat dirname find grep head mkdir mv rm sed tail touch tr; do
	ln -s "$(command -v "$u")" "$nogo/$u"
done
# A go that logs each call to $tmp/go-calls, then runs the real one.
logged=$tmp/logged-go
mkdir "$logged"
printf '#!/bin/sh\necho "$*" >>"%s/go-calls"\nexec "%s" "$@"\n' "$tmp" "$realgo" >"$logged/go"
# A go too old for the tools, behind a toolchain download line, as GOTOOLCHAIN=local says it.
oldgo=$tmp/old-go
mkdir "$oldgo"
printf '#!/bin/sh\necho "$*" >>"%s/go-calls"\necho "go: downloading go1.26.7 (linux/amd64)" >&2\necho "go: go.mod requires go >= 1.26.7 (running go 1.25.0; GOTOOLCHAIN=local)" >&2\nexit 1\n' "$tmp" >"$oldgo/go"
# A go whose build hangs until it's killed, like a cold build the hook timeout stops.
slowgo=$tmp/slow-go
mkdir "$slowgo"
printf '#!/bin/sh\nexec %s 1\n' "$(command -v sleep)" >"$slowgo/go"
chmod +x "$logged/go" "$oldgo/go" "$slowgo/go"
# builds counts the go builds so far (tuilint's own go list calls are logged too)
builds() { grep -c ' build ' "$tmp/go-calls" 2>/dev/null || true; }

# now_ms prints a millisecond clock: date's %N where it has one, whole seconds otherwise.
now_ms() {
	t=$(date +%s%N)
	case $t in
	*[!0-9]*) echo $(($(date +%s) * 1000)) ;;
	*) echo $((t / 1000000)) ;;
	esac
}

# check NAME FILE WANT_STATUS [WANT_STDERR [WANT_STDOUT]]: an Edit of FILE (relative to
# $tmp). Each WANT_ is a substring; an empty one means that stream must be empty.
# HOOK_PATH and HOOK_CACHE, when set, replace PATH and the tools cache for the hook only;
# the harness keeps its own PATH.
check() {
	name=$1 file=$2 want=$3 grep=${4:-} out=${5:-}
	start=$(now_ms)
	printf '{"session_id":"test","hook_event_name":"PostToolUse","cwd":"%s","tool_name":"Edit","tool_input":{"file_path":"%s/%s","old_string":"a","new_string":"b"},"tool_response":{"filePath":"%s/%s"}}' \
		"$tmp" "$tmp" "$file" "$tmp" "$file" |
		PATH=${HOOK_PATH:-$PATH} TUITOOLS_CACHE=${HOOK_CACHE:-$TUITOOLS_CACHE} "$shell" "$hook" >"$tmp/stdout" 2>"$tmp/stderr"
	got=$?
	ms=$(($(now_ms) - start))
	if [ "$got" != "$want" ]; then
		echo "FAIL $name: exit $got, want $want"
	elif [ -n "$grep" ] && ! grep -qF -- "$grep" "$tmp/stderr"; then
		echo "FAIL $name: stderr lacks \"$grep\""
	elif [ -z "$grep" ] && [ -s "$tmp/stderr" ]; then
		echo "FAIL $name: want silence on stderr"
	elif [ -n "$out" ] && ! grep -qF -- "$out" "$tmp/stdout"; then
		echo "FAIL $name: stdout lacks \"$out\""
	elif [ -z "$out" ] && [ -s "$tmp/stdout" ]; then
		echo "FAIL $name: want silence on stdout"
	else
		echo "ok   $name (${ms} ms)"
		return 0
	fi
	sed 's/^/     2| /' "$tmp/stderr"
	sed 's/^/     1| /' "$tmp/stdout"
	return 1
}

# expect NAME CONDITION...: a check on the state the hook left behind.
expect() {
	name=$1
	shift
	if "$@"; then
		echo "ok   $name"
	else
		echo "FAIL $name"
		return 1
	fi
}

fail=0
echo "hook under $shell"
check "unthemed huh.NewForm reports C-HUHTHEME (first run builds the tools)" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
check "unthemed huh.NewForm again, tools cached" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
check "clean file is silent" tui/clean/clean.go 0 || fail=1
check "non-Go file is silent" tui/notes.md 0 || fail=1
check "non-TUI Go module is silent" plain/main.go 0 || fail=1
check "Bubble Tea app without TUI.md is silent" bare/main.go 0 || fail=1

# An edit in a Go project that isn't a TUI never builds the tools: the shell's cheap gate
# (TUI.md above the file, or a go.mod/import naming Bubble Tea or Huh) runs before go.
HOOK_PATH=$logged:$PATH HOOK_CACHE=$tmp/cache-plain
n=$(builds)
check "non-TUI module: not built" plain/main.go 0 || fail=1
expect "  ...no go build ran" [ "$(builds)" = "$n" ] || fail=1
expect "  ...and the cache stays empty" [ ! -e "$tmp/cache-plain/tuilint" ] || fail=1
unset HOOK_PATH HOOK_CACHE

# The hook's own failures never block the edit. They tell the user once, on stdout.
HOOK_PATH=$nogo HOOK_CACHE=$tmp/empty
check "no Go toolchain: tells the user, exit 0" tui/form/form.go 0 "go is not on PATH" '"systemMessage":"bubbletea-tui plugin: Go edits are not being linted: go is not on PATH' || fail=1
check "no Go toolchain again: told once" tui/form/form.go 0 "go is not on PATH" || fail=1

# A build that fails says why (not "go: downloading"), and isn't retried until a source
# changes.
HOOK_PATH=$oldgo:$nogo HOOK_CACHE=$tmp/cache-old
check "Go too old: tells the user the real error" tui/form/form.go 0 "requires go >= 1.26.7" '"systemMessage":"bubbletea-tui plugin: Go edits are not being linted: Go is too old to build the plugin'"'"'s tools (go: go.mod requires go >= 1.26.7' || fail=1
expect "  ...not the toolchain download line" sh -c '! grep -q downloading "$1"' - "$tmp/stdout" || fail=1
n=$(builds)
check "failed build again: silent" tui/form/form.go 0 || fail=1
expect "  ...and not rebuilt" [ "$(builds)" = "$n" ] || fail=1

# A broken source in a copy of the plugin: a failed build, then the fix rebuilds.
plugin=$tmp/plugin
mkdir -p "$plugin/skills/bubbletea-tui"
cp -R "$root/hooks" "$plugin/"
cp -R "$root/skills/bubbletea-tui/tools" "$root/skills/bubbletea-tui/kit" "$plugin/skills/bubbletea-tui/"
printf 'package main\n\nvar broken int = "s"\n' >"$plugin/skills/bubbletea-tui/tools/cmd/tuilint/broken.go"
HOOK_PATH=$logged:$PATH HOOK_CACHE=$tmp/cache-copy
CLAUDE_PLUGIN_ROOT=$plugin
check "compile error: tells the user the file and line" tui/form/form.go 0 "broken.go:3" 'building the plugin'"'"'s tools failed: cmd/tuilint/broken.go:3' || fail=1
n=$(builds)
check "compile error again: silent" tui/form/form.go 0 || fail=1
expect "  ...and not rebuilt" [ "$(builds)" = "$n" ] || fail=1
sleep 1 # mtimes in whole seconds on some filesystems
printf 'package main\n' >"$plugin/skills/bubbletea-tui/tools/cmd/tuilint/broken.go"
check "source fixed: rebuilds and lints" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
expect "  ...and rebuilt once" [ "$(builds)" = "$((n + 1))" ] || fail=1
# The same cache, another plugin root (an update): the old tools are stale.
CLAUDE_PLUGIN_ROOT=$root
check "plugin root changed: rebuilds" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
expect "  ...once" [ "$(builds)" = "$((n + 2))" ] || fail=1
check "same root again: cached" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
expect "  ...not rebuilt" [ "$(builds)" = "$((n + 2))" ] || fail=1

# A lock left by a killed build (its PID is gone) is reclaimed at once; a live one isn't.
rm "$tmp/cache-copy/.built"
mkdir "$tmp/cache-copy/.build"
sh -c 'exit 0' &
dead=$!
wait "$dead"
echo "$dead" >"$tmp/cache-copy/.build/pid"
check "lock with a dead PID: reclaimed, builds" tui/form/form.go 2 "C-HUHTHEME tui/form/form.go:9" || fail=1
expect "  ...and the lock is gone" [ ! -e "$tmp/cache-copy/.build" ] || fail=1
rm "$tmp/cache-copy/.built"
mkdir "$tmp/cache-copy/.build"
echo $$ >"$tmp/cache-copy/.build/pid"
check "lock with a live PID: not linted, exit 0" tui/form/form.go 0 "another edit is building the tools" || fail=1
rm -rf "$tmp/cache-copy/.build"

# A build stopped by SIGTERM (the hook timeout) removes its lock.
HOOK_PATH=$slowgo:$nogo HOOK_CACHE=$tmp/cache-term
printf '{"tool_input":{"file_path":"%s/tui/form/form.go"}}' "$tmp" |
	PATH=$HOOK_PATH TUITOOLS_CACHE=$HOOK_CACHE "$shell" "$hook" >/dev/null 2>&1 &
pid=$!
i=0
while [ ! -f "$HOOK_CACHE/.build/start" ] && [ $i -lt 100 ]; do
	sleep 0.05 2>/dev/null || sleep 1
	i=$((i + 1))
done
kill -TERM "$pid"
wait "$pid"
got=$?
expect "SIGTERM mid-build: the hook exits 143 (got $got)" [ "$got" = 143 ] || fail=1
expect "  ...and removes the lock" sh -c '[ -d "$1" ] && [ ! -e "$1/.build" ]' - "$HOOK_CACHE" || fail=1

# A crash in tuilint (a Go panic also exits 2) tells the user, and never reaches the
# model as findings.
mkdir -p "$tmp/cache-crash"
printf '#!/bin/sh\necho "panic: boom" >&2\nexit 2\n' >"$tmp/cache-crash/tuilint"
chmod +x "$tmp/cache-crash/tuilint"
echo "$root" >"$tmp/cache-crash/.built"
HOOK_PATH=$PATH HOOK_CACHE=$tmp/cache-crash
check "tuilint crash: tells the user, exit 0" tui/form/form.go 0 "tuilint exited 2: panic: boom" '"systemMessage":"bubbletea-tui plugin: Go edits are not being linted: tuilint exited 2: panic: boom' || fail=1
unset HOOK_PATH HOOK_CACHE

[ "$fail" = 0 ] && echo "hook: all cases pass"
exit $fail
