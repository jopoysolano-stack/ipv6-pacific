package ipv4outage

import (
	"html/template"
	"net/http"
	"time"
)

// Middleware enforces monthly IPv4 outage policy before the application mux.
func Middleware(cfg Config, tmpl *template.Template, enrich PageEnricher, now func() time.Time, next http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := now()

		if OutageActive(cfg, t) && AppliesToHost(r, cfg) {
			if ok, tok := ParseRecoveryHeader(r); ok {
				LogRecovery(r, tok)
			}
		}

		if ShouldBlock(r, cfg, t) {
			token, err := NewToken()
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			until := UnavailableUntil(t)
			LogSignal(r, token)
			ServeUnavailable(w, r, tmpl, until, token, cfg.IPv6OnlySite, enrich)
			return
		}

		next.ServeHTTP(w, r)
	})
}
