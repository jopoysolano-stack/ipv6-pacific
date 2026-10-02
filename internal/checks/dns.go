package checks

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func checkDNS(ctx context.Context, apex string, cfg Config) (model.ServiceColumn, []model.ServiceHost, error) {
	col := model.ServiceColumn{Location: "-", Display: "[0] -/-/- [-]"}
	var hosts []model.ServiceHost

	resolver := net.Resolver{PreferGo: true}

	ctxLookup, cancel := context.WithTimeout(ctx, cfg.DNSResolveTimeout)
	defer cancel()

	c := new(dns.Client)
	c.Timeout = cfg.DNSResolveTimeout

	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(apex), dns.TypeNS)
	msg.RecursionDesired = true

	r, _, err := c.ExchangeContext(ctxLookup, msg, "8.8.8.8:53")
	if err != nil || r == nil {
		col.Display = fmt.Sprintf("[S] error: %v", err)
		col.Class = model.DeployUnknown
		return col, hosts, err
	}

	var nsHosts []string
	var locTags []string
	for _, a := range r.Answer {
		if ns, ok := a.(*dns.NS); ok {
			h := strings.TrimSuffix(strings.ToLower(ns.Ns), ".")
			nsHosts = append(nsHosts, h)
			locTags = append(locTags, classifyLocation(h, apex))
		}
	}
	if len(nsHosts) == 0 {
		col.Display = "[0] -/-/- [-]"
		col.Class = model.DeployUnknown
		col.Location = "-"
		return col, hosts, nil
	}

	col.Location = mergeLocationTags(locTags)

	v4cfg, v4reach, v4op := 0, 0, 0
	v6cfg, v6reach, v6op := 0, 0, 0

	for _, host := range uniqueStrings(nsHosts) {
		loc := classifyLocation(host, apex)
		sh := model.ServiceHost{Host: host, Location: loc}
		addrs, err := resolver.LookupIPAddr(ctxLookup, host)
		if err != nil {
			hosts = append(hosts, sh)
			continue
		}
		var v4s, v6s []net.IP
		for _, a := range addrs {
			if a.IP.To4() != nil {
				v4s = append(v4s, a.IP)
			} else if a.IP.To16() != nil && a.IP.To4() == nil {
				v6s = append(v6s, a.IP)
			}
		}
		v4s = uniqueIPs(v4s)
		v6s = uniqueIPs(v6s)

		for _, ip := range v4s {
			sh.IPv4 = append(sh.IPv4, ip.String())
		}
		for _, ip := range v6s {
			sh.IPv6 = append(sh.IPv6, ip.String())
		}

		v4cfg += len(v4s)
		v6cfg += len(v6s)

		for _, ip := range v4s {
			ok, reason := udpProbe(ctx, c, ip.String()+":53", apex, dns.TypeSOA)
			sh.Probes = append(sh.Probes, model.ProbeEndpoint{IP: ip.String(), Family: "ipv4", OK: ok, Error: reason})
			if ok {
				v4reach++
				v4op++
			}
		}
		for _, ip := range v6s {
			ok, reason := udpProbe(ctx, c, "["+ip.String()+"]:53", apex, dns.TypeSOA)
			sh.Probes = append(sh.Probes, model.ProbeEndpoint{IP: ip.String(), Family: "ipv6", OK: ok, Error: reason})
			if ok {
				v6reach++
				v6op++
			}
		}
		hosts = append(hosts, sh)
	}

	col.IPv4 = model.ServiceMetrics{Configured: v4cfg, Reachable: v4reach, Operational: v4op}
	col.IPv6 = model.ServiceMetrics{Configured: v6cfg, Reachable: v6reach, Operational: v6op}
	un := uniqueStrings(nsHosts)
	col.Display = fmt.Sprintf("[%d] %d/%d/%d [%s]", len(un), v6cfg, v6reach, v6op, col.Location)
	col.Class = classifyService(col.IPv4, col.IPv6)

	return col, hosts, nil
}

func udpProbe(ctx context.Context, c *dns.Client, serverAddr, qname string, qtype uint16) (bool, string) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(qname), qtype)
	msg.RecursionDesired = false
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	r, _, err := c.ExchangeContext(ctx2, msg, serverAddr)
	if err != nil {
		return false, classifyNetError(err)
	}
	if r == nil {
		return false, ProbeOther
	}
	if r.Rcode == dns.RcodeSuccess || r.Rcode == dns.RcodeNameError {
		return true, ""
	}
	return false, ProbeDNSError
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func uniqueIPs(in []net.IP) []net.IP {
	seen := map[string]struct{}{}
	var out []net.IP
	for _, ip := range in {
		k := ip.String()
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, ip)
	}
	return out
}

func dnsLegendExplanation() LegendCheckExplanation {
	return LegendCheckExplanation{
		ID:           "dns",
		Title:        "DNS",
		Format:       "[NS count] v6 configured/reachable/operational [location]",
		PlainMeaning: "Shows how many authoritative name servers exist, and how many IPv6 endpoints are configured, responsive, and operational.",
		Notes: []string{
			"Configured/Reachable/Operational maps to DNS server endpoints discovered from NS records.",
			"Location tags indicate hosting locality context for the tested domain.",
		},
	}
}
