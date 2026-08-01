# Pacific / Caribbean IPv6 deployment monitor

This project publishes **IPv6 (and related DNS/mail/web) deployment estimates** for island and coastal economies in the **Pacific** and the **Caribbean**. It supports [IPv6 Forum](https://www.ipv6forum.com/) regional work—starting with the Pacific Islands IPv6 Council—by measuring curated government, telecom/ISP, education, and regional-organisation domains, combining those checks with **APNIC Labs** capability data (and optional HE BGP / RIPEstat RPKI signals), and showing results on a public dashboard with an EEZ map and per-economy pages. Figures are **measurement estimates**, not a compliance certification.

The same Go codebase runs **one web + collector pair per region** (`REGION=pacific` or `REGION=caribbean`), each with its own data directory, listen port, and public hostname.

## Services

- **`cmd/collector`** — measures configured domains (NIST-style DNS / Mail / Web + simplified DNSSEC + **DMARC** `_dmarc` TXT), ingests **APNIC Labs** `v6economy/{CC}.json`, fetches **Hurricane Electric** [`bgp.he.net/country/{CC}`](https://bgp.he.net/) and merges per-ASN **IPv6 preferred** from [`stats.labs.apnic.net/ipv6/{CC}`](https://stats.labs.apnic.net/ipv6/TK) into `bgp_he_net`, samples per-ASN **RPKI** via [RIPEstat](https://stat.ripe.net/), writes `data/{region}/countries/{ISO2}.json` and `data/{region}/index.json`.
- **`cmd/web`** — serves the UI, JSON API, region EEZ overview (`static/img/EEZ_*.svg` from `config/regions.yaml`), and a sortable home economies table. Set **`PUBLIC_SITE_URL`** when TLS terminates in front of the app.

## Quick start (Pacific)

```bash
cp .env.example .env.local
cp .env.pacific.example .env.pacific   # optional; start scripts source it when present
./scripts/gen_dev_certs.sh
./scripts/start_collector.sh pacific -run-once
./scripts/start_server.sh pacific     # HTTPS on :8082 → data/pacific
```

## Caribbean (second instance on the same machine)

```bash
cp .env.caribbean.example .env.caribbean
./scripts/start_collector.sh caribbean -run-once -country=JM
./scripts/start_server.sh caribbean   # HTTPS on :8083 → data/caribbean
```

Then open **`https://127.0.0.1:8082/`** (Pacific) or **`:8083`** (Caribbean).

**Documentation:** start at **[docs/index.md](docs/index.md)** — regions and domains: **[docs/config-and-domains.md](docs/config-and-domains.md)**.

**License:** [Apache License 2.0](LICENSE) — Copyright 2026 PeachyMango LLC. See [CONTRIBUTING.md](CONTRIBUTING.md).
