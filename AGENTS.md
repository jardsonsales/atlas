# AGENTS.md — Atlas Architecture & Coding Guidelines

This document defines the architectural rules, conventions, and workflows for **Atlas**, a terminal-based Domain Control Plane built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

All human developers and AI agents working on this repository **must** strictly follow these instructions.

---

## 1. Core Principles

1. **100% English Codebase:**
   - All code, package names, identifiers, comments, UI text, error messages, test descriptions, commit messages, and Makefile targets **must be written in English**.
2. **Domain-Driven Design (DDD):**
   - Business rules and domain invariants live strictly inside the **Domain** and **Application** layers, completely decoupled from Bubble Tea (UI) or OS filesystem implementation details.
3. **Entity & Contract-Oriented Programming:**
   - **Entities** (`internal/domain/entities`) encapsulate state with unexported fields, validated constructors (`New...`), getters, and behavior methods. Never expose raw mutable structs as domain entities.
   - **Contracts** (`internal/domain/contracts` and `internal/ui/contracts`) define interfaces between layers. Every repository, system service, use case, and UI component must be consumed via its contract interface.
4. **Virtual Root Filesystem (Sandbox Pattern):**
   - Atlas interacts with Linux server directories and files (e.g., `/var/www`, `/etc/nginx`, `/etc/letsencrypt`, `/var/log`).
   - **Never hardcode direct OS reads/writes to `/var/...` or `/etc/...` without passing through `contracts.PathResolver`.**
   - Always call `pathResolver.Resolve("/virtual/path")` so the application seamlessly operates against `./dev/fake_root` in development/testing and `/` in production.

---

## 2. Directory Structure

```text
atlas/
├── main.go                                 # Composition root & dependency injection
├── Makefile                                # Build, run, sandbox, and test automation
├── assets/                                 # Embedded Go templates (`//go:embed`)
│   ├── embed.go                            # package assets: exports the embed.FS
│   ├── vhosts/                             # Vhost templates per `<web server>/<model>.txt`
│   └── systemd/service.txt                 # Backend systemd unit template
├── dev/
│   └── fake_root/                          # Auto-seeded sandbox filesystem (git-ignored/disposable)
│       ├── .config/atlas/                    # Sandbox config store (domains.json)
│       ├── etc/nginx/sites-available/      # Fake vhost files (one <domain>.conf per domain)
│       ├── etc/nginx/sites-enabled/        # Symlinks marking active vhosts
│       ├── etc/letsencrypt/live/           # Fake TLS certificates (per domain)
│       ├── var/log/nginx/
│       └── var/www/                        # Fake domain folders (public/, .env, .atlas.json)
└── internal/
    ├── domain/
    │   ├── entities/                       # Domain, WebServer, DomainAudit, DomainConfig, SystemdService, Settings
    │   └── contracts/                      # Domain, repository, use case & system contracts
    ├── application/
    │   └── usecases/                       # Business use cases implementing domain contracts
    ├── infrastructure/
    │   ├── filesystem/                     # PathResolver & development sandbox seeder
    │   ├── rendering/                      # Vhost & systemd template renderers (text/template)
    │   ├── repositories/
    │   │   └── fsrepo/                     # Filesystem-backed repository & inspector implementations
    │   └── system/                         # Host system probe (web server detection) & provisioning applier
    └── ui/
        ├── actions/                        # Bubble Tea commands (tea.Cmd) & messages (tea.Msg)
        ├── app/                            # Root Bubble Tea model (phase-based orchestrator)
        ├── components/                     # systemcheck, domainselector, domainconfig & settings components
        ├── contracts/                      # UI component contracts (Component interface)
        └── styles/                         # Lipgloss color palette and visual styles
