package main

// Fixtures: realistic content, never lorem ipsum. Design with the data the
// product will actually show, including the long name, the failing row and
// the empty list.

type service struct {
	Name, Env, Version, State string
	Prev                      string // the last healthy version (rollback target)
	P99                       []float64
	ErrRate                   string
	Owner                     string
	Deployed                  string
}

var services = []service{
	{"checkout-api", "prod", "v2.4.0", "error", "v2.3.9", []float64{120, 118, 125, 130, 128, 190, 340, 610, 580, 720, 690, 705}, "4.81%", "payments", "12m ago"},
	{"inventory-sync", "prod", "v1.19.2", "ok", "v1.19.1", []float64{40, 42, 41, 39, 44, 43, 40, 41, 42, 40, 39, 41}, "0.02%", "catalog", "3d ago"},
	{"search-indexer", "prod", "v3.0.1", "deploying", "v3.0.0", []float64{88, 90, 87, 95, 99, 91, 89, 92, 94, 90, 93, 96}, "0.10%", "discovery", "now"},
	{"auth-gateway", "prod", "v5.2.7", "ok", "v5.2.6", []float64{22, 21, 23, 22, 24, 22, 21, 23, 22, 22, 23, 21}, "0.00%", "identity", "8d ago"},
	{"notifications-worker-eu-west", "prod", "v0.9.14", "warning", "v0.9.13", []float64{60, 64, 70, 82, 95, 110, 120, 118, 121, 130, 127, 133}, "0.92%", "growth", "1d ago"},
	{"ledger", "prod", "v7.1.0", "ok", "v7.0.9", []float64{150, 149, 152, 151, 150, 148, 153, 150, 149, 151, 150, 152}, "0.01%", "payments", "21d ago"},
	{"image-resizer", "prod", "v2.0.3", "ok", "v2.0.2", []float64{210, 205, 230, 220, 215, 225, 218, 222, 219, 221, 217, 220}, "0.05%", "media", "5d ago"},
	{"recommendations", "prod", "v4.4.4", "idle", "v4.4.3", []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, "-", "discovery", "34d ago"},
}

type command struct{ Title, Desc, Key string }

var commands = []command{
	{"Deploy service", "ship a version to an environment", "d"},
	{"Roll back", "return to the previous healthy version", "r"},
	{"Tail logs", "stream logs from every pod", "l"},
	{"Open dashboard", "Grafana for the selected service", "o"},
	{"Silence alerts", "mute for 1h, 4h or until resolved", ""},
	{"Switch environment", "prod, staging or dev", "e"},
	{"Copy service URL", "", "y"},
	{"Toggle theme", "dark or light", ""},
	{"Status legend", "every mark in its role", ""},
}

var chatUser = "checkout-api is throwing 502s since the last deploy. what changed?"

var chatAnswer = "The 502s start at 14:02, two minutes after `v2.4.0` rolled out. The new version " +
	"reads `PAYMENTS_TIMEOUT` as seconds, but the config still sets it in milliseconds, so every " +
	"call to the payments provider waits 2,000 s and the gateway gives up first.\n\n" +
	"I've changed the parser to accept a unit suffix and kept the old value working. Roll back now, " +
	"or ship the fix as `v2.4.1`?"

var logLines = []string{
	"14:02:11  INFO   rollout complete  checkout-api v2.4.0  3/3 pods ready",
	"14:02:40  WARN   upstream timeout  payments.charge  after 30s",
	"14:02:41  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:02:43  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:02:44  WARN   upstream timeout  payments.charge  after 30s",
	"14:02:47  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:02:51  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:02:52  WARN   circuit half-open  payments",
	"14:02:58  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:03:02  ERROR  502 POST /v1/checkout  upstream: payments",
	"14:03:05  INFO   circuit open  payments  (12 failures in 60s)",
	"14:03:09  ERROR  502 POST /v1/checkout  upstream: payments",
}

var diffLines = []string{
	"@@ config/timeouts.go:41 @@",
	"-	d := time.Duration(n) * time.Second",
	"+	d, err := parseDuration(raw) // accepts 2000ms, 2s, 2000",
	"+	if err != nil {",
	"+		return 0, fmt.Errorf(\"PAYMENTS_TIMEOUT: %w\", err)",
	"+	}",
}

var thinkingLines = []string{
	"The deploy at 14:00 is the only change in the window.",
	"Diffing v2.3.9..v2.4.0 for config handling.",
	"timeouts.go switched from time.ParseDuration to a bare integer multiply.",
	"PAYMENTS_TIMEOUT=2000 in prod; that is 2000 seconds now.",
}

// activity: time, what, who, state
var activity = [][4]string{
	{"14:03", "circuit opened on payments for checkout-api", "auto", "error"},
	{"14:02", "502 rate crossed 1% on checkout-api", "alertmanager", "error"},
	{"14:00", "deployed checkout-api v2.4.0", "@mara", "ok"},
	{"13:52", "search-indexer v3.0.1 rolling, 2/5 pods", "@ci", "busy"},
	{"13:31", "notifications-worker-eu-west p99 above 120ms", "alertmanager", "warning"},
	{"13:10", "silenced disk-pressure on ledger for 4h", "@jun", "idle"},
	{"12:44", "deployed auth-gateway v5.2.7", "@ci", "ok"},
}

// event is one line of the deploy log. It names its service, so selecting a
// row selects that service (by identity, never by row index).
type event struct{ Ago, Svc, Env, Version, By, State string }

var initialEvents = []event{
	{"now", "search-indexer", "prod", "v3.0.1", "@ci", "busy"},
	{"14m ago", "checkout-api", "prod", "v2.4.0", "@mara", "error"},
	{"1h ago", "auth-gateway", "prod", "v5.2.7", "@ci", "ok"},
	{"1d ago", "notifications-worker-eu-west", "prod", "v0.9.14", "@devon", "warning"},
	{"2d ago", "ledger", "staging", "v7.1.1", "@jun", "ok"},
	{"3d ago", "inventory-sync", "prod", "v1.19.2", "@ci", "ok"},
	{"5d ago", "image-resizer", "prod", "v2.0.3", "@priya", "ok"},
	{"21d ago", "ledger", "prod", "v7.1.0", "@jun", "ok"},
	{"34d ago", "recommendations", "prod", "v4.4.4", "@ci", "idle"},
}
