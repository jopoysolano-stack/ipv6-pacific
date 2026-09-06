package ipv4outage

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

const (
	statusIPv4Unavailable = http.StatusServiceUnavailable
	problemTypeURI        = "urn:ietf:params:problem:ipv4-unavailable"
)

// ProblemDetails is RFC 9457-shaped JSON per draft-martin-retry-over-ipv6.
type ProblemDetails struct {
	Type                 string `json:"type"`
	Title                string `json:"title"`
	Status               int    `json:"status"`
	Detail               string `json:"detail"`
	IPv4UnavailableUntil string `json:"ipv4UnavailableUntil"`
	IPv6OnlySite         string `json:"ipv6OnlySite,omitempty"`
}

// PageData is passed to ipv4-unavailable.html.
type PageData struct {
	ResumePlain       string
	AboutURL          string
	IPv6OnlySite      string
	Nonce             string
	InlineCSS         template.CSS
	InlineJS          template.JS
	ConnStatusVariant string
	SiteURL           string
	SiteName          string
	OutageToken       string
}

// PageEnricher adds conn-status bundle fields before rendering ipv4-unavailable.html.
type PageEnricher func(r *http.Request, data *PageData)

// NewProblemDetails builds the machine-readable IPv4-unavailability body.
func NewProblemDetails(until time.Time, ipv6OnlySite string) ProblemDetails {
	return ProblemDetails{
		Type:                 problemTypeURI,
		Title:                "IPv4 Unavailable",
		Status:               statusIPv4Unavailable,
		Detail:               fmt.Sprintf("IPv4 unavailable until %s.", until.UTC().Format(time.RFC3339)),
		IPv4UnavailableUntil: until.UTC().Format(time.RFC3339),
		IPv6OnlySite:         ipv6OnlySite,
	}
}

func setUnavailableHeaders(w http.ResponseWriter, until time.Time, token string) {
	w.Header().Set("Retry-Over-IPv6", "?1")
	w.Header().Set("IPv4-Unavailable-Until", until.UTC().Format(http.TimeFormat))
	if token != "" {
		w.Header().Set("Retry-Over-IPv6-Token", `"`+token+`"`)
	}
	sec := int(time.Until(until).Seconds())
	if sec < 0 {
		sec = 0
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", sec))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

// ServeUnavailable writes a draft-compliant 503 + Retry-Over-IPv6 response.
func ServeUnavailable(w http.ResponseWriter, r *http.Request, tmpl *template.Template, until time.Time, token string, ipv6OnlySite string, enrich PageEnricher) {
	setUnavailableHeaders(w, until, token)
	w.WriteHeader(statusIPv4Unavailable)

	if PrefersProblemJSON(r) {
		w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(NewProblemDetails(until, ipv6OnlySite))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if tmpl == nil {
		_, _ = w.Write([]byte(plainFallbackBody(until, ipv6OnlySite)))
		return
	}
	data := PageData{
		ResumePlain:       until.UTC().Format("2 January 2006, 00:00 UTC"),
		AboutURL:          "/about#ipv6-day-drill",
		IPv6OnlySite:      ipv6OnlySite,
		ConnStatusVariant: "outage",
		OutageToken:       token,
	}
	if enrich != nil {
		enrich(r, &data)
	}
	_ = tmpl.Execute(w, data)
}

func plainFallbackBody(until time.Time, ipv6OnlySite string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>Site not available on this connection</title></head><body>
<p>This site is not available on your current Internet connection.</p>
`)
	if ipv6OnlySite != "" {
		fmt.Fprintf(&b, `<p>If you can open this IPv6-only address in your browser, try: <a href="%s">%s</a></p>
`, template.HTMLEscapeString(ipv6OnlySite), template.HTMLEscapeString(ipv6OnlySite))
	}
	b.WriteString(`<p>The Internet is moving to a newer protocol generation called IPv6. This service is not reachable over the older generation (IPv4) on your network. You probably cannot fix this yourself.</p>
<p>Contact your Internet provider or your organization's IT help desk and say: "I cannot reach this site — it may require IPv6, but my system does not seem to work with IPv6."</p>
`)
	fmt.Fprintf(&b, `<p>If this is a planned outage, service over the older connection may resume after %s.</p>
</body></html>`, until.UTC().Format("2 January 2006, 00:00 UTC"))
	return b.String()
}
