package domainlookup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

func TestNormalize(t *testing.T) {
	if got := Normalize(" Ag.Gov.FJ. "); got != "ag.gov.fj" {
		t.Fatalf("got %q", got)
	}
	if Normalize("") != "" || Normalize("../etc/passwd") != "" || Normalize("a b.com") != "" {
		t.Fatal("expected invalid inputs to be rejected")
	}
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	countries := filepath.Join(dir, "countries")
	if err := os.MkdirAll(countries, 0o755); err != nil {
		t.Fatal(err)
	}
	cf := model.CountryFile{
		ISO2: "FJ",
		Name: "Fiji",
		Domains: []model.DomainResult{
			{Domain: "ag.gov.fj", Organization: "AG"},
		},
	}
	raw, _ := json.Marshal(cf)
	if err := os.WriteFile(filepath.Join(countries, "FJ.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]struct{}{"FJ": {}}
	got, ok := Find(dir, allowed, "ag.gov.fj")
	if !ok || got.ISO2 != "FJ" || got.Domain.Organization != "AG" {
		t.Fatalf("find failed: ok=%v got=%+v", ok, got)
	}
	if _, ok := Find(dir, allowed, "missing.example"); ok {
		t.Fatal("expected miss")
	}
	if _, ok := Find(dir, map[string]struct{}{"TO": {}}, "ag.gov.fj"); ok {
		t.Fatal("should not find outside allowlist")
	}
}
