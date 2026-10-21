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
	{"Refresh rate", "cycles 1s, 2s, 5s, 15s, 1m", ""},
	{"Toggle theme", "dark or light", ""},
	{"Status legend", "every mark in its role", ""},
}

var chatUser = "checkout-api is throwing 502s since the last deploy. what changed?"

var chatAnswer = "The 502s start at 14:02, two minutes after `v2.4.0` rolled out. The new version " +
	"reads `PAYMENTS_TIMEOUT` as seconds, but the config still sets it in milliseconds, so every " +
	"call to the payments provider waits 2,000 s and the gateway gives up first.\n\n" +
	"I've changed the parser to accept a unit suffix and kept the old value working. Roll back now, " +
	"or ship the fix as `v2.4.1`?"

// testLines are go test's output as it prints it, tabs and all: the kit expands a tab
// to its 8-column stop (ui.ExpandTabs), so the panel's edge holds.
var testLines = []string{
	"ok  \tcheckout-api/internal/config\t0.087s",
	"ok  \tcheckout-api/internal/payments\t0.412s",
}

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

// rstep is one step of a rollout (the rollout screen): its name, how long it took,
// and the lines it logged, each "level message" with {svc}, {ver} and {prev} filled in
// from the deploy. Log lines run long, as real ones do: the log pane scrolls sideways.
type rstep struct {
	Name, Took, At string // At: the step's start, the clock its log lines count from
	Log            []string
}

var rolloutSteps = []rstep{
	{"fetch source", "2s", "14:00:00", []string{
		"info git fetch --depth=50 origin refs/heads/main && git checkout --detach 9f3c2e1d7a4b (tag {ver}, signed by mara@payments)"}},
	{"restore cache", "4s", "14:00:02", []string{
		"info cache hit: go-build-{svc}-linux-amd64-5e1f0a 1.4 GiB restored from s3://ci-cache/go-build/{svc}/5e1f0a.tar.zst in 3.8s"}},
	{"resolve config", "1s", "14:00:06", []string{
		"info loaded config/prod.yaml + config/prod.us-east-1.yaml: 41 keys, 3 secrets from vault:kv/{svc}",
		"info PAYMENTS_TIMEOUT=2000 (from config/prod.yaml:18), PAYMENTS_RETRIES=2, CHECKOUT_MAX_INFLIGHT=512"}},
	{"go vet", "9s", "14:00:07", []string{"info go vet ./... : no issues in 214 packages"}},
	{"go build", "38s", "14:00:16", []string{
		"info go build -trimpath -ldflags=\"-s -w -X main.version={ver} -X main.commit=9f3c2e1d\" -o bin/{svc} ./cmd/{svc}",
		"info bin/{svc}: 31.2 MiB, go1.26.7 linux/amd64, buildid 7c1e9a0f4d2b"}},
	{"unit tests", "1m12s", "14:00:54", []string{
		"info ok   {svc}/internal/config     0.087s  coverage: 91.2% of statements",
		"info ok   {svc}/internal/payments   0.412s  coverage: 81.4% of statements",
		"info ok   {svc}/internal/cart       1.930s  coverage: 77.0% of statements",
		"info 1,284 tests passed in 214 packages, 0 failed, 12 skipped (integration build tag not set)"}},
	{"integration tests", "2m31s", "14:02:06", []string{
		"info started postgres:16.4, redis:7.2 and a payments stub on 127.0.0.1:18080 (docker compose -f test/compose.yaml)",
		"warn payments stub answered in 1.8s on TestCheckoutSlowProvider (budget 2s): the stub reads PAYMENTS_TIMEOUT in ms",
		"info 86 tests passed, 0 failed"}},
	{"lint", "21s", "14:04:37", []string{"info golangci-lint run --timeout 5m ./... : 0 issues (34 linters)"}},
	{"build image", "54s", "14:04:58", []string{
		"info docker buildx build --platform linux/amd64,linux/arm64 -t registry.internal/{svc}:{ver} .",
		"info image registry.internal/{svc}:{ver} 48.6 MiB, 7 layers, base distroless/static:nonroot"}},
	{"sbom", "6s", "14:05:52", []string{"info syft: 142 packages, 0 with known critical CVEs (grype db 2026-10-21)"}},
	{"push image", "9s", "14:05:58", []string{"info pushed registry.internal/{svc}:{ver} sha256:4be1c9f07d3a82e6b1f0c55e9a7d2c41"}},
	{"sign image", "3s", "14:06:07", []string{"info cosign sign --key kms://payments-release registry.internal/{svc}@sha256:4be1c9f0"}},
	{"migrate schema", "6s", "14:06:10", []string{
		"info migrate up: 0042_add_checkout_attempts_idx.sql applied to payments-db-prod in 5.1s (online, no lock held)"}},
	{"canary 1 pod", "2m0s", "14:06:16", []string{
		"info {svc}-7d9f8c6b5-x2kqp ready on {ver}; 10% of traffic for 2m against {prev} on 2 pods",
		"warn canary p99 412ms vs baseline 128ms on POST /v1/checkout (threshold 3x not crossed: 3.2x needs 5 min)"}},
	{"roll pod 2/3", "41s", "14:08:16", []string{
		"info {svc}-7d9f8c6b5-mh7rd ready on {ver}; drained {svc}-5c4b7f9d8-q8wzn ({prev}) after 30s grace"}},
	{"roll pod 3/3", "39s", "14:08:57", []string{
		"info {svc}-7d9f8c6b5-zt5vn ready on {ver}; drained {svc}-5c4b7f9d8-l2cfj ({prev}); 3/3 pods on {ver}"}},
	{"watch error rate", "5m0s", "14:09:36", []string{
		"info watching 5xx rate on {svc} for 5m; auto-rollback above 1.00% is off for prod by policy deploy/policies/prod.yaml:14",
		"warn 5xx rate 0.62% over the last minute (POST /v1/checkout 502, upstream: payments, after 30s)",
		"error 5xx rate 4.81% over the last minute, above 1.00%: 502 POST /v1/checkout, upstream payments timed out after 2000s (PAYMENTS_TIMEOUT read as seconds)",
		"error rollout held: {svc} {ver} failing; roll back to {prev} with r on Services, or ship a fix"}},
	{"promote", "", "14:14:36", nil},
	{"notify #deploys", "", "14:14:36", nil},
}
