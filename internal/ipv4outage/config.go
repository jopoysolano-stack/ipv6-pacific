package ipv4outage

import (
	"log"
	"net/url"
	"os"
	"strings"
)

var defaultOutageHost = "pacific.ipv6forum.com"

// SetDefaultOutageHost sets the fallback host when IPV4_OUTAGE_HOST and PUBLIC_SITE_URL are unset.
func SetDefaultOutageHost(host string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	defaultOutageHost = host
}

// Config holds IPv4 outage policy from the environment.
type Config struct {
	OutageHost   string
	IPv6OnlySite string
	Skip         bool
	Force        bool
}

// LoadConfig reads IPV4_OUTAGE_* and PUBLIC_SITE_URL.
func LoadConfig() Config {
	cfg := Config{
		OutageHost: strings.TrimSpace(os.Getenv("IPV4_OUTAGE_HOST")),
		Skip:       strings.TrimSpace(os.Getenv("IPV4_OUTAGE_SKIP")) == "1",
		Force:      strings.TrimSpace(os.Getenv("IPV4_OUTAGE_FORCE")) == "1",
	}
	if cfg.OutageHost == "" {
		if h := hostFromPublicSiteURL(os.Getenv("PUBLIC_SITE_URL")); h != "" {
			cfg.OutageHost = h
		} else {
			cfg.OutageHost = defaultOutageHost
		}
	}
	cfg.IPv6OnlySite = ipv6OnlySiteFromProbeV6(os.Getenv("PROBE_V6_URL"), cfg.OutageHost)
	return cfg
}

func ipv6OnlySiteFromProbeV6(probeV6, outageHost string) string {
	raw := strings.TrimSpace(probeV6)
	if raw == "" {
		raw = "https://ipv6." + outageHost + "/api/healthz"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	scheme := u.Scheme
	if scheme == "" {
		scheme = "https"
	}
	host := stripPort(u.Host)
	if host == "" {
		return ""
	}
	return scheme + "://" + host + "/"
}

func hostFromPublicSiteURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	return stripPort(u.Host)
}

// WarnForceInProduction logs if IPV4_OUTAGE_FORCE is enabled.
func WarnForceInProduction(cfg Config) {
	if cfg.Force {
		log.Print("ipv4_outage: WARNING IPV4_OUTAGE_FORCE=1 — IPv4 clients may receive 503 + Retry-Over-IPv6 on the main site outside the monthly schedule")
	}
}
