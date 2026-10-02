package domainmiss

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLog_rateCap(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	l := New(time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/domain/missing.example", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "https://example.com/")

	l.Log(req, "missing.example", "pacific")
	l.Log(req, "missing.example", "pacific") // capped
	l.Log(req, "other.example", "pacific")

	out := buf.String()
	if strings.Count(out, `"domain":"missing.example"`) != 1 {
		t.Fatalf("expected one missing.example log, got:\n%s", out)
	}
	if strings.Count(out, `"domain":"other.example"`) != 1 {
		t.Fatalf("expected other.example log, got:\n%s", out)
	}
	if !strings.Contains(out, "domain_miss {") {
		t.Fatalf("expected domain_miss prefix, got:\n%s", out)
	}
}

func TestLog_emptyDomain(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	l := New(time.Minute)
	l.Log(httptest.NewRequest(http.MethodGet, "/", nil), "", "pacific")
	if buf.Len() != 0 {
		t.Fatalf("expected no log, got %q", buf.String())
	}
}
