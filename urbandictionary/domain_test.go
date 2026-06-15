package urbandictionary

import (
	"testing"
)

// These tests are offline: they exercise the domain's Info() fields
// which need no network.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "urbandictionary" {
		t.Errorf("Scheme = %q, want urbandictionary", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "urbandictionary" {
		t.Errorf("Identity.Binary = %q, want urbandictionary", info.Identity.Binary)
	}
}
