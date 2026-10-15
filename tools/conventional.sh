#!/usr/bin/env bash
# conventional.sh: check commit subjects or a PR title against Conventional Commits 1.0.
#
#   tools/conventional.sh "feat(kit): add a tree component"              one line
#   git log --no-merges --format=%s main..HEAD | tools/conventional.sh   every line of stdin
#
# Format  type(scope)!: description     scope and ! are optional; the header is ≤ 100 chars
# Types   feat fix docs style refactor perf test build ci chore revert
# Scopes  skill kit tools scripts themes evals docs ci deps (free-form; these are the usual ones)
# fixup!/squash!/amend! subjects fail: squash them before merge (the commit-msg hook lets them by)
set -u
re='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-z0-9][a-z0-9._/-]*\))?!?: [^ ].*$'
status=0
check() {
	if [[ $1 =~ ^(fixup|squash|amend)!\  ]]; then
		printf '%s! commit, squash it before merge: %s\n' "${BASH_REMATCH[1]}" "$1"
		status=1
	elif [[ ! $1 =~ $re ]]; then
		printf 'not conventional: %s\n' "$1"
		status=1
	elif ((${#1} > 100)); then
		printf 'over 100 chars (%d): %s\n' "${#1}" "$1"
		status=1
	fi
}
if (($#)); then
	for s in "$@"; do check "$s"; done
else
	while IFS= read -r s; do [[ -n $s ]] && check "$s"; done
fi
((status)) && echo 'expected "type(scope)!: description", see AGENTS.md § Commits' >&2
exit $status
