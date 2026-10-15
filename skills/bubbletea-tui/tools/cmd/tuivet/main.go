// Command tuivet runs the tuilint code rules as go/analysis analyzers, so they work under
// go vet and in editors:
//
//	go vet -vettool=$(which tuivet) ./...
//	tuivet ./...                          # standalone, same analyzers
//	tuivet -tuihex.theme-pkg=brand ./...  # an extra theme package for C-HEX
//
// Each diagnostic starts with its rule ID (C-HEX, C-SLEEP, ...).
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"tuitools/internal/codelint"
)

func main() { multichecker.Main(codelint.Analyzers...) }
