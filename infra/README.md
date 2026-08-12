# SUDA FORGE production delivery layer

This directory contains the first production deployment layer for SUDA FORGE. It targets an Ubuntu or Debian server with systemd, PostgreSQL, Go, Node.js, pnpm, Caddy, and LXC available or installable by the operator.

## Install

Run from a checked-out repository:

```bash
sudo SUDA_INSTALL_DIR=/opt/suda-forge bash infra/install.sh suda.example.com
```

For a controlled bootstrap where public DNS is not ready yet, use `SUDA_SKIP_DNS_CHECK=1`. Do not expose the application publicly until DNS and HTTPS are configured.

The installer creates `/etc/suda-forge/suda-forge.env` from the example file on first run. Replace the database URL and optional runtime URLs before production use. Migrations are applied through `scripts/migrate.sh`, the frontend is built into `apps/web/dist`, the backend is built into `bin/suda-forge`, and a systemd unit is installed at `/etc/systemd/system/suda-forge.service`.

## Operations

```bash
systemctl status suda-forge
journalctl -u suda-forge -f
systemctl restart suda-forge
bash infra/lib/health-check.sh
```

Caddy serves the built frontend and proxies `/api/*`, `/healthz`, `/health`, `/ready`, and `/events` to the loopback Go server. The current Caddy template is intentionally conservative: project preview routing, authentication, and domain/certificate lifecycle still require the next production phase.

## Important boundary

The service currently runs as root because SUDA FORGE's LXC runtime provider has not yet been converted to a delegated non-root service account. This is a temporary operational requirement and must be replaced before a security-sensitive public deployment.
