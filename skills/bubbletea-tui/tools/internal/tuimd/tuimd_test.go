package tuimd

import (
	"reflect"
	"testing"
)

const doc = "# x TUI\n\n### Screens\n" +
	"#### overview — what is this machine doing\n" +
	"80×24:  header 0,0 80×1 · watch 0,1 80×1 · mem/net/disk 41,2 38×7 · list 1,10 78×13 (column header + 12 rows) · footer 0,23 80×1\n" +
	"        (outer widths; gutter 1)\n" +
	"120x40: header 0,0 120×1 · gap · list 0,14 120×25 (command 50+)\n" +
	"sketch:\n```\n80×24: fenced 0,0 1×1\n```\n" +
	"#### detail — one process\n" +
	"80×24:  header, watch line as overview · detail 1,2 78×7 · horizon 1,9\n" +
	"#### <screen> — <its job>\n80×24:  <region> <x,y w×h> · footer 0,23 80×1\n" +
	"## Decisions\n" +
	"- 2026-10-18 cuts: command, Program — a command line has no longest\n" +
	"- cuts: `title` because no bound\n" +
	"- 2026-10-18 tuilint F-TRUNC warns: cuts: is a word here, not a line\n" +
	"- 2026-10-18: cuts: a\n- 2026-10-18 — cuts: b\n- [2026-10-18] cuts: c\n- 2026-10-18 shape: cuts: d\n" +
	"- cuts: detail: name, owner — a file name has no longest\n" +
	"## Assumed decisions\n- cuts: url\n"

func TestBudgets(t *testing.T) {
	got := Budgets(doc)
	want := []Budget{
		{Screen: "overview", W: 80, H: 24, Line: 5, Regions: []Region{{"header", 0, 0, 80, 1}, {"watch", 0, 1, 80, 1}, {"mem/net/disk", 41, 2, 38, 7}, {"list", 1, 10, 78, 13}, {"footer", 0, 23, 80, 1}}},
		{Screen: "overview", W: 120, H: 40, Line: 7, Regions: []Region{{"header", 0, 0, 120, 1}, {"list", 0, 14, 120, 25}}},
		{Screen: "detail", W: 80, H: 24, Line: 13, Regions: []Region{{"detail", 1, 2, 78, 7}}},
		{Screen: "<screen>", W: 80, H: 24, Line: 15, Regions: []Region{{"footer", 0, 23, 80, 1}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Budgets:\n got %+v\nwant %+v", got, want)
	}
	if r := (Region{"list", 1, 10, 78, 13}); !r.Contains(78, 10) || r.Contains(79, 10) || r.Contains(5, 23) {
		t.Errorf("Contains is off by one")
	}
}

func TestCuts(t *testing.T) {
	got := Cuts(doc)
	want := []Cut{{Names: []string{"command", "program"}, Line: 17}, {Names: []string{"title"}, Line: 18},
		{Names: []string{"a"}, Line: 20}, {Names: []string{"b"}, Line: 21}, {Names: []string{"c"}, Line: 22}, {Names: []string{"d"}, Line: 23},
		{Screen: "detail", Names: []string{"name", "owner"}, Line: 24}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Cuts:\n got %+v\nwant %+v", got, want)
	}
}
