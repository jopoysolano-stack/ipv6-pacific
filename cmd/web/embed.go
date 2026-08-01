package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/pacific-monitor/pacific-monitor/internal/config"
	"github.com/pacific-monitor/pacific-monitor/internal/httpserver"
	"github.com/pacific-monitor/pacific-monitor/internal/ipv4outage"
	"github.com/pacific-monitor/pacific-monitor/internal/siteurl"
)

func enrichOutagePage(bundle *connStatusBundle, publicSiteURL string, region *config.RegionMeta) ipv4outage.PageEnricher {
	siteURL := strings.TrimRight(strings.TrimSpace(publicSiteURL), "/")
	if siteURL == "" {
		siteURL = "https://" + region.DefaultHost
	}
	return func(r *http.Request, data *ipv4outage.PageData) {
		data.Nonce = httpserver.CSPNonce(r)
		data.InlineCSS = template.CSS(bundle.css)
		data.InlineJS = template.JS(bundle.inlineJS)
		data.ConnStatusVariant = "outage"
		data.SiteURL = siteURL
	}
}

func serveEmbedConnStatus(tmpl *template.Template, bundle *connStatusBundle, w http.ResponseWriter, r *http.Request, siteURL string, region *config.RegionMeta) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if siteURL == "" {
		siteURL = siteurl.AbsoluteURL(r, "/")
		siteURL = strings.TrimSuffix(siteURL, "/")
	}
	data := map[string]any{
		"InlineCSS":         template.CSS(bundle.css),
		"InlineJS":          template.JS(bundle.inlineJS),
		"Nonce":             httpserver.CSPNonce(r),
		"ConnStatusVariant": "embed",
		"SiteURL":           siteURL,
		"SiteName":          region.SiteName,
	}
	_ = tmpl.ExecuteTemplate(w, "embed_conn_status.html", data)
}

func serveEmbedConnStatusDetails(tmpl *template.Template, bundle *connStatusBundle, w http.ResponseWriter, r *http.Request, siteURL string, region *config.RegionMeta) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	if siteURL == "" {
		siteURL = siteurl.AbsoluteURL(r, "/")
		siteURL = strings.TrimSuffix(siteURL, "/")
	}
	data := map[string]any{
		"InlineCSS":       template.CSS(bundle.css),
		"InlineDetailsJS": template.JS(bundle.inlineDetailsJS),
		"Nonce":           httpserver.CSPNonce(r),
		"SiteURL":         siteURL,
		"SiteName":        region.SiteName,
	}
	_ = tmpl.ExecuteTemplate(w, "embed_conn_status_details.html", data)
}

func serveEmbedScript(w http.ResponseWriter, bundle *connStatusBundle) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(bundle.embedScript)
}

func embedPage(tmpl *template.Template, w http.ResponseWriter, r *http.Request, region *config.RegionMeta) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	borderClass := "border--ipv4"
	if httpserver.IsIPv6Client(r) {
		borderClass = "border--ipv6"
	}
	probeV4, probeV6, probeDS := probeURLsFromEnv()
	origin := strings.TrimSuffix(siteurl.AbsoluteURL(r, "/"), "/")
	iframeSnippet := fmt.Sprintf(`<iframe
  src="%s/embed/conn-status"
  title="Your IPv6 connection status"
  width="185" height="48"
  style="border:0;overflow:hidden"
  loading="lazy"
></iframe>`, origin)
	scriptSnippet := fmt.Sprintf(`<div id="ipv6-conn-status"></div>
<link rel="stylesheet" href="%s/static/css/conn-status-embed.css">
<script async src="%s/embed/conn-status.js"></script>`, origin, origin)

	scriptCSPOrigins := []string{origin}

	pageTitle := "Embed — " + region.SiteName
	metaDesc := fmt.Sprintf("Embed the %s connection-status widget on your site — iframe or script tag.", region.SiteName)
	data := map[string]any{
		"Title":             pageTitle,
		"SiteName":          region.SiteName,
		"BorderClass":       borderClass,
		"FooterVariant":     "about",
		"ProbeV4":           probeV4,
		"ProbeV6":           probeV6,
		"ProbeDS":           probeDS,
		"ShowDualProbe":     probeV4 != "" && probeV6 != "",
		"Nonce":             httpserver.CSPNonce(r),
		"IframeSnippet":     iframeSnippet,
		"ScriptSnippet":     scriptSnippet,
		"ScriptCSPOrigins":  scriptCSPOrigins,
	}
	seoMerge(r, data, pageTitle, metaDesc, region.SiteName)
	mergeOutagePageData(data)
	_ = tmpl.ExecuteTemplate(w, "embed.html", data)
}
