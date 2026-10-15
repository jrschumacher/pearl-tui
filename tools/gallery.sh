#!/usr/bin/env bash
# Regenerate the README gallery (docs/images/*.png) from the kit's showcase.
# Run after any change that alters what the kit renders.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
kit="$root/skills/bubbletea-tui/kit"
out="$root/docs/images"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"
go -C "$root/skills/bubbletea-tui/tools" build -o "$tmp/" ./cmd/cellshot
cd "$kit"
render() { go run ./cmd/showcase "$@"; }

for n in charm ferrous lumen halyard moss cartridge; do
  mode=""; case $n in lumen|moss) mode="-light";; esac
  render -theme "themes/$n.json" $mode -state services -w 120 -h 40 > "$tmp/$n · services.ansi"
  render -theme "themes/$n.json" $mode -state palette  -w 120 -h 40 > "$tmp/$n · palette.ansi"
done
render -theme themes/halyard.json -state confirm -w 100 -h 30 > "$tmp/halyard · dark.ansi"
render -theme themes/halyard.json -light -state confirm -w 100 -h 30 > "$tmp/halyard · light.ansi"
render -theme themes/halyard.json -ascii -profile ascii -state confirm -w 100 -h 30 > "$tmp/halyard · ascii, no colour.ansi"
render -theme themes/charm.json -state assistant-busy -w 120 -h 40 > "$tmp/charm · assistant working.ansi"
render -theme themes/moss.json -light -state assistant -w 120 -h 40 > "$tmp/moss (light) · assistant.ansi"

cd "$tmp"
sheet() { local name=$1; shift; "$tmp/cellshot" "$@" --sheet "$out/$name" --cell 8x16 | tail -1; }
sheet gallery-services.png  {charm,ferrous,lumen,halyard,moss,cartridge}" · services.ansi"
sheet gallery-palette.png   {charm,ferrous,lumen,halyard,moss,cartridge}" · palette.ansi"
sheet gallery-modes.png     "halyard · dark.ansi" "halyard · light.ansi" "halyard · ascii, no colour.ansi"
sheet gallery-assistant.png "charm · assistant working.ansi" "moss (light) · assistant.ansi"
