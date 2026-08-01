package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRegionAndEconomies(t *testing.T) {
	root := findRepoRoot(t)
	r, err := ResolveRegion(root, "pacific")
	if err != nil {
		t.Fatal(err)
	}
	if r.SiteName == "" || r.EEZSVG != "EEZ_Pacific.svg" {
		t.Fatalf("unexpected pacific meta: %+v", r)
	}
	_, err = ResolveRegion(root, "nope")
	if err == nil {
		t.Fatal("expected unknown region error")
	}
	list, err := LoadEconomies(root, "pacific")
	if err != nil || len(list.Countries) < 1 {
		t.Fatalf("LoadEconomies pacific: %v len=%d", err, len(list.Countries))
	}
}

func TestValidateNoISOOverlap(t *testing.T) {
	root := findRepoRoot(t)
	if err := ValidateNoISOOverlap(root); err != nil {
		t.Fatal(err)
	}
}

func TestRegionIDFromEnv(t *testing.T) {
	t.Setenv("REGION", "")
	if got := RegionIDFromEnv(); got != DefaultRegionID {
		t.Fatalf("got %q", got)
	}
	t.Setenv("REGION", "Caribbean")
	if got := RegionIDFromEnv(); got != "caribbean" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultDataDir(t *testing.T) {
	if got := DefaultDataDir("pacific"); got != filepath.Join("data", "pacific") {
		t.Fatalf("got %q", got)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
