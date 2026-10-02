package advice

import (
	"strings"
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestForDomain_oldJSONFallback(t *testing.T) {
	d := model.DomainResult{
		Domain: "ag.gov.fj",
		DNS:    model.ServiceColumn{Class: model.DeployIPv4Only, IPv4: model.ServiceMetrics{Configured: 2, Reachable: 2, Operational: 2}},
		Mail:   model.ServiceColumn{Class: model.DeployIPv4Only, IPv4: model.ServiceMetrics{Configured: 2}},
		Web:    model.ServiceColumn{Class: model.DeployIPv4Only, IPv4: model.ServiceMetrics{Configured: 1, Operational: 1}},
		DNSSEC: model.DNSSECColumn{State: "unsigned"},
		DMARC:  model.DMARCColumn{State: "absent"},
	}
	r := ForDomain(d)
	if len(r.Priority) == 0 {
		t.Fatal("expected priority steps for ipv4-only unsigned domain")
	}
	if HasHostDetail(d) {
		t.Fatal("old JSON should not report host detail")
	}
}

func TestForDomain_hostAwareAAAA(t *testing.T) {
	d := model.DomainResult{
		Domain: "example.test",
		DNS:    model.ServiceColumn{Class: model.DeployIPv4Only, IPv4: model.ServiceMetrics{Configured: 1, Operational: 1}},
		DNSHosts: []model.ServiceHost{
			{Host: "ns1.example.test", IPv4: []string{"203.0.113.1"}},
		},
		Mail:    model.ServiceColumn{IntentionallyNA: true, Class: model.DeployUnknown},
		Web:     model.ServiceColumn{Class: model.DeployDual, IPv4: model.ServiceMetrics{Configured: 1, Operational: 1}, IPv6: model.ServiceMetrics{Configured: 1, Operational: 1}},
		WebHost: "www.example.test",
		DNSSEC:  model.DNSSECColumn{State: "signed"},
		DMARC:   model.DMARCColumn{State: "present", Policy: "reject"},
	}
	r := ForDomain(d)
	found := false
	for _, p := range r.Priority {
		if strings.Contains(p, "ns1.example.test") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected AAAA advice naming ns1; priority=%v", r.Priority)
	}
}
