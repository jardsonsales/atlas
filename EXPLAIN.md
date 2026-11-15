# EXPLAIN.md — Preparing an application for Atlas

Audience: the AI agent responsible for **a single application**. Atlas owns the whole server
side — web server, ports, vhosts, TLS and systemd. Your only job is to leave the application
**ready to be wired**. You do **not** install or configure nginx/Apache/certbot, and you do
**not** write vhosts. Just make the app behave the way Atlas expects.

## 1. How Atlas sees your app

A domain is just a folder under `/var/www`. Atlas lists each folder and lets the operator
provision it:

```
/var/www/<domain>/            # folder name == domain name (hostname-safe)
├── .env.example              # committed template of the runtime config  (you provide)
├── .env                      # real runtime config, incl. PORT            (human writes it)
├── .atlas.json               # optional metadata (description/environment/status)
└── public/                   # document root for static / PHP apps
```

Rules:
- Folder name must be **hostname-safe**: labels of `[A-Za-z0-9]` separated by `.` or `-`,
  max 253 chars. No `/`, spaces, `..` (path traversal is rejected).
- Atlas never bootstraps the app: you deliver the folder and the runnable binary.

## 2. The `.env` file — why your app needs one

Atlas assigns each app a **runtime port** and communicates it through `.env`; it never edits
your code or your command line. Concretely:

- For reverse-proxy apps Atlas **auto-assigns an operation port** (starting at `4000`) and
  writes it to `/var/www/<domain>/.env` as `PORT=<port>`. It creates the file with just that
  line when missing, and otherwise **only touches the `PORT` key** — every other entry and
  comment is preserved.
- **Your app must read `PORT` from its environment / `.env` and bind to it.** Never hardcode
  the port: the operator and Atlas rely on this contract to keep apps collision-free.
- **The real `.env` is created by a human (DevOps), never by an automated script.** Atlas only
  writes/updates the `PORT` key; everything else is authored by the operator on the server, by
  hand or from a secret store. The application agent and the deploy pipeline must **never**
  generate, overwrite or commit the real `.env` — an auto-written file would hide required
  configuration and leak secrets.

Because `.env` holds real configuration (and often secrets), commit an **`.env.example`** that
documents every variable instead, and keep `.env` out of version control:

```dotenv
# Assigned by Atlas (reverse proxy models) — the app MUST bind to this port.
PORT=4001

# Application configuration
APP_ENV=production
DATABASE_URL=postgres://user:pass@localhost:5432/app
```

Load the file at startup (e.g. `godotenv`, `dotenv`, `os.environ`) and treat `PORT` as the
single source of truth for the listening port.

## 3. Reverse proxy: your binary is the systemd entry point

For the **Reverse proxy** models, nginx/Apache sits in front and forwards traffic to your app:

```
Internet → nginx/Apache (80/443) → http://127.0.0.1:<PORT> → your binary
```

- Your binary runs as a **systemd service**; its launch command is the service's `ExecStart`,
  i.e. it **is the entry point** that nginx/Apache reverse-proxies to.
- Therefore the app must be a **long-running HTTP server** listening on **loopback**
  (`127.0.0.1:<PORT>`, never `0.0.0.0` — the web server is the only public listener).
- Point the systemd `EnvironmentFile` at the app's `.env` so `PORT` (and the rest) reach the
  process. The binary may be shipped **relative to the domain root** (e.g. `bin/<app>`) or as an
  **absolute path** (e.g. `/var/www/<domain>/bin/<app>`); Atlas turns it into an absolute
  `ExecStart` (no shell).
- Atlas also forwards `Host`, `X-Real-IP`, `X-Forwarded-Proto` and `X-Forwarded-For`, and
  supports WebSocket upgrades, so a standard HTTP server works without extra config.

## 4. Deploy workflow

### 4.1 Minimal deploy

The smallest deploy that is useful ships exactly two things:

- the **EntryPoint** — for reverse-proxy apps, the runnable command that becomes the systemd
  `ExecStart` (a path relative to the domain root, e.g. `bin/<app>`, or an absolute one);
- the **`.env.example`** — the committed template documenting every variable, including `PORT`.

