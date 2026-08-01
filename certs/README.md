# TLS certificates

**Do not commit** `*.pem` / `*.key` / `*.crt` files. They are listed in `.gitignore`.

Each public site needs certificate coverage for its own hostnames (`pacific.ipv6forum.com` vs `caribbean.ipv6forum.com`, plus `ipv4.` / `ipv6.` probe names). Do **not** reuse a Pacific-only public cert for Caribbean (unless you intentionally issue one multi-SAN cert covering every name).

## Layout

```text
certs/
  pacific/
    cert.pem / key.pem          # local Go listener (dev)
    fullchain.pem / key.pem     # production public / upstream (typical)
  caribbean/
    cert.pem / key.pem
    fullchain.pem / key.pem
```

Set paths per region via **`TLS_CERT_FILE`** and **`TLS_KEY_FILE`** in `.env.pacific` / `.env.caribbean` (see `.env.*.example`). Shared `.env` / `.env.local` may keep a fallback only when a single local process is enough.

| Layer | What the cert must cover | Typical paths |
|--------|---------------------------|---------------|
| **Public** (nginx / Caddy) | Apex + `ipv4.` + `ipv6.` for that region | `/opt/ipv6-pacific/certs/{region}/fullchain.pem` |
| **Go upstream** (`LISTEN`) | Often `localhost` only (nginx → local HTTPS) | `certs/{region}/cert.pem` or the same public files if you terminate on Go |

## Local development

```bash
./scripts/gen_dev_certs.sh
```

Writes a self-signed **localhost** pair into **`certs/pacific/`** and **`certs/caribbean/`** (same SANs: `localhost`, `127.0.0.1`, `::1`). Browsers will warn until you add an exception.

Production LE/public material goes in the same per-region directories (or elsewhere — point `TLS_*` / nginx `ssl_certificate*` at the real files). See **`docs/security.md`**.
