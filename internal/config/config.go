package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RegionMeta is one entry from config/regions.yaml.
type RegionMeta struct {
	ID                   string `yaml:"id"`
	SiteName             string `yaml:"site_name"`
	DefaultHost          string `yaml:"default_host"`
	RipestatSourceApp    string `yaml:"ripestat_sourceapp"`
	EEZSVG               string `yaml:"eez_svg"`
	EEZNotice            string `yaml:"eez_notice"`
	AboutTemplate        string `yaml:"about_template"`
	MapAriaLabel         string `yaml:"map_aria_label"`
	MetaDescription      string `yaml:"meta_description"`
	AboutMetaDescription string `yaml:"about_meta_description"`
	DevListen            string `yaml:"dev_listen"`
}

// RegionsFile wraps config/regions.yaml.
type RegionsFile struct {
	Regions []RegionMeta `yaml:"regions"`
}

// PacificCountry is one economy entry from config/{region}_iso2.yaml.
type PacificCountry struct {
	ISO2         string    `yaml:"iso2"`
	Name         string    `yaml:"name"`
	Centroid     []float64 `yaml:"centroid"`                // [lon, lat]
	ExcludeAPNIC bool      `yaml:"exclude_apnic,omitempty"` // AU/NZ style skip for Labs fetch
}

// PacificList wraps an economies YAML root (name kept for existing call sites).
type PacificList struct {
	Countries []PacificCountry `yaml:"countries"`
}

// DomainEntry is one monitored apex domain.
type DomainEntry struct {
	Domain       string `yaml:"domain"`
	Organization string `yaml:"organization,omitempty"`
	Sector       string `yaml:"sector,omitempty"`
	Regional     bool   `yaml:"regional,omitempty"`
	// WebURL is tried first for HTTPS reachability when the public site is not at apex or www.
	WebURL string `yaml:"web_url,omitempty"`
}

// DomainsFile is config/domains/{ISO}.yaml body.
type DomainsFile struct {
	Domains []DomainEntry `yaml:"domains"`
}

const DefaultRegionID = "pacific"

// RegionIDFromEnv returns REGION from the environment, or DefaultRegionID.
func RegionIDFromEnv() string {
	id := strings.ToLower(strings.TrimSpace(os.Getenv("REGION")))
	if id == "" {
		return DefaultRegionID
	}
	return id
}

// DefaultDataDir returns ./data/{regionID}.
func DefaultDataDir(regionID string) string {
	id := strings.ToLower(strings.TrimSpace(regionID))
	if id == "" {
		id = DefaultRegionID
	}
	return filepath.Join("data", id)
}

// LoadRegions reads config/regions.yaml.
func LoadRegions(dir string) (*RegionsFile, error) {
	path := filepath.Join(dir, "config", "regions.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list RegionsFile
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	for i := range list.Regions {
		list.Regions[i].ID = strings.ToLower(strings.TrimSpace(list.Regions[i].ID))
	}
	return &list, nil
}

// ResolveRegion returns metadata for id, or an error if unknown.
func ResolveRegion(dir, id string) (*RegionMeta, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		id = DefaultRegionID
	}
	list, err := LoadRegions(dir)
	if err != nil {
		return nil, err
	}
	for i := range list.Regions {
		if list.Regions[i].ID == id {
			r := list.Regions[i]
			return &r, nil
		}
	}
	return nil, fmt.Errorf("unknown REGION %q (not in config/regions.yaml)", id)
}

// KnownRegionIDs returns all registered region ids.
func KnownRegionIDs(dir string) ([]string, error) {
	list, err := LoadRegions(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(list.Regions))
	for _, r := range list.Regions {
		if r.ID != "" {
			out = append(out, r.ID)
		}
	}
	return out, nil
}

// LoadEconomies reads config/{regionID}_iso2.yaml.
func LoadEconomies(dir, regionID string) (*PacificList, error) {
	regionID = strings.ToLower(strings.TrimSpace(regionID))
	if regionID == "" {
		regionID = DefaultRegionID
	}
	path := filepath.Join(dir, "config", regionID+"_iso2.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var list PacificList
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// LoadPacific reads config/pacific_iso2.yaml (legacy helper; prefer LoadEconomies).
func LoadPacific(dir string) (*PacificList, error) {
	return LoadEconomies(dir, DefaultRegionID)
}

// ValidateNoISOOverlap ensures no ISO2 appears in more than one region's economy list.
func ValidateNoISOOverlap(dir string) error {
	list, err := LoadRegions(dir)
	if err != nil {
		return err
	}
	seen := map[string]string{} // ISO2 → region id
	for _, r := range list.Regions {
		economies, err := LoadEconomies(dir, r.ID)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("region %s: %w", r.ID, err)
		}
		for _, c := range economies.Countries {
			iso := strings.ToUpper(strings.TrimSpace(c.ISO2))
			if iso == "" {
				continue
			}
			if prev, ok := seen[iso]; ok {
				return fmt.Errorf("ISO2 %s listed in both regions %q and %q", iso, prev, r.ID)
			}
			seen[iso] = r.ID
		}
	}
	return nil
}

// LoadDomainsFile loads a single country YAML.
func LoadDomainsFile(path string) (*DomainsFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var df DomainsFile
	if err := yaml.Unmarshal(raw, &df); err != nil {
		return nil, err
	}
	for i := range df.Domains {
		df.Domains[i].Domain = strings.TrimSpace(strings.ToLower(df.Domains[i].Domain))
		df.Domains[i].WebURL = strings.TrimSpace(df.Domains[i].WebURL)
	}
	return &df, nil
}

// AllowedISO builds a map of uppercase ISO2 codes present in the economy list.
func AllowedISO(list *PacificList) map[string]struct{} {
	m := make(map[string]struct{})
	if list == nil {
		return m
	}
	for _, c := range list.Countries {
		m[strings.ToUpper(c.ISO2)] = struct{}{}
	}
	return m
}

// ValidateISO returns error if iso2 not in allowlist.
func ValidateISO(allowed map[string]struct{}, iso2 string) error {
	u := strings.ToUpper(strings.TrimSpace(iso2))
	if len(u) != 2 {
		return fmt.Errorf("invalid iso2")
	}
	if _, ok := allowed[u]; !ok {
		return fmt.Errorf("iso2 not allowed: %s", u)
	}
	return nil
}

// EEZTitleISOPath returns config/eez_title_iso_{regionID}.json.
func EEZTitleISOPath(dir, regionID string) string {
	regionID = strings.ToLower(strings.TrimSpace(regionID))
	if regionID == "" {
		regionID = DefaultRegionID
	}
	return filepath.Join(dir, "config", "eez_title_iso_"+regionID+".json")
}
