#!/bin/sh
# PostToolUse hook of the bubbletea-tui Claude Code plugin (hooks/hooks.json).
#
# After an Edit, Write or MultiEdit of a .go file, runs `tuilint hook`, which lints the
# edited file with the `tuilint code` rules when its module is a Bubble Tea or Huh app in
# a project that opted in to the skill (a TUI.md, or a copy of the kit's theme package),
# and prints the file's P0/P1 findings to stderr with exit 2, which Claude Code shows the
# model.
#
# Exit codes. Findings are the only exit 2: tuilint's exit 2 passes through only when its
# stderr starts with "tuilint code: ", the findings header. A Go crash also exits 2, so
# any other exit 2, and any other status, is the hook's own failure and exits 0.
#
# The hook's own failures never block an edit. They exit 0, and the ones that switch
# linting off (no Go, Go too old, a failed build, a crash) print a systemMessage on
# stdout, which Claude Code shows the user, once per cause per tools cache. Passing
# notes ("another edit is building the tools") go to stderr, which Claude Code keeps in
# its debug log only.
#
# Builds the skill's tools on first use (Setup step 4's go build, into the plugin's own
# cache rather than a project's .tuitools/), into $CLAUDE_PLUGIN_DATA/tuitools (or
# $HOME/.cache/tuitools outside a plugin). Before that build, which costs a toolchain
# and module download on a cold machine, the shell does the cheap half of tuilint hook's
# gate: the edited file must sit under a TUI.md (up to the repository root) or in a
# module whose go.mod requires charm.land/bubbletea/v2 or charm.land/huh/v2, or import
# one of them itself; otherwise the edit is in some other Go project and nothing is
# built. The theme-package opt-in (a copy of the kit's theme/) is left to tuilint hook,
# which makes the exact decision once the tools exist. The cache holds, beside the
# binaries:
#   .built          the plugin root the tools were built from; its mtime is the build's
#                   start. A different root (a plugin update) or a newer Go source,
#                   go.mod or go.sum under tools/ or kit/ means rebuild.
#   .build-failed   the plugin root and the error of the last failed build; its mtime is
#                   that build's start. Until a source is newer or the root changes, the
#                   hook doesn't retry the build on every edit; it stays silent.
#   .build/         the build lock, holding the builder's PID. A lock whose PID is gone
#                   (a build killed by SIGKILL) is reclaimed at once, any lock after 10
#                   minutes.
#   .told-<cause>   the user was told about this failure; removed by the next good build.
#
# Test: hooks/test.sh

input=$(cat)
# Not a Go file: done, before building anything. tuilint hook makes the exact decision.
case $input in *'.go"'*) ;; *) exit 0 ;; esac