```

---

## 3. Layer Responsibilities & Rules

### Domain Layer (`internal/domain/`)
- **Zero external dependencies:** Must not import Bubble Tea, Lipgloss, or infrastructure packages.
- **`entities/`:** Contains domain models that validate their own invariants upon construction and expose domain behavior:
  - `Domain` — domain identity, path, environment, status, and `IsSelectable()`. `NewDomain(...)` validates both the ID and the name with `ValidateDomainName`, which enforces a hostname-safe pattern (`[A-Za-z0-9]` labels separated by `.` or `-`, max `MaxDomainNameLength` 253) and rejects path traversal, whitespace, shell metacharacters and any other unsafe input (`ErrUnsafeDomainName`).
  - `WebServer` — kind (`NGINX` / `APACHE` / `NONE`), version, service name, config dir, running state; exposes `IsInstalled()` / `IsRunning()` / `DisplayName()`.
  - `DomainAudit` — read-only provisioning snapshot of a domain (readiness, vhost linking, HTTPS); exposes `IsReadyForUse()`, `IsActive()`, `IsVHostPending()`, `HasHTTPS()`, `VHostTemplate()`, `CertbotCommand()` and `PendingActions()`.
  - `DomainConfig` — the configurable provisioning plan of a domain (connection type, vhost model, max body size, reverse proxy entry point, operation port, behind-proxy flag, systemd unit); validated by `NewDomainConfig(...)` against the `VHostCatalog` and the sanitized domain name. `OperationPort()` is only meaningful for reverse proxy models (`UsesOperationPort()`); `NextOperationPort(used)` allocates the next free port from `OperationPortBase` (4000). `BehindProxy()` (default `false`) tells the templates whether an upstream proxy already sets `X-Forwarded-*`. It also exposes the vhost layout helpers `SitesAvailableDir()`, `SitesEnabledDir()` and `VHostFileName()` used by the applier.
  - `SystemdService` — the backend service unit generated for a domain (ExecStart, WorkingDirectory, EnvironmentFile, User/Group, Restart, MemoryMax...).
  - `ProvisioningResult` — the outcome of applying a plan: `Domain()` + `Steps()`, where each `ProvisioningStep` (`NewProvisioningStep`) carries a label, a `StepStatus` (`StepApplied` / `StepSkipped` / `StepFailed`) and a human-readable detail. `Succeeded()` reports whether no step failed.
  - `ProvisioningPreview` — the rendered (not yet applied) vhost + systemd output.
  - `Settings` — global Atlas preferences persisted between runs (currently only the certbot contact e-mail); `NewSettings(...)` validates the optional address (`ErrInvalidCertbotEmail`) and `HasCertbotEmail()` reports whether the certificate should be issued with `-m <email>` instead of `--register-unsafely-without-email`.
- **`contracts/`:** Contains interfaces for:
  - Repositories (e.g., `DomainRepository`, `DomainConfigRepository`, `SettingsRepository`)
  - Use Cases (e.g., `ListDomainsUseCase`, `SelectDomainUseCase`, `DetectWebServerUseCase`, `AuditDomainUseCase`, `GetDomainConfigUseCase`, `SaveDomainConfigUseCase`, `GetSettingsUseCase`, `SaveSettingsUseCase`, `RenderProvisioningUseCase`, `AllocateOperationPortUseCase`, `ApplyProvisioningUseCase`)
  - System abstractions (e.g., `PathResolver`, `WebServerProbe`, `DomainInspector`, `VHostRenderer`, `SystemdRenderer`, `DomainEnvWriter`, `ProvisioningApplier`)

### Application Layer (`internal/application/usecases/`)
- Implements the Use Case interfaces defined in `internal/domain/contracts`.
- Depends **only** on `internal/domain/contracts` and `internal/domain/entities`.
- Current use cases: `ListDomains`, `SelectDomain`, `DetectWebServer`, `AuditDomain`, `GetDomainConfig`, `SaveDomainConfig`, `GetSettings`, `SaveSettings`, `RenderProvisioning`, `AllocateOperationPort`, `ApplyProvisioning`.
- Every new feature or operation must have a dedicated Use Case and unit tests (`*_test.go`).

### Infrastructure Layer (`internal/infrastructure/`)
- Implements repository and system contracts defined in `internal/domain/contracts`.
- **Filesystem Sandbox (`filesystem/`):**
  - `NewPathResolver(rootDir)` maps virtual paths (like `/var/www` or `/etc/nginx`) to either the sandbox directory (`./dev/fake_root`) or the real host root (`/`).
  - `SeedSandbox(resolver)` populates fake directories and files in `./dev/fake_root` whenever new system paths or configuration files are required by upcoming features.
- **Rendering (`rendering/`):**
  - `VHostRenderer` and `SystemdRenderer` implement their contracts using `text/template` over the templates embedded in `assets/` (`//go:embed`), so rendering never depends on the working directory.
  - The vhost template is selected via the `VHostCatalog` entry's `TemplateKey` (e.g. `nginx/reverse_proxy` → `assets/vhosts/nginx/reverse_proxy.txt`).
  - Max body size is stored in bytes and formatted per server (`10M` for nginx, `10485760` for Apache).
- **Repositories (`repositories/fsrepo/`):**
  - `FSDomainRepository` implements `DomainRepository`.
  - `FSDomainInspector` implements `DomainInspector` (checks `.env`, `sites-enabled` symlinks and `/etc/letsencrypt` certificates). The vhost layout follows the detected web server: `/etc/nginx/sites-*` or `/etc/apache2/sites-*`.
  - `FSDomainConfigRepository` implements `DomainConfigRepository`, persisting every per-domain configuration as JSON in the config store (`~/.config/atlas/domains.json` in production, `dev/fake_root/.config/atlas/domains.json` in sandbox) with atomic writes and upsert-by-domain.
  - `FSDomainEnvWriter` implements `DomainEnvWriter`, editing **only** the `PORT` key of the domain `.env` (resolved via `PathResolver`) and creating it with just `PORT=<port>` when missing — every other entry and comment is preserved.
  - `FSSettingsRepository` implements `SettingsRepository`, persisting the global Atlas settings (certbot contact e-mail) as JSON in the config store (`settings.json`) with atomic writes — a missing file loads as `DefaultSettings()`.
