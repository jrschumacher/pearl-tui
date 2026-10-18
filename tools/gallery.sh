#!/usr/bin/env bash
# Regenerate the README gallery (docs/images/*.png) from the kit's showcase.
# Run after any change that alters what the kit renders.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
kit="$root/skills/pearl/kit"
out="$root/docs/images"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"
go -C "$root/skills/pearl/tools" build -o "$tmp/" ./cmd/cellshot
cd "$kit"
render() { go run ./cmd/showcase "$@"; }

for n in charm ferrous lumen halyard moss cartridge terrarium; do
  mode=""; case $n in lumen|moss) mode="-light";; esac
  render -theme "themes/$n.json" $mode -state services -w 120 -h 40 > "$tmp/$n · services.ansi"
  render -theme "themes/$n.json" $mode -state palette  -w 120 -h 40 > "$tmp/$n · palette.ansi"
done
render -theme themes/halyard.json -state confirm -w 100 -h 30 > "$tmp/halyard · dark.ansi"
render -theme themes/halyard.json -light -state confirm -w 100 -h 30 > "$tmp/halyard · light.ansi"
render -theme themes/halyard.json -ascii -profile ascii -state confirm -w 100 -h 30 > "$tmp/halyard · ascii, no colour.ansi"
render -theme themes/charm.json -state assistant-busy -w 120 -h 40 > "$tmp/charm · assistant working.ansi"
render -theme themes/moss.json -light -state assistant -w 120 -h 40 > "$tmp/moss (light) · assistant.ansi"
render -theme themes/halyard.json -state setup-auth -w 120 -h 24 > "$tmp/halyard · setup, sign-in.ansi"
render -theme themes/lumen.json -light -state setup-review -w 120 -h 24 > "$tmp/lumen (light) · setup, review.ansi"
render -theme themes/ferrous.json -state setup-signin -w 120 -h 24 > "$tmp/ferrous · setup, waiting.ansi"
# reduced colour: a borderless dialog keeps an edge where its fill is gone, and the focus
# "fg" test theme (theme/testdata/quire.json) keeps its one hue for a state
render -theme themes/moss.json -profile ansi -state confirm -w 120 -h 40 > "$tmp/moss · 16 colours.ansi"
NO_COLOR=1 render -theme themes/moss.json -state palette -w 120 -h 40 > "$tmp/moss · NO_COLOR.ansi"
render -theme theme/testdata/quire.json -state deploys -w 120 -h 40 > "$tmp/quire (test) · focus fg, amber = new.ansi"
render -theme theme/testdata/quire.json -light -state confirm -w 120 -h 40 > "$tmp/quire (test, light) · dialog rules.ansi"

cd "$tmp"
sheet() { local name=$1; shift; "$tmp/cellshot" "$@" --sheet "$out/$name" --cell 8x16 | tail -1; }
sheet gallery-services.png  {charm,ferrous,lumen,halyard,moss,cartridge,terrarium}" · services.ansi"
sheet gallery-palette.png   {charm,ferrous,lumen,halyard,moss,cartridge,terrarium}" · palette.ansi"
sheet gallery-modes.png     "halyard · dark.ansi" "halyard · light.ansi" "halyard · ascii, no colour.ansi"
sheet gallery-assistant.png "charm · assistant working.ansi" "moss (light) · assistant.ansi"
sheet gallery-setup.png     "halyard · setup, sign-in.ansi" "lumen (light) · setup, review.ansi" "ferrous · setup, waiting.ansi"
sheet gallery-reduced.png   "moss · 16 colours.ansi" "moss · NO_COLOR.ansi" "quire (test) · focus fg, amber = new.ansi" "quire (test, light) · dialog rules.ansi"
