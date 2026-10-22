package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

// cloudEndpoint is the hosted Quayside Cloud API.
const cloudEndpoint = "https://api.quayside.example/v2"

// Profile is what `quayside setup` writes.
type Profile struct {
	Name     string
	Endpoint string
	Auth     string // "browser" or "token"
	Token    string
	Output   string // table, json, yaml
	Color    string // auto, always, never
	Org      string
}

// Service talks to the Quayside API. Everything here is a fake: there is no
// network access anywhere in this program.
type Service interface {
	Check(endpoint string) error
	DeviceCode() (code, verifyURL string)
	Connect(p Profile) error
}

// Store holds the profiles on disk. Also a fake.
type Store interface {
	Profiles() []Profile
	Default() string
	Save(p Profile) (path string, err error)
}

var (
	errUnreachable = errors.New("unreachable")
	errRefused     = errors.New("refused")
)

// fakeService pretends to be the API. Set QUAYSIDE_FAKE_FAIL=check or
// QUAYSIDE_FAKE_FAIL=connect to force a failure; an endpoint whose host
// contains "unreachable" always fails the check.
type fakeService struct {
	checkDelay   time.Duration
	connectDelay time.Duration
	failCheck    bool
	failConnect  bool
}

func newFakeService() *fakeService {
	fail := os.Getenv("QUAYSIDE_FAKE_FAIL")
	return &fakeService{
		checkDelay:   time.Second,
		connectDelay: 2 * time.Second,
		failCheck:    fail == "check",
		failConnect:  fail == "connect",
	}
}

func (s *fakeService) Check(endpoint string) error {
	time.Sleep(s.checkDelay)
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%s: %w", endpoint, errUnreachable)
	}
	if s.failCheck || strings.Contains(u.Host, "unreachable") {
		return fmt.Errorf("%s: %w", u.Host, errUnreachable)
	}
	return nil
}

func (s *fakeService) DeviceCode() (string, string) {
	return "KQ7M-4TXP", "https://quayside.example/device"
}

func (s *fakeService) Connect(p Profile) error {
	time.Sleep(s.connectDelay)
	if s.failConnect {
		return fmt.Errorf("%s: connection %w", p.Endpoint, errRefused)
	}
	return nil
}

// fakeStore has three existing profiles and remembers what was saved.
type fakeStore struct {
	profiles []Profile
	def      string
	saved    []Profile
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		def: "staging-eu",
		profiles: []Profile{
			{Name: "local", Endpoint: "http://localhost:8640/v2", Auth: "token", Token: "qs_local_0000", Output: "table", Color: "auto", Org: "dev"},
			{Name: "local-dev", Endpoint: "http://localhost:8640/v2", Auth: "token", Token: "qs_localdev_0000", Output: "json", Color: "always", Org: "dev"},
			{Name: "staging-eu", Endpoint: cloudEndpoint, Auth: "browser", Output: "table", Color: "auto", Org: "tidewater-eu"},
		},
	}
}

func (s *fakeStore) Profiles() []Profile { return s.profiles }
func (s *fakeStore) Default() string     { return s.def }

func (s *fakeStore) Save(p Profile) (string, error) {
	s.saved = append(s.saved, p)
	for i := range s.profiles {
		if s.profiles[i].Name == p.Name {
			s.profiles[i] = p
			return "~/.config/quayside/profiles.toml", nil
		}
	}
	s.profiles = append(s.profiles, p)
	return "~/.config/quayside/profiles.toml", nil
}
