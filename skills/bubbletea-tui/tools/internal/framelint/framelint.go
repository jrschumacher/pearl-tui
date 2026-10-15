// Package framelint is a stub until the port lands.
package framelint

import (
	"fmt"
	"io"
)

// Main runs the mode with its arguments and returns the exit status.
func Main(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "framelint: not implemented yet")
	return 2
}