- **System (`system/`):**
  - `WebServerProbe` implements `WebServerProbe`, detecting the installed web server via systemd units, binaries and config directories, and parsing the version from the server binary (`-v`).
  - **In sandbox mode it always pretends Nginx is installed and running**, so the whole UI flow can be exercised locally.
  - `ProvisioningApplier` implements `ProvisioningApplier`, wiring a rendered plan to the host: it writes the vhost into `sites-available` (via `PathResolver`), (re)creates the `sites-enabled` symlink, runs the web server config test (`nginx -t` / `apachectl -t`) and reloads it through `systemctl`, installs the backend unit into `/etc/systemd/system` (`daemon-reload`, `enable`, `restart`) and issues the certificate with certbot (`--nginx` / `--apache`, `--non-interactive --agree-tos --redirect`, plus `-m <settings certbot email>` or `--register-unsafely-without-email`). A failed config test removes the symlink so a broken vhost never reaches the running server. **All external commands run through an internal `commandRunner` interface using `exec.CommandContext` with fixed argument lists (never a shell).** In sandbox mode no command is executed (each is reported as a skipped step) but the files and symlinks are still written to the fake root.

### UI Layer (`internal/ui/`)
- Built with `github.com/charmbracelet/bubbletea` and `github.com/charmbracelet/lipgloss`.
- **`contracts/`:** All child views/widgets must implement `uicontracts.Component`:
  ```go
  type Component interface {
      Init() tea.Cmd
      Update(msg tea.Msg) (Component, tea.Cmd)
      View() string
      IsInputActive() bool
  }
  ```
- **`app/`:** The root `RootModel` orchestrates a phase state machine — `System Check` → `Domain Selector` → `Domain Detail` → `Domain Config`, plus a `Settings` screen reachable from the Domain Selector — and loads the domain list only after the host web server check is acknowledged.
- **`actions/`:** Side effects and use case invocations must never happen directly inside `Update()` or `View()`. Instead, wrap use case calls inside action functions in `internal/ui/actions/` that return `tea.Cmd` and emit strongly typed `tea.Msg` structs. Keep one action file per feature area (e.g., `domain_actions.go`, `webserver_actions.go`).
- **`components/`:** Each visual feature lives in its own package under `internal/ui/components/` (e.g., `systemcheck`, `domainselector`, `domainconfig`, `settings`) and communicates upward/downward via messages (`actions`).
- **`styles/`:** Centralize all Lipgloss styles and colors in `internal/ui/styles/theme.go` rather than scattering inline styles across components.

---

## 4. Development & Sandbox Workflow

### Startup Flow
1. On launch, Atlas detects the host web server (Nginx / Apache / none) and shows the **System Check** screen.
2. After the user confirms, the **Domain Selector** loads and lists the domains found in `/var/www`. Pressing `s` there opens the **Settings** screen (currently only the certbot contact e-mail; an empty value makes certbot register with `--register-unsafely-without-email`).
3. Selecting a domain runs the **domain audit** and renders the provisioning checks (ready to use, vhost, active, HTTPS).
4. Pressing `c` on the domain detail opens the **Domain Config** form, where the user picks the connection type, vhost model, max body size, reverse proxy entry point, the "Behind a proxy" toggle (trust upstream `X-Forwarded-*`, default off; shown for reverse proxy and PHP-FPM socket models) and backend systemd unit; `p` renders a preview and `s` persists it to the config store. For reverse proxy models Atlas also auto-assigns the operation port (shown read-only) and mirrors it into the domain `.env`.
5. Back on the domain detail, pressing `a` applies the saved plan through the `ApplyProvisioningUseCase` (renders the preview and delegates to the `ProvisioningApplier`), showing the per-step result (applied / skipped / failed), then re-audits the domain. Applying requires a saved configuration; otherwise Atlas asks to save one first with `c`.

### Environment Variables
- `ATLAS_ROOT_DIR`: Root filesystem the `PathResolver` maps virtual paths onto. Defaults to `/` (production); set it to `./dev/fake_root` (as the `run`/`dev` Makefile targets do) to use the development sandbox.
- `ATLAS_CONFIG_DIR`: Overrides the configuration store directory (defaults to `~/.config/atlas` in production, `dev/fake_root/.config/atlas` in sandbox). The store holds both `domains.json` (per-domain plans) and `settings.json` (global settings).
- `CLICOLOR_FORCE=1`: Forces ANSI colors when `termenv` cannot auto-detect color support (e.g., SSH sessions, CI, or non-TTY capture). Useful when running or screenshotting the app outside a fully-featured terminal.

