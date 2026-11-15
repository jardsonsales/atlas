# Atlas

Atlas is a terminal-based **Domain Control Plane** for Linux servers. It inventories the
domains under `/var/www`, audits how each one is provisioned (vhost, HTTPS, `.env`) and lets
you compose a repeatable provisioning plan (vhost model, max body size, reverse proxy entry
point, backend systemd unit) from a Bubble Tea TUI, then applies it to the host.

## Install

Download the latest release binary for your server architecture directly on the machine.

amd64 (most servers):

```sh
curl -fsSL https://github.com/jardsonsales/atlas/releases/latest/download/atlas-linux-amd64 -o atlas
```

Or with `wget`:

```sh
wget -qO atlas https://github.com/jardsonsales/atlas/releases/latest/download/atlas-linux-amd64
```

arm64:

```sh
curl -fsSL https://github.com/jardsonsales/atlas/releases/latest/download/atlas-linux-arm64 -o atlas
```

Then make it executable and move it onto your `PATH`:

```sh
chmod +x atlas
sudo mv atlas /usr/local/bin/atlas
```

### Verify the download (optional)

Each release also publishes `checksums.txt`:

```sh
curl -fsSLO https://github.com/jardsonsales/atlas/releases/latest/download/checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

## Usage

Atlas operates on the real host filesystem by default:

```sh
sudo atlas
```

Root privileges are required when applying a plan, since Atlas writes under `/etc` (vhosts and
systemd units). Development boots against the bundled sandbox (`./dev/fake_root`) instead by
setting `ATLAS_ROOT_DIR`, which is what `make run` / `make dev` do:

```sh
ATLAS_ROOT_DIR=./dev/fake_root atlas
```

On the domain detail screen press `c` to compose the provisioning plan (vhost model, max body
size, reverse proxy entry point, backend systemd unit), `s` to save it, `p` to preview the
rendered vhost and unit, and `a` to apply it. Applying writes the vhost to `sites-available`,
links it in `sites-enabled`, reloads the web server after a successful config test, requests
the TLS certificate with certbot and installs/enables the backend systemd unit. In the sandbox
the filesystem changes are still written, but the `nginx`/`apachectl`/`systemctl`/`certbot`
commands are skipped.

Environment variables:

| Variable              | Purpose                                                                        |
| --------------------- | ------------------------------------------------------------------------------ |
| `ATLAS_ROOT_DIR`      | Root filesystem Atlas maps virtual paths onto (defaults to `/`).                  |
| `ATLAS_CONFIG_DIR`    | Override the config store (defaults to `~/.config/atlas`).                       |
| `ATLAS_CERTBOT_EMAIL` | Registration email passed to certbot (`-m`); without it Atlas uses `--register-unsafely-without-email`. |
| `CLICOLOR_FORCE`      | Set to `1` to force colors over SSH or in CI.                                   |

## Build from source

```sh
make build   # produces ./bin/atlas
make test    # runs the unit tests
make run     # runs the sandbox (./dev/fake_root)
```

Requires Go (see `go.mod`).
