package system

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

// sandboxPlaceholderVersion is reported while running against the development sandbox,
// where a real web server installation is not present.
const sandboxPlaceholderVersion = "1.24.0 (sandbox)"

type webServerProbe struct {
	resolver contracts.PathResolver
}

// NewWebServerProbe creates a WebServerProbe that inspects the host filesystem,
// systemd units and binaries (or fakes Nginx while running in the development sandbox).
func NewWebServerProbe(resolver contracts.PathResolver) contracts.WebServerProbe {
	return &webServerProbe{
		resolver: resolver,
	}
}

// candidate describes how to recognise a supported web server on the host.
type candidate struct {
	kind        entities.WebServerKind
	serviceName string
	units       []string
	binaries    []string
	configDirs  []string
}

func (p *webServerProbe) Detect(ctx context.Context) (*entities.WebServer, error) {
	// Development sandbox: always pretend Nginx is installed so the whole UI flow can be exercised.
	if p.resolver.IsSandbox() {
		return entities.NewWebServer(
			entities.WebServerNginx,
			sandboxPlaceholderVersion,
			"nginx",
			p.resolver.Resolve("/etc/nginx"),
			true,
		)
	}

	for _, c := range p.candidates() {
		if !p.isInstalled(c) {
			continue
		}
		return entities.NewWebServer(
			c.kind,
			p.detectVersion(ctx, c),
			c.serviceName,
			p.configDir(c),
			p.isRunning(ctx, c),
		)
	}

	return entities.NewWebServer(entities.WebServerNone, "", "", "", false)
}

func (p *webServerProbe) candidates() []candidate {
	return []candidate{
		{
			kind:        entities.WebServerNginx,
			serviceName: "nginx",
			units:       []string{"nginx"},
			binaries:    []string{"/usr/sbin/nginx", "/usr/local/sbin/nginx"},
			configDirs:  []string{"/etc/nginx", "/usr/local/etc/nginx"},
		},
		{
			kind:        entities.WebServerApache,
			serviceName: "apache2",
			units:       []string{"apache2", "httpd"},
			binaries:    []string{"/usr/sbin/apache2", "/usr/sbin/httpd", "/usr/sbin/apachectl"},
			configDirs:  []string{"/etc/apache2", "/etc/httpd"},
		},
	}
}

// isInstalled reports whether any binary, config directory or systemd unit of the candidate exists.
func (p *webServerProbe) isInstalled(c candidate) bool {
	for _, bin := range c.binaries {
		if fileExists(p.resolver.Resolve(bin)) {
			return true
		}
	}
	for _, dir := range c.configDirs {
		if fileExists(p.resolver.Resolve(dir)) {
			return true
		}
	}
	for _, unit := range c.units {
		if unitEnabled(unit) {
			return true
		}
	}
	return false
}

func (p *webServerProbe) configDir(c candidate) string {
	for _, dir := range c.configDirs {
		if fileExists(p.resolver.Resolve(dir)) {
			return dir
		}
	}
	if len(c.configDirs) > 0 {
		return c.configDirs[0]
	}
	return ""
}

func (p *webServerProbe) firstBinary(c candidate) string {
	for _, bin := range c.binaries {
		if fileExists(p.resolver.Resolve(bin)) {
			return p.resolver.Resolve(bin)
		}
	}
	return ""
}

// detectVersion asks the server binary for its version string (-v), tolerating failures.
func (p *webServerProbe) detectVersion(ctx context.Context, c candidate) string {
	bin := p.firstBinary(c)
	if bin == "" {
		return ""
	}
	out, _ := exec.CommandContext(ctx, bin, "-v").CombinedOutput()
	return parseServerVersion(c.kind, string(out))
}

// isRunning reports whether the candidate systemd service is currently active.
func (p *webServerProbe) isRunning(ctx context.Context, c candidate) bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	for _, unit := range c.units {
		out, _ := exec.CommandContext(ctx, "systemctl", "is-active", unit+".service").Output()
		if strings.TrimSpace(string(out)) == "active" {
			return true
		}
	}
	return false
}

// parseServerVersion extracts the version token from a server's `-v` output,
// e.g. "nginx version: nginx/1.24.0" or "Server version: Apache/2.4.68 (Debian)".
func parseServerVersion(kind entities.WebServerKind, raw string) string {
	marker := "nginx/"
	if kind == entities.WebServerApache {
		marker = "Apache/"
	}

	idx := strings.Index(raw, marker)
	if idx < 0 {
		return ""
	}
	rest := raw[idx+len(marker):]
	if end := strings.IndexAny(rest, " \t\r\n"); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func unitEnabled(unit string) bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	out, err := exec.Command("systemctl", "is-enabled", unit+".service").Output()
	if err != nil {
		return false
	}
	switch strings.TrimSpace(string(out)) {
	case "enabled", "enabled-runtime", "static":
		return true
	default:
		return false
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
