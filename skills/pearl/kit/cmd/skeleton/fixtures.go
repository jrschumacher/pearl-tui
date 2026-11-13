package main

// item is one thing your app lists. Replace it with your domain's type, and the
// fixtures below with plainly fictional data in the shape your real data takes: the
// longest name, an empty field, every state your list can show.
type item struct {
	Name, Desc, State, Owner, Updated string
}

// fixtures are what every state renders: the -golden, -out and harness frames are
// only as honest as this data is realistic.
func fixtures() []item {
	return []item{
		{"Juniper", "Seed catalogue for the spring planting", "ok", "R. Okafor", "2h ago"},
		{"Tidewater", "Harbour schedule, imported nightly", "busy", "M. Lindqvist", "just now"},
		{"Basalt", "Quarry survey with two missing sheets", "warning", "A. Reyes", "yesterday"},
		{"Meridian", "Star chart notes, archived", "idle", "J. Park", "3 weeks ago"},
		{"Orchard", "Harvest ledger for the north rows", "ok", "S. Novak", "4d ago"},
	}
}
