package ogmap

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

var (
	titleMu   sync.RWMutex
	titleToISO = map[string]string{}
)

// SetTitleToISO replaces the EEZ <title> → ISO2 map used by SVG mutation.
func SetTitleToISO(m map[string]string) {
	titleMu.Lock()
	defer titleMu.Unlock()
	titleToISO = m
	if titleToISO == nil {
		titleToISO = map[string]string{}
	}
}

// LoadTitleToISOFile reads a JSON object of territory title → ISO2.
func LoadTitleToISOFile(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]string{}
	}
	return m, nil
}

// ISOForTerritoryTitle returns the monitored ISO2 for an EEZ path <title> text, or "".
func ISOForTerritoryTitle(title string) string {
	key := normalizeTitle(title)
	if key == "" {
		return ""
	}
	titleMu.RLock()
	defer titleMu.RUnlock()
	return titleToISO[key]
}

func normalizeTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}
