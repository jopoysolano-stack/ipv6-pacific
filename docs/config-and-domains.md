# Configuration and monitored domains

## Layout

| Path | Role |
|------|------|
| `config/regions.yaml` | Region registry (`REGION` env). Site name, default host, EEZ asset, about template, `dev_listen`. |
| `config/{region}_iso2.yaml` | Economies for that region (map allowlist + collector schedule). |
| `config/eez_title_iso_{region}.json` | EEZ SVG `<title>` text → ISO2 (browser map + OG PNG). |
| `config/domains/{ISO2}.yaml` | Apex domains measured for that economy (scoped by active region allowlist). |

## Adding a region

1. Append an entry to `config/regions.yaml`
2. Add `config/{id}_iso2.yaml` + domain YAMLs (ISO2 must not overlap another region)
3. Add EEZ SVG under `cmd/web/static/img/` (named in registry) + `eez_title_iso_{id}.json` (or ship table-only until SVG exists)
4. Add about template named in registry
5. Register RIPEstat `sourceapp` before large collector bursts
6. Create `.env.{id}` (see `.env.pacific.example` / `.env.caribbean.example`) and enable `ipv6-web@{id}` / `ipv6-collector@{id}`
7. nginx + probe DNS (`ipv4.` / `ipv6.` / apex), TLS SANs, CORS for the new origin
8. Search Console / `PUBLIC_SITE_URL`; outage-report journal unit `ipv6-web@{id}`

## Regional bodies (headquarters rule)

List each regional organization **once**, under its **headquarters** economy’s YAML (not under every member). Update this table when facts change.

### Pacific

| Domain | HQ economy (ISO2) |
|--------|-------------------|
| `ffa.int` | SB |
| `spc.int` | NC |
| `forumsec.org` | FJ |
| `pita.org.fj` | FJ |
| `sprep.org` | WS |

### Caribbean

| Domain | HQ economy (ISO2) |
|--------|-------------------|
| `caricom.org` | GY |
| `ctu.int` | TT |
| `canto.org` | TT |
| `acs-aec.org` | TT |
| `caribank.org` | BB |
| `uwi.edu` | JM |
| `oecs.int` | LC |

Use **`sector: Regional`** and **`regional: true`** on those entries.

## Domain count policy

- **New regions (e.g. Caribbean):** target **≥10** apex domains spanning telecom/ISP, education, government, and regional orgs where applicable.
- **Pacific:** existing lists are grandfathered (some small islands have fewer than 10).

Caribbean domain files: all ISO2 listed in `caribbean_iso2.yaml` have `config/domains/{ISO2}.yaml` curated (target ≥10 apexes). Phase 2 may refine weak/micro-territory entries after collector runs.

## Adding or editing domain lists

Step-by-step methodology: **[domains-methodology.md](domains-methodology.md)**.
