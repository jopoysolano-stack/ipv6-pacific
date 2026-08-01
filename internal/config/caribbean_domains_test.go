package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCaribbeanDomainsComplete(t *testing.T) {
	root := findRepoRoot(t)
	list, err := LoadEconomies(root, "caribbean")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list.Countries {
		iso := strings.ToUpper(c.ISO2)
		p := filepath.Join(root, "config", "domains", iso+".yaml")
		df, err := LoadDomainsFile(p)
		if err != nil {
			t.Errorf("%s: %v", iso, err)
			continue
		}
		if n := len(df.Domains); n < 10 {
			t.Errorf("%s: only %d domains (want >=10)", iso, n)
		}
	}
}
