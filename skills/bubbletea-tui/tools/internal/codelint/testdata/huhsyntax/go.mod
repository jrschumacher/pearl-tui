module example.com/huhsyntax

go 1.26.7

// No go.sum: the dependencies cannot be resolved, so C-HUHTHEME runs on syntax and the
// module's own types only.
require (
	charm.land/bubbles/v2 v2.2.1
	charm.land/huh/v2 v2.0.3
	charm.land/lipgloss/v2 v2.0.6
)
