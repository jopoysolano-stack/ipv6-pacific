package domainmiss

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxLoggedUA   = 120
	defaultMinGap = time.Minute
)

// Logger emits rate-capped domain_miss journal lines for unknown /domain/{name} requests.
type Logger struct {
	minGap time.Duration
	mu     sync.Mutex
	last   map[string]time.Time
}

// Default is the process-wide logger (1 log per domain per minute).
var Default = New(defaultMinGap)

// New returns a Logger that logs at most once per domain per minGap.
func New(minGap time.Duration) *Logger {
	if minGap <= 0 {
		minGap = defaultMinGap
	}
	return &Logger{minGap: minGap, last: make(map[string]time.Time)}
}

type record struct {
	Event     string `json:"event"`
	Domain    string `json:"domain"`
	Region    string `json:"region,omitempty"`
	Path      string `json:"path,omitempty"`
	Referer   string `json:"referer,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// Log records a miss if the domain has not been logged within minGap.
// domain must already be normalized; empty domain is ignored.
func (l *Logger) Log(r *http.Request, domain, region string) {
	if l == nil || domain == "" {
		return
	}
	now := time.Now()
	l.mu.Lock()
	if t, ok := l.last[domain]; ok && now.Sub(t) < l.minGap {
		l.mu.Unlock()
		return
	}
	l.last[domain] = now
	if len(l.last) > 10_000 {
		cutoff := now.Add(-l.minGap * 2)
		for k, t := range l.last {
			if t.Before(cutoff) {
				delete(l.last, k)
			}
		}
	}
	l.mu.Unlock()

	rec := record{
		Event:     "domain_miss",
		Domain:    domain,
		Region:    strings.TrimSpace(region),
		Path:      "/domain/" + domain,
		Referer:   strings.TrimSpace(r.Referer()),
		UserAgent: truncateUA(r.UserAgent()),
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	log.Printf("domain_miss %s", string(b))
}

func truncateUA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLoggedUA {
		return s
	}
	return s[:maxLoggedUA]
}
