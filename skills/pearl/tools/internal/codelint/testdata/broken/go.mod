module example.com/broken

go 1.26.7

// No go.sum: the dependencies cannot be resolved, so tuilint falls back to syntax.
require (
	charm.land/bubbletea/v2 v2.1.0
	charm.land/lipgloss/v2 v2.0.6
)
