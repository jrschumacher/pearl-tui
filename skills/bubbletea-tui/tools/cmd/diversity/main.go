// Diversity defines "different", or every brand picks the same different.
//
//	diversity themes/*.json kit/themes/ink.json kit/themes/charm.json [--max-shared 9] [--max-popular 0.5]
//	          [--baseline ink,charm --min-distance 8]
//	diversity TUI.md kit/themes/ink.json kit/themes/charm.json          # an app: its theme and its skeleton
//	diversity ../a/TUI.md ../b/TUI.md kit/themes/ink.json kit/themes/charm.json [--max-shared-skeleton 3]   # a pair eval
//	diversity -v ...                                                    # --verbose: the whole seating plan
//
// A baseline is matched by theme name, so its file must be among the themes: an explicit
// --baseline name that matches none exits 2 (the default ink,charm only notes it on
// stderr; --baseline "" turns the rule off).
//
// Tabulates every theme's visible choices (expressions + signature glyphs) and fails when:
//
//   - two themes agree on more than --max-shared of the counted axes (one design, two colours)
//   - one value is used by more than --max-popular of the themes (a house style, not a trait);
//     two-valued axes are exempt (one of two values always has a majority)
//   - a theme sits closer than --min-distance axes to a baseline (the default ink theme, the
//     Crush study): it is the default with a new palette, which is exactly the slop it replaces.
//     The failure names the axes it shares with that baseline: move the ones the brand read
//     gives a reason to move, and say the reason; an axis moved only to pass moves back
//
// Dead axes (choices nothing renders) are shown as "-" and never count: headerRule under a
// bar/stamp/minimal header, wordmark under a banner/stamp header, dialogTitle under a rules
// dialog.
//
// Apps: the skeleton. A theme file holds paint; where the regions go is composition, and
// two brands built from one spec came out one skeleton in two paints. A TUI.md among the
// inputs is an app: the theme its § Theme names (relative to the TUI.md; that file passed
// again is not a second column) and the main screen's composition, one line in § Brand
// read (fenced or not), written at brand as planned and corrected once craft renders the
// default frame:
//
//	skeleton: head <…> · status <…> · order <…> · foot <…> · verdict <…> · chart <…> · chartGlyphs <…> · turnEnd <…>
//
//	head         bar | sentence | none | folio (the name and facts close the page instead)
//	status       header | top-row | bottom-row | footer | pane | none: where session facts sit
//	order        main-first | summary-first | input-first | sidebar-left | sidebar-right | single
//	foot         keys | keys-status | status | none: what the last row holds
//	verdict      none | top | bottom | header | pane: where on the page a sentence names
//	             its state or trouble ("4.8% of checkout requests failing since 14:02")
//	chart        sparkline | trace | bars | gauge | grid | art | other: the dominant one
//	chartGlyphs  blocks | braille | shade | line | dots | ascii: the cells that carry a
//	             value; an idle floor (⣀, ·) names no family
//	turnEnd      rule | marker | space: what ends a chat turn
//
// Items split on "·", ";" or ","; "key value", "key: value" and "key=value" all read. "-"
// (or n/a) is an axis the screen doesn't have, and never counts; on chart and turnEnd
// "none" reads as "-" too, so one render gives one count, and with no chart chartGlyphs is
// "-". A skeleton line outside § Brand read is never read. The kit's own values (the
// showcase's services frame: bar, header, main-first, keys-status, bottom, sparkline,
// blocks, rule) are the "kit" column. A missing line, a missing axis, an unknown key or a
// word outside the vocabulary is a [P2] note, never a failure; the axis shows "?" and
// doesn't count. Then, with apps among the inputs:
//
//   - FAIL when two apps share more than --max-shared-skeleton (3) live skeleton axes: one
//     composition in two paints. The pair evals drew the line, with a margin of 1 each side:
//     pairs a blind critic called one product (5 shared) or siblings (4) fail; the pair it
//     called two products (2) passes. Axes move together (chart with chartGlyphs, head with
//     status and foot when an app keeps the kit's header and footer), so it is a coarse
//     count, not a measured constant; a family that shares its chrome on purpose raises it
//   - [P2] a value more than --max-popular of the apps hold (with two apps: both) that the
//     kit doesn't (no baseline or default, for an expression): newcomers converging on one
//     departure, counted over the apps alone so the kit's themes can't dilute it, and not
//     repeated for a value the house-style rule already fails
//   - advice: each app's skeleton axes that are still the kit's; its brand read's move
//     should show among the others
//
// These name an axis, never a value to move to, and are followed by two lines pointing
// back to the brand read: declare the frame as built (a line the frame contradicts is the
// critic's P1); move an axis only where TUI.md's move: or signature: gives the reason; a
// value two brands both have a reason for, the orchestrator seats and a lone build keeps;
// an axis moved only to pass moves back.
//
// --verbose (-v) adds the seating plan under the tables: every axis (skeleton included)
// with each value and who holds it, then every pair's shared axes (and an app pair's
// shared skeleton axes), not only a failing pair's.
//
// Under the table, one advisory line per brand theme names the other brand theme it is
// nearest to and how many of the axes live for both they share ("nearest to welp: moss
// (shares 11/21 axes)"): a theme can pass the baselines and still be a kit study with a
// new palette, which is worth knowing and never a failure. Pass the kit's themes beside
// yours to be told which.
//
// It is a floor, not a target. A plain brand stays plain and spends its differences on the
// two or three moves it owns; never move an axis just to pass. Learned over three rounds of
// nine brands: parallel agents given a bigger vocabulary converge on the NEWEST options, so
// after a first pass the orchestrator writes a seating plan (each contested value goes to
// the brand whose product actually has it) and lets each brand finish in its lane.
//
// Exit status: 0 OK, 1 problems, 2 usage or input errors.
package main

import (
	_ "embed"
	"os"

	"tuitools/internal/cmddoc"
	"tuitools/internal/diversity"
)

//go:embed main.go
var src string

func main() {
	os.Exit(diversity.Main(os.Args[1:], cmddoc.Text(src), os.Stdout, os.Stderr))
}
