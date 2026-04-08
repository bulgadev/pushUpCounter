# pushUpCounter

Simple push-up tracker built with Go, Echo, templ, and DuckDB.

This project is configured to run behind Caddy so your public site uses HTTPS.

## Why Caddy

Caddy automatically:

- provisions TLS certificates for your domain
- renews certificates
- redirects HTTP to HTTPS

Your Go app runs on an internal port and Caddy handles public traffic.

## Files Added for HTTPS

- Caddyfile: Caddy reverse-proxy and HTTPS config
- main.go: app bind address can now be set with APP_ADDR

Default app bind address is 127.0.0.1:6432.

## Oracle Server Setup (Ubuntu/Debian)

1. Open firewall/security list for ports 80 and 443.
2. Point DNS A record of your domain to the Oracle VM public IP.
3. Install Caddy:

```bash
sudo apt update
sudo apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update
sudo apt install -y caddy
```

4. Copy this repository to the server and build/run your app.

## Configure Environment

Set these environment variables where your app and Caddy run:

- APP_DOMAIN: your domain, for example flexoes.bulga.top
- APP_ADDR: app listen address, default is 127.0.0.1:6432
- ACME_EMAIL: email for certificate notices

Example:

```bash
export APP_DOMAIN=flexoes.bulga.top
export ACME_EMAIL=you@example.com
export APP_ADDR=127.0.0.1:6432
```

## Run Locally With Caddy (Linux)

Terminal 1:

```bash
go run main.go
```

Terminal 2:

```bash
caddy run --config ./Caddyfile --adapter caddyfile
```

Then open:

- https://your-domain

## Production Service Approach

Recommended:

- run the Go app as a systemd service on 127.0.0.1:6432
- run Caddy as the public entrypoint

If Caddy is installed from the package manager, it usually reads:

- /etc/caddy/Caddyfile

Deploy by copying project Caddyfile content there and reloading Caddy:

```bash
sudo caddy fmt --overwrite /etc/caddy/Caddyfile
sudo caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
sudo systemctl reload caddy
```

## Quick Verification

After deployment:

```bash
curl -I http://your-domain
curl -I https://your-domain
```

Expected:

- HTTP responds with redirect to HTTPS
- HTTPS responds with 200

## Troubleshooting

- White screen on HTTPS with no app logs: check Caddy logs first.
- Certificate not issuing: verify DNS points to server and ports 80/443 are open.
- Site unreachable but app works on localhost: confirm Caddy is running and reading the expected Caddyfile.