Everything else (vhost, TLS, systemd unit and the real `.env`) is created by Atlas and the
operator, not by your deploy.

### 4.2 The real `.env` is authored by a human

Atlas only writes/updates the `PORT` key. The **real `.env` is created by a human (DevOps)** on
the server. Neither the application agent nor the deploy pipeline may generate or overwrite it
automatically: it carries secrets and the configuration the operator is expected to review. Ship
`.env.example`; let a person produce the real `.env`.

### 4.3 Detect the first deploy (no `.env`)

A workflow must behave differently the first time, because the app is not configured yet. Detect
the first deploy by checking for `/var/www/<domain>/.env` — **absent means first deploy**:

- **First deploy (`/.env` missing):** ship **only the primary/essential artifacts** — the
  EntryPoint and `.env.example` — so the operator can provision the domain in Atlas, fill in the
  real `.env` and configure everything. Do **not** run migrations, seeders, cache warmups or
  anything that needs real configuration.
- **Subsequent deploys (`.env` exists):** the app is configured, so run the **full** workflow —
  dependencies, build, **migrations**, asset compilation, etc. — and finish with the restart
  below.

```sh
if [ -f "/var/www/<domain>/.env" ]; then
  # Configured: run the full workflow.
  # ... install deps, build, migrate, compile assets ...
  atlas --restart <domain>
else
  # First deploy: ship EntryPoint + .env.example only, then stop and wait for the operator.
  :
fi
```

### 4.4 Restart the service with Atlas

Shipping a new build is not enough: because the app runs as a systemd unit, the running process
keeps serving the **old** binary until it is restarted. Do **not** call `systemctl` yourself —
Atlas owns the unit, so every (re-)deploy ends with:

```sh
atlas --restart <domain>
```

- It runs `systemctl restart <domain>.service` for the domain, prints the result and exits
  non-zero on failure, so your pipeline aborts when the service does not come back up.
- It only restarts the backend unit — the vhost, TLS certificate and `.env` are left untouched.
- It requires no TUI and no extra flags; run it as `root` (same as provisioning).
- On a machine where the domain was never provisioned (no unit yet), the restart fails — only
  call it for a domain that has an enabled backend (reverse-proxy apps).

So: **provision once** (in the Atlas TUI, then a human writes `.env`) and on every later deploy
run the full workflow ending in **`atlas --restart <domain>`**.

## 5. Static / PHP apps

For the **PHP-FPM socket** and **mod_php / static** models there is no binary: the web server
serves the app directly.
- Put the document root in `public/` with an entry file: `index.php` (socket/PHP) and/or
  `index.html` (static).
- The PHP-FPM socket model targets `unix:/var/run/php/php<version>-fpm.sock`.
- These models have no systemd unit, so `atlas --restart` does not apply to them.

## 6. What Atlas does for you

So you don't duplicate it: Atlas writes the vhost into `sites-available`, links it in
`sites-enabled`, validates and reloads the web server, installs/enables the systemd unit, and
issues the TLS certificate with certbot. It also adds baseline security headers and the
per-domain max body size. Your app only needs to run, read `PORT`, and be restarted on deploy.

## 7. Checklist for the application agent

- [ ] The app lives at `/var/www/<domain>` with a hostname-safe name.
- [ ] A committed `.env.example` documents every variable, including `PORT`.
- [ ] The minimal deploy ships the **EntryPoint** (reverse-proxy apps) and `.env.example`.
- [ ] The real `.env` is created by a **human (DevOps)**; neither the agent nor a script writes it.
- [ ] The workflow **detects the first deploy** (no `.env`) and then ships only the essential
      artifacts, deferring migrations and the rest to later deploys.
- [ ] `.env` is **not** committed; secrets stay out of the repository.
- [ ] Reverse-proxy apps read `PORT` and bind **loopback**; no hardcoded port.
- [ ] The app starts from an **absolute** command (the systemd `ExecStart`).
- [ ] The deploy pipeline ends with `atlas --restart <domain>` (reverse-proxy apps).
- [ ] Static/PHP apps expose `public/index.php` and/or `public/index.html`.

