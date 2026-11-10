package harness

import "testing"

var tests = []struct {
	name string
	run  func(*testing.T)
}{
	{"FrameNameIsRenderedSize", testFrameNameIsRenderedSize},
	{"Footprint", testFootprint},
	{"ASCIIFallback", testASCIIFallback},
	{"Contrast", testContrast},
	{"NoUnstyledText", testNoUnstyledText},
	{"OverlaysInReducedProfiles", testOverlaysInReducedProfiles},
	{"NoScrimAt16Colours", testNoScrimAt16Colours},
	{"DialogsStayAboveTheFooter", testDialogsStayAboveTheFooter},
	{"DialogPositionIsReadFromItsEdge", testDialogPositionIsReadFromItsEdge},
	{"ScrollAllowanceIsChecked", testScrollAllowanceIsChecked},
	{"OneFocusMarkerColumn", testOneFocusMarkerColumn},
	{"ASCIIKeepsTextRunes", testASCIIKeepsTextRunes},
	{"OverlayKeepsThePageOffItsRows", testOverlayKeepsThePageOffItsRows},
	{"OverlayFrameHasOneFocus", testOverlayFrameHasOneFocus},
	{"OverlayTablesMatchTheStates", testOverlayTablesMatchTheStates},
	{"HelpOverAnOverlayNamesItsKeys", testHelpOverAnOverlayNamesItsKeys},
	{"PressNamesKeys", testPressNamesKeys},
	{"HelpIsHandled", testHelpIsHandled},
	{"ToastKeysAreHandled", testToastKeysAreHandled},
	{"GraceWindowsHold", testGraceWindowsHold},
	{"TooSmallKeysAreTheTable", testTooSmallKeysAreTheTable},
	{"ASCIIIsASCII", testASCIIIsASCII},
	{"ASCIIStatusDistinct", testASCIIStatusDistinct},
	{"WarningInkCarriesItsGlyph", testWarningInkCarriesItsGlyph},
	{"NoTextOnItsFillAt16Colours", testNoTextOnItsFillAt16Colours},
	{"WarningIsAFillWhereItsSlotIsFaint", testWarningIsAFillWhereItsSlotIsFaint},
}
