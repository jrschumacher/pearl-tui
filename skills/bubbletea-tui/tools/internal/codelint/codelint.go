// Package codelint is a stub until the port lands.
package codelint

import (
	"fmt"
	"io"
)

// Main runs the mode with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "codelint: not implemented yet")
	return 2
}
