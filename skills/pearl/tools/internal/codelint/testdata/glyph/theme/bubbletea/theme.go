// Package theme is one framework's theme in the kit's layout (theme/<framework>, the
// kit's tuikit/theme/bubbletea): the package is still named theme, its path ends in the
// framework. Its glyph literals are the theme's, and its Ellipsis is the bare constant
// C-ELLIPSIS fails outside the ui package, as the theme/ package's is.
package theme

// Ellipsis is the kit's default ellipsis, for a cut made with no theme at hand.
const Ellipsis = "…"

// Sep is a glyph the theme package may spell.
const Sep = "·"