The certbot contact e-mail is **not** an environment variable: it is configured in-app through the **Settings** screen (`s` on the domain list) and persisted in `settings.json` inside the config store.

### Adding New System Paths or Fake Files
When a new task requires reading or modifying system files (e.g., Nginx/Apache configs, systemd units, SSL certificates, `.env` files, or log files):
1. Resolve all paths via `contracts.PathResolver` (e.g., `resolver.Resolve("/etc/nginx/sites-available")`).
2. Update `SeedSandbox` in `internal/infrastructure/filesystem/sandbox_seeder.go` so the fake files/directories are automatically generated in `./dev/fake_root` during development and tests. Today it seeds domain folders (`public/index.html`, `.atlas.json`, optional `.env`), vhost files in `sites-available`, `sites-enabled` symlinks, `/etc/letsencrypt/live` certificates and the `/etc/systemd/system` unit directory.
3. Write unit tests using `t.TempDir()` passed into `filesystem.NewPathResolver(t.TempDir())`.

### Vhost Templates
- Vhosts are **pluggable Go templates** (`text/template`) embedded from `assets/vhosts/<web server>/<model>.txt`; the `VHostCatalog` in `internal/domain/entities/vhost_catalog.go` maps every (web server × connection type × model) combination to its template and required inputs.
- Every template is parameterised with the domain (`{{ .ServerName }}`), the max body size (`{{ .MaxBodySize }}`, formatted per server) and, when applicable, the reverse proxy entry point (`{{ .EntryPoint }}`), the public root (`{{ .RootPath }}`) and the PHP version (`{{ .PHPVersion }}`).
- nginx reverse proxy templates render a per-domain `$connection_upgrade_<domain>` map (`{{ .ConnectionUpgradeVar }}`) so several enabled domains never collide on the same `map` name.
- On the nginx reverse proxy templates the `BehindProxy` flag (`{{ if .BehindProxy }}`, default off) forwards `X-Forwarded-For: $proxy_add_x_forwarded_for` when an upstream proxy is trusted, or overwrites it with `$remote_addr` otherwise. The flag is also captured for PHP-FPM socket models (catalog `SupportsBehindProxy`), which do not consume it in the rendered vhost.
- All templates ship baseline security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`); TLS/HSTS is left to certbot.
- The backend systemd unit is rendered from `assets/systemd/service.txt`, supporting `MemoryMax`, `EnvironmentFile`, `WorkingDirectory`, `User`, `Group`, `Restart`/`RestartSec`.
- Rendering itself is **read-only** (it only produces the preview text shown in the TUI, also stored as JSON in the config store). The `a` key promotes a saved plan to reality through the `ProvisioningApplier`: writing the vhost to `sites-available`, linking it in `sites-enabled`, validating and reloading the web server, installing/enabling the backend unit and issuing the certificate with certbot. Every write goes through `contracts.PathResolver`, every external command runs through the `commandRunner` contract (no shell), and in sandbox mode the commands are skipped while the files are still written to `./dev/fake_root`.

### Makefile Commands
- `make build` — Compile the binary to `./bin/atlas`.
- `make run` — Run in development sandbox mode (`./dev/fake_root`).
- `make dev` — Build binary to `./bin/atlas` and run in sandbox mode.
- `make prod` — Build and run against the real host filesystem (`/`) (requires root).
- `make test` — Run all unit tests (`go test -v ./...`).
- `make vet` / `make fmt` / `make tidy` — Code quality and module maintenance.
- `make clean` — Remove compiled binaries.
- `make clean-sandbox` — Wipe `./dev/fake_root` so it can be cleanly re-seeded on next run.
- `make clean-config` — Remove the sandbox configuration store (`dev/fake_root/.config/atlas`).
- `make help` — List available targets.

---

## 5. Checklist Before Completing Any Task

- [ ] Is all code, UI text, and documentation written in **English**?
- [ ] Are domain entities protecting their invariants with private fields and constructors?
- [ ] Are components, repositories, and use cases wired exclusively through **contracts** (`interfaces`)?
- [ ] Are Bubble Tea side-effects isolated in `internal/ui/actions/`?
- [ ] Are all OS filesystem paths resolved via `contracts.PathResolver` and seeded in `SeedSandbox`?
- [ ] Is every domain name sanitized by the `Domain`/`DomainConfig` constructors before it reaches a path or a command?
- [ ] Does every external command run through the `commandRunner` contract with fixed arguments (no shell) and get skipped in sandbox mode?
- [ ] Does the sandbox keep pretending Nginx is installed, and does the System Check run before the domain list is shown?
- [ ] Do `make vet`, `make test`, and `make build` pass with zero errors?
