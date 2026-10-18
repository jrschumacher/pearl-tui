// Package tuitools holds the skill's Go tools: the checks an agent runs on every build
// iteration (tuilint, cellshot, palette check, diversity). They live in their own module,
// next to the kit, so an app that copies the kit never inherits their dependencies.
//
// Build them once, then run the binaries from anywhere:
//
//	go -C <skill>/tools build -o "$HOME/.cache/tuitools/" ./cmd/...
package tuitools
