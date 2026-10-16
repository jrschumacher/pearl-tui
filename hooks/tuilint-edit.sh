#!/bin/sh
# PostToolUse hook of the bubbletea-tui Claude Code plugin (hooks/hooks.json).
#
# After an Edit, Write or MultiEdit of a .go file, runs `tuilint hook`, which lints the
# edited package with the `tuilint code` rules when the module is a Bubble Tea or Huh app,
# and prints P0/P1 findings to stderr with exit 2, which Claude Code shows the model.
#
# Builds the skill's tools on first use, with Setup step 4's command, into
# $CLAUDE_PLUGIN_DATA/tuitools (or $HOME/.cache/tuitools outside a plugin), and rebuilds
# them when the skill's Go sources are newer than the binary. Its own failures (no Go, a
# failed build) print one line and exit 0: the hook never blocks an edit for itself.
#
# Test: hooks/test.sh

input=$(cat)
# Not a Go file: done, before building anything. tuilint hook makes the exact decision.
case $input in *'.go"'*) ;; *) exit 0 ;; esac

root=${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}
skill=$root/skills/bubbletea-tui
cache=${TUITOOLS_CACHE:-${CLAUDE_PLUGIN_DATA:-$HOME/.cache}/tuitools}
bin=$cache/tuilint

note() {
	printf 'tuilint hook: %s; edit not linted\n' "$1" >&2
	exit 0
}

stale() {
	[ -x "$bin" ] || return 0
	[ -n "$(find "$skill/tools" "$skill/kit" \( -name '*.go' -o -name go.mod -o -name go.sum \) -newer "$bin" 2>/dev/null | head -n 1)" ]
}

if stale; then
	command -v go >/dev/null 2>&1 || note "go is not on PATH, so the tools can't be built"
	mkdir -p "$cache" || note "can't create $cache"
	lock=$cache/.build
	# a build killed by the hook timeout leaves its lock behind: reclaim it after 10 minutes
	find "$cache" -maxdepth 1 -name .build -mmin +10 -exec rm -rf {} + 2>/dev/null
	mkdir "$lock" 2>/dev/null || note "another edit is building the tools"
	trap 'rm -rf "$lock"' EXIT
	if ! go -C "$skill/tools" build -o "$lock/bin/" ./cmd/... 2>"$lock/err"; then
		note "building the tools failed: $(head -n 1 "$lock/err")"
	fi
	mv -f "$lock/bin/"* "$cache/" || note "can't install the tools into $cache"
fi

printf '%s' "$input" | "$bin" hook
status=$?
case $status in
0 | 2) exit $status ;;
esac
note "tuilint hook exited $status"
