package domainlookup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/pacific-monitor/pacific-monitor/internal/model"
)

const maxDomainLen = 253

// Normalize validates and lowercases a domain path segment.
// Returns empty string if invalid.
func Normalize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".")
	if name == "" || len(name) > maxDomainLen {
		return ""
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' {
			continue
		}
		return ""
	}
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.Contains(name, "..") {
		return ""
	}
	return name
}

// Found is a domain row plus parent economy metadata.
type Found struct {
	ISO2    string
	Name    string
	Domain  model.DomainResult
	Country model.CountryFile
}

// Entry is one domain for sitemap emission.
type Entry struct {
	Domain string
	ISO2   string
	ModUTC time.Time
	HasMod bool
}

// Find scans allowlisted country JSON files for a matching domain.
// allowed maps uppercase ISO2 → struct{}.
func Find(dataDir string, allowed map[string]struct{}, domain string) (Found, bool) {
	domain = Normalize(domain)
	if domain == "" || dataDir == "" {
		return Found{}, false
	}
	dir := filepath.Join(dataDir, "countries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Found{}, false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		iso := strings.ToUpper(strings.TrimSuffix(e.Name(), ".json"))
		if len(iso) != 2 {
			continue
		}
		if allowed != nil {
			if _, ok := allowed[iso]; !ok {
				continue
			}
		}
		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cf model.CountryFile
		if err := json.Unmarshal(raw, &cf); err != nil {
			continue
		}
		for _, d := range cf.Domains {
			if strings.EqualFold(d.Domain, domain) {
				name := cf.Name
				if name == "" {
					name = iso
				}
				return Found{ISO2: iso, Name: name, Domain: d, Country: cf}, true
			}
		}
	}
	return Found{}, false
}

// ListAll returns every monitored domain under allowlisted country files (sorted by domain).
func ListAll(dataDir string, allowed map[string]struct{}) []Entry {
	dir := filepath.Join(dataDir, "countries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		iso := strings.ToUpper(strings.TrimSuffix(e.Name(), ".json"))
		if len(iso) != 2 {
			continue
		}
		if allowed != nil {
			if _, ok := allowed[iso]; !ok {
				continue
			}
		}
		path := filepath.Join(dir, e.Name())
		st, err := os.Stat(path)
		var mod time.Time
		var hasMod bool
		if err == nil {
			mod = st.ModTime().UTC()
			hasMod = true
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cf model.CountryFile
		if err := json.Unmarshal(raw, &cf); err != nil {
			continue
		}
		for _, d := range cf.Domains {
			dom := Normalize(d.Domain)
			if dom == "" {
				continue
			}
			out = append(out, Entry{Domain: dom, ISO2: iso, ModUTC: mod, HasMod: hasMod})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Domain == out[j].Domain {
			return out[i].ISO2 < out[j].ISO2
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}
