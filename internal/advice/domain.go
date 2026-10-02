package advice

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

// Item is one recommendation or status note for a check.
type Item struct {
	Text string
	OK   bool // true = meets this check / affirmative
}

// CheckAdvice is advice for one measurement column.
type CheckAdvice struct {
	ID    string // dns, mail, web, dnssec, dmarc
	Title string
	Items []Item
}

// Report is the full domain advice payload for the detail page.
type Report struct {
	Priority []string
	Checks   []CheckAdvice
}

// ForDomain builds host-aware advice when Tier A/B fields exist, else class/count heuristics.
func ForDomain(d model.DomainResult) Report {
	var r Report
	dns := adviceDNS(d)
	mail := adviceMail(d)
	web := adviceWeb(d)
	dnssec := adviceDNSSEC(d)
	dmarc := adviceDMARC(d)
	r.Checks = []CheckAdvice{dns, mail, web, dnssec, dmarc}
	r.Priority = priorityFrom(dns, mail, web, dnssec, dmarc)
	return r
}

func priorityFrom(checks ...CheckAdvice) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, c := range checks {
		for _, it := range c.Items {
			if it.OK {
				continue
			}
			t := strings.TrimSpace(it.Text)
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

func adviceDNS(d model.DomainResult) CheckAdvice {
	a := CheckAdvice{ID: "dns", Title: "DNS"}
	if len(d.DNSHosts) > 0 {
		a.Items = append(a.Items, hostAdvice("name server", d.DNSHosts)...)
		if len(a.Items) == 0 && (d.DNS.Class == model.DeployDual || d.DNS.Class == model.DeployIPv6Only) {
			a.Items = append(a.Items, Item{Text: "DNS meets this check (IPv6 operational).", OK: true})
		}
		return a
	}
	a.Items = append(a.Items, serviceClassAdvice("DNS", "name servers", d.DNS)...)
	return a
}

func adviceMail(d model.DomainResult) CheckAdvice {
	a := CheckAdvice{ID: "mail", Title: "Mail"}
	if d.Mail.IntentionallyNA {
		a.Items = append(a.Items, Item{Text: "No MX records observed — mail is not applicable for scoring on this domain.", OK: true})
		return a
	}
	if len(d.MailHosts) > 0 {
		a.Items = append(a.Items, hostAdvice("mail (MX)", d.MailHosts)...)
		if len(a.Items) == 0 && (d.Mail.Class == model.DeployDual || d.Mail.Class == model.DeployIPv6Only) {
			a.Items = append(a.Items, Item{Text: "Mail meets this check (IPv6 SMTP operational).", OK: true})
		}
		return a
	}
	a.Items = append(a.Items, serviceClassAdvice("Mail", "MX hosts", d.Mail)...)
	return a
}

func adviceWeb(d model.DomainResult) CheckAdvice {
	a := CheckAdvice{ID: "web", Title: "Web"}
	if len(d.WebHosts) > 0 {
		a.Items = append(a.Items, hostAdvice("web host", d.WebHosts)...)
		if len(a.Items) == 0 && (d.Web.Class == model.DeployDual || d.Web.Class == model.DeployIPv6Only) {
			a.Items = append(a.Items, Item{Text: "Web meets this check (IPv6 HTTPS operational).", OK: true})
		}
		if len(a.Items) == 0 {
			a.Items = append(a.Items, serviceClassAdvice("Web", "web host", d.Web)...)
		}
		return a
	}
	if d.WebHost != "" {
		hostLabel := d.WebHost
		if d.Web.IPv6.Configured == 0 && d.Web.IPv4.Configured > 0 {
			a.Items = append(a.Items, Item{Text: fmt.Sprintf("Publish AAAA records for web host %s.", hostLabel)})
		}
		for _, p := range d.WebProbes {
			if p.OK {
				continue
			}
			reason := p.Error
			if reason == "" {
				reason = "error"
			}
			if p.Family == "ipv6" {
				a.Items = append(a.Items, Item{Text: fmt.Sprintf("HTTPS over IPv6 to %s failed (%s) — check firewall, routing, and the web server.", hostLabel, reason)})
			} else {
				a.Items = append(a.Items, Item{Text: fmt.Sprintf("HTTPS over IPv4 to %s failed (%s).", hostLabel, reason)})
			}
		}
		if len(a.Items) == 0 && (d.Web.Class == model.DeployDual || d.Web.Class == model.DeployIPv6Only) {
			a.Items = append(a.Items, Item{Text: "Web meets this check (IPv6 HTTPS operational).", OK: true})
		}
		if len(a.Items) == 0 {
			a.Items = append(a.Items, serviceClassAdvice("Web", "web host", d.Web)...)
		}
		return a
	}
	a.Items = append(a.Items, serviceClassAdvice("Web", "web host", d.Web)...)
	return a
}

func adviceDNSSEC(d model.DomainResult) CheckAdvice {
	a := CheckAdvice{ID: "dnssec", Title: "DNSSEC"}
	switch d.DNSSEC.State {
	case "signed":
		a.Items = append(a.Items, Item{Text: "DNSKEY observed at the apex (simplified check). Use DNSViz for full chain analysis.", OK: true})
	case "unsigned":
		a.Items = append(a.Items, Item{Text: "Enable DNSSEC signing and publish a DNSKEY at the apex. Use DNSViz to validate the full chain."})
	case "error":
		a.Items = append(a.Items, Item{Text: "DNSKEY lookup failed — re-check DNSSEC and review the chain on DNSViz."})
	default:
		a.Items = append(a.Items, Item{Text: "DNSSEC state unknown — review the zone on DNSViz."})
	}
	return a
}

func adviceDMARC(d model.DomainResult) CheckAdvice {
	a := CheckAdvice{ID: "dmarc", Title: "DMARC"}
	switch d.DMARC.State {
	case "absent":
		a.Items = append(a.Items, Item{Text: "Publish a _dmarc TXT record (start with p=none, then tighten). Check policy on dmarcian."})
	case "error":
		a.Items = append(a.Items, Item{Text: "DMARC lookup failed — re-check _dmarc TXT (dmarcian inspector can help)."})
	case "present":
		switch d.DMARC.Policy {
		case "reject":
			a.Items = append(a.Items, Item{Text: "DMARC policy is reject — strong published policy (SPF/DKIM alignment not measured here).", OK: true})
		case "quarantine":
			a.Items = append(a.Items, Item{Text: "Tighten DMARC toward p=reject when ready. Review on dmarcian."})
		case "none":
			a.Items = append(a.Items, Item{Text: "DMARC is monitoring-only (p=none). Move toward quarantine, then reject. Review on dmarcian."})
		default:
			a.Items = append(a.Items, Item{Text: "Review DMARC policy publication on dmarcian."})
		}
	default:
		a.Items = append(a.Items, Item{Text: "Review DMARC policy publication on dmarcian."})
	}
	return a
}

func serviceClassAdvice(title, hostKind string, col model.ServiceColumn) []Item {
	switch col.Class {
	case model.DeployDual, model.DeployIPv6Only:
		return []Item{{Text: fmt.Sprintf("%s meets this check (IPv6 operational).", title), OK: true}}
	case model.DeployIPv4Only:
		if col.IPv6.Configured == 0 {
			return []Item{{Text: fmt.Sprintf("Publish AAAA records / enable IPv6 on %s.", hostKind)}}
		}
		if col.IPv6.Configured > 0 && col.IPv6.Operational == 0 {
			return []Item{{Text: fmt.Sprintf("IPv6 addresses exist for %s but are not operational — check reachability, firewall, and service configuration.", hostKind)}}
		}
		return []Item{{Text: fmt.Sprintf("Improve IPv6 readiness for %s.", hostKind)}}
	default:
		return []Item{{Text: fmt.Sprintf("%s measurement was inconclusive or unavailable.", title)}}
	}
}

func hostAdvice(kind string, hosts []model.ServiceHost) []Item {
	var items []Item
	var needAAAA []string
	for _, h := range hosts {
		if len(h.IPv4) > 0 && len(h.IPv6) == 0 {
			needAAAA = append(needAAAA, h.Host)
		}
		for _, p := range h.Probes {
			if p.OK || p.Family != "ipv6" {
				continue
			}
			reason := p.Error
			if reason == "" {
				reason = "error"
			}
			if p.IP != "" {
				items = append(items, Item{Text: fmt.Sprintf("IPv6 %s on %s %s failed (%s) — check firewall, routing, and service.", p.IP, kind, h.Host, reason)})
			} else {
				items = append(items, Item{Text: fmt.Sprintf("IPv6 probe for %s %s failed (%s) — check firewall, routing, and service.", kind, h.Host, reason)})
			}
		}
	}
	if len(needAAAA) > 0 {
		sort.Strings(needAAAA)
		items = append([]Item{{Text: fmt.Sprintf("Publish AAAA records for %s: %s.", kind, strings.Join(needAAAA, ", "))}}, items...)
	}
	return items
}

// HasHostDetail reports whether collector ≥0.4 host fields are present.
func HasHostDetail(d model.DomainResult) bool {
	return len(d.DNSHosts) > 0 || len(d.MailHosts) > 0 || len(d.WebHosts) > 0 || d.WebHost != ""
}