# tui_project FILE: FILE is in a Bubble Tea or Huh module, or under a TUI.md, up to the
# first .git. The cheap half of tuilint hook's gate, so an edit in an unrelated Go
# project never builds the tools.
tui_project() {
	d=$(dirname "$1")
	grep -q -e 'charm\.land/bubbletea/v2' -e 'charm\.land/huh/v2' "$1" 2>/dev/null && return 0
	while :; do
		[ -f "$d/TUI.md" ] && return 0
		[ -f "$d/go.mod" ] && grep -q -e 'charm\.land/bubbletea/v2' -e 'charm\.land/huh/v2' "$d/go.mod" && return 0
		[ -e "$d/.git" ] && return 1
		case $d in / | .) return 1 ;; esac
		d=$(dirname "$d")
	done
}
# The edited path, from tool_input.file_path (Claude Code sends it absolute). When the
# input holds no such path, the gate is skipped and tuilint hook decides after the build.
file=$(printf '%s' "$input" | sed -n 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\.go\)".*/\1/p' | head -n 1)
case $file in
/*) tui_project "$file" || exit 0 ;;
esac

root=${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}
skill=$root/skills/bubbletea-tui
cache=${TUITOOLS_CACHE:-${CLAUDE_PLUGIN_DATA:-$HOME/.cache}/tuitools}
bin=$cache/tuilint
built=$cache/.built
failed=$cache/.build-failed
lock=$cache/.build

# skip WHY: a passing reason not to lint this edit, for the debug log. Exits 0.
skip() {
	printf 'tuilint hook: %s; edit not linted\n' "$1" >&2
	exit 0
}

# tell CAUSE WHY: linting is off until the user fixes something. Shows the user a
# systemMessage, once per CAUSE until a good build, then exits 0.
tell() {
	if [ ! -e "$cache/.told-$1" ]; then
		msg=$(printf 'bubbletea-tui plugin: Go edits are not being linted: %s' "$2" |
			tr '\n\t\r' '   ' | tr -d '\000-\037' | sed 's/\\/\\\\/g; s/"/\\"/g')
		printf '{"systemMessage":"%s"}\n' "$msg"
		: >"$cache/.told-$1" 2>/dev/null
	fi
	skip "$2"
}

# newer STAMP: some Go source, go.mod or go.sum of the tools or the kit is newer.
newer() {
	[ -n "$(find "$skill/tools" "$skill/kit" \( -name '*.go' -o -name go.mod -o -name go.sum \) -newer "$1" 2>/dev/null | head -n 1)" ]
}

# stamped STAMP: STAMP exists, names this plugin root, and no source is newer.
stamped() {
	[ -f "$1" ] && [ "$(head -n 1 "$1")" = "$root" ] && ! newer "$1"
}

if ! { [ -x "$bin" ] && stamped "$built"; }; then
	# the last build failed and nothing changed since: the user has been told
	stamped "$failed" && exit 0
	mkdir -p "$cache" 2>/dev/null || skip "can't create $cache"
	command -v go >/dev/null 2>&1 || tell nogo "go is not on PATH, so the plugin can't build its tools. Install Go 1.26.7 or later."
	if [ -d "$lock" ]; then
		pid=$(cat "$lock/pid" 2>/dev/null)
		if [ -n "$pid" ] && ! kill -0 "$pid" 2>/dev/null; then
			rm -rf "$lock" # its builder is gone
		fi
		find "$cache" -maxdepth 1 -name .build -mmin +10 -exec rm -rf {} + 2>/dev/null
	fi
	mkdir "$lock" 2>/dev/null || skip "another edit is building the tools"
	# dash runs no EXIT trap on a signal: clean up on each, and exit as the signal would
	trap 'rm -rf "$lock"' EXIT
	trap 'rm -rf "$lock"; exit 129' HUP
	trap 'rm -rf "$lock"; exit 130' INT
	trap 'rm -rf "$lock"; exit 143' TERM
	echo $$ >"$lock/pid"
	: >"$lock/start" # sources saved during the build are newer than this, so rebuild next time
	if ! go -C "$skill/tools" build -o "$lock/bin/" ./cmd/... 2>"$lock/err"; then
		why=$(grep -v -e '^go: downloading ' -e '^#' -e 'too many errors$' "$lock/err" | sed '/^[[:space:]]*$/d' | tail -n 1)
		printf '%s\n%s\n' "$root" "${why:-go build failed}" >"$failed" && touch -r "$lock/start" "$failed"
		rm -f "$cache/.told-build" "$cache/.told-goold"
		case $why in
		*'requires go >='*) tell goold "Go is too old to build the plugin's tools ($why)" ;;
		*) tell build "building the plugin's tools failed: ${why:-go build failed}" ;;
		esac
	fi
	mv -f "$lock/bin/"* "$cache/" || skip "can't install the tools into $cache"
	printf '%s\n' "$root" >"$built" && touch -r "$lock/start" "$built"
	rm -f "$failed" "$cache"/.told-*
fi

# tuilint's stdout (a systemMessage on a crash it caught) goes straight to ours; its
# stderr is held, to check that an exit 2 is findings.
{ report=$(printf '%s' "$input" | "$bin" hook 2>&1 >&3 3>&-); status=$?; } 3>&1
case $status in
0)
	[ -z "$report" ] || printf '%s\n' "$report" >&2
	exit 0
	;;
2)
	case $report in
	"tuilint code: "*)
		printf '%s\n' "$report" >&2
		exit 2
		;;
	esac
	;;
esac
tell crash "tuilint exited $status: $(printf '%s\n' "$report" | head -n 1)"
