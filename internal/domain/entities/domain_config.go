package entities

import (
	"errors"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
)

var (
	ErrInvalidDomainConfigDomain     = errors.New("domain configuration requires a domain name")
	ErrInvalidDomainConfigWebServer  = errors.New("domain configuration requires nginx or apache")
	ErrInvalidDomainConfigModel      = errors.New("unknown or incompatible vhost model for the selected web server and connection type")
	ErrInvalidDomainConfigMaxBody    = errors.New("max body size must be greater than zero")
	ErrInvalidDomainConfigEntryPoint = errors.New("reverse proxy requires a valid entry point (host:port)")
	ErrInvalidDomainConfigPort       = errors.New("operation port must be between 1 and 65535")
	ErrInvalidDomainConfigPHPVersion = errors.New("the socket model requires a PHP version")
	ErrInvalidDomainConfigPublicDir  = errors.New("this vhost model does not support serving a public directory")
	ErrInvalidByteSize               = errors.New("invalid byte size value")
	ErrOperationPortExhausted        = errors.New("no operation port available above 4000")
)

const (
	// OperationPortBase is the first operation port Atlas hands out to a domain backend.
	OperationPortBase = 4000
	// OperationPortMax is the highest valid TCP port.
	OperationPortMax = 65535
)

// ConnectionType identifies how the web server reaches the application.
type ConnectionType string

const (
	ConnectionSocket       ConnectionType = "SOCKET"
	ConnectionReverseProxy ConnectionType = "REVERSE_PROXY"
	ConnectionModPHP       ConnectionType = "MOD_PHP"
)

// VHostModel identifies a specific vhost template preset.
type VHostModel string

const (
	ModelNginxReverseProxy       VHostModel = "nginx-reverse-proxy"
	ModelNginxReverseProxyPublic VHostModel = "nginx-reverse-proxy-public"
	ModelNginxSocket             VHostModel = "nginx-socket"
	ModelApacheReverseProxy      VHostModel = "apache-reverse-proxy"
	ModelApacheMod               VHostModel = "apache-mod"
)

// DomainConfig is the per-domain provisioning configuration: connection type, vhost model,
// max body size, reverse proxy entry point, operation port and the backend systemd service.
type DomainConfig struct {
	domain         string
	webServerKind  WebServerKind
	connectionType ConnectionType
	vhostModel     VHostModel
	maxBodyBytes   int64
	entryPoint     string
	operationPort  int
	behindProxy    bool
	servePublicDir bool
	phpVersion     string
	systemd        SystemdService
}

// NewDomainConfig creates a validated DomainConfig, enforcing that the vhost model matches
// the web server and connection type and that every required input is present.
//
// The operation port is only meaningful for reverse proxy models (the backend port written
// to the domain .env as PORT); it may be 0 while still unassigned and is cleared otherwise.
//
// BehindProxy tells the templates whether the domain sits behind an upstream proxy that
// already sets X-Forwarded-*; when false (the default) client-supplied headers are not trusted.
func NewDomainConfig(
	domain string,
	webServerKind WebServerKind,
	connectionType ConnectionType,
	vhostModel VHostModel,
	maxBodyBytes int64,
	entryPoint string,
	operationPort int,
	behindProxy bool,
	servePublicDir bool,
	phpVersion string,
	systemd SystemdService,
) (*DomainConfig, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, ErrInvalidDomainConfigDomain
	}
	// The domain name becomes a filesystem segment (vhost file, /var/www folder, .env), so it
	// must be hostname-safe to block path traversal when Atlas applies the plan.
	if err := ValidateDomainName(domain); err != nil {
		return nil, err
	}

	if webServerKind != WebServerNginx && webServerKind != WebServerApache {
		return nil, ErrInvalidDomainConfigWebServer
	}

	entry, ok := FindModel(webServerKind, vhostModel)
	if !ok || entry.Connection != connectionType {
		return nil, ErrInvalidDomainConfigModel
	}

	if maxBodyBytes <= 0 {
		return nil, ErrInvalidDomainConfigMaxBody
	}

	entryPoint = strings.TrimSpace(entryPoint)
	if entry.RequiresEntryPoint {
		if !isValidEntryPoint(entryPoint) {
			return nil, ErrInvalidDomainConfigEntryPoint
		}
		if operationPort < 0 || operationPort > OperationPortMax {
			return nil, ErrInvalidDomainConfigPort
		}
	} else {
		entryPoint = ""
		operationPort = 0
	}

	phpVersion = strings.TrimSpace(phpVersion)
	if entry.RequiresPHPVersion {
		if phpVersion == "" {
			return nil, ErrInvalidDomainConfigPHPVersion
		}
	} else {
		phpVersion = ""
	}

	if servePublicDir && !entry.SupportsPublicDir {
		return nil, ErrInvalidDomainConfigPublicDir
	}

	return &DomainConfig{
		domain:         domain,
		webServerKind:  webServerKind,
		connectionType: connectionType,
		vhostModel:     vhostModel,
		maxBodyBytes:   maxBodyBytes,
		entryPoint:     entryPoint,
		operationPort:  operationPort,
		behindProxy:    behindProxy,
		servePublicDir: servePublicDir && entry.SupportsPublicDir,
		phpVersion:     phpVersion,
		systemd:        systemd,
	}, nil
}

func (c *DomainConfig) Domain() string                 { return c.domain }
func (c *DomainConfig) WebServerKind() WebServerKind   { return c.webServerKind }
func (c *DomainConfig) ConnectionType() ConnectionType { return c.connectionType }
func (c *DomainConfig) VHostModel() VHostModel         { return c.vhostModel }
func (c *DomainConfig) MaxBodySizeBytes() int64        { return c.maxBodyBytes }
func (c *DomainConfig) EntryPoint() string             { return c.entryPoint }
func (c *DomainConfig) OperationPort() int             { return c.operationPort }
func (c *DomainConfig) BehindProxy() bool              { return c.behindProxy }
func (c *DomainConfig) ServePublicDir() bool           { return c.servePublicDir }
func (c *DomainConfig) PHPVersion() string             { return c.phpVersion }
func (c *DomainConfig) Systemd() SystemdService        { return c.systemd }

// MaxBodySizeNginx renders the max body size the way nginx expects it (e.g. "10M").
func (c *DomainConfig) MaxBodySizeNginx() string { return FormatByteSize(c.maxBodyBytes) }

// MaxBodySizeApache renders the max body size the way Apache expects it (bytes).
func (c *DomainConfig) MaxBodySizeApache() string { return strconv.FormatInt(c.maxBodyBytes, 10) }

// MaxBodySizeForServer renders the max body size using the format of the configured web server.
func (c *DomainConfig) MaxBodySizeForServer() string {
	if c.webServerKind == WebServerApache {
		return c.MaxBodySizeApache()
	}
	return c.MaxBodySizeNginx()
}

// RequiresEntryPoint reports whether the selected model needs a reverse proxy backend.
func (c *DomainConfig) RequiresEntryPoint() bool {
	entry, ok := FindModel(c.webServerKind, c.vhostModel)
	return ok && entry.RequiresEntryPoint
}

// UsesOperationPort reports whether the configuration owns a backend operation port.
// Only reverse proxy models expose a backend port that is written to the domain .env.
func (c *DomainConfig) UsesOperationPort() bool { return c.RequiresEntryPoint() }

// WithOperationPort returns a copy of the configuration carrying the given operation port.
func (c *DomainConfig) WithOperationPort(port int) (*DomainConfig, error) {
	return NewDomainConfig(
		c.domain,
		c.webServerKind,
		c.connectionType,
		c.vhostModel,
		c.maxBodyBytes,
		c.entryPoint,
		port,
		c.behindProxy,
		c.servePublicDir,
		c.phpVersion,
		c.systemd,
	)
}

// NextOperationPort returns the next free operation port given the ports already in use.
// Allocation is sequential from OperationPortBase (4000, 4001, ...).
func NextOperationPort(used []int) int {
	next := OperationPortBase
	for _, port := range used {
		if port >= next {
			next = port + 1
		}
	}
	return next
}

// DomainRoot returns the virtual root directory of the domain (parent of the public dir).
func (c *DomainConfig) DomainRoot() string {
	return path.Join("/var/www", c.domain)
}

// PublicRoot returns the virtual public directory of the domain.
func (c *DomainConfig) PublicRoot() string {
	return path.Join(c.DomainRoot(), "public")
}

// SitesAvailableDir returns the virtual sites-available directory for the detected web server.
func (c *DomainConfig) SitesAvailableDir() string {
	return path.Join(c.vhostBaseDir(), "sites-available")
}

// SitesEnabledDir returns the virtual sites-enabled directory for the detected web server.
func (c *DomainConfig) SitesEnabledDir() string {
	return path.Join(c.vhostBaseDir(), "sites-enabled")
}

// VHostFileName returns the conventional vhost file name of the domain.
func (c *DomainConfig) VHostFileName() string { return c.domain + ".conf" }

// SuggestedVHostPath returns the conventional vhost path for the detected web server.
func (c *DomainConfig) SuggestedVHostPath() string {
	return path.Join(c.SitesAvailableDir(), c.VHostFileName())
}

// vhostBaseDir returns /etc/nginx or /etc/apache2 depending on the web server kind.
func (c *DomainConfig) vhostBaseDir() string {
	if c.webServerKind == WebServerApache {
		return "/etc/apache2"
	}
	return "/etc/nginx"
}

// UnitNameFor returns the conventional systemd unit name of a domain, e.g.
// "nexus.example.com.service". It is the single convention shared by the provisioning
// applier and the non-interactive `atlas --restart <domain>` mode.
func UnitNameFor(domain string) string { return domain + ".service" }

// SuggestedUnitName returns the conventional systemd unit name for the domain.
func (c *DomainConfig) SuggestedUnitName() string { return UnitNameFor(c.domain) }

// isValidEntryPoint validates a reverse proxy target in "host:port" form.
func isValidEntryPoint(value string) bool {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return false
	}
	if strings.TrimSpace(host) == "" {
		return false
	}
	number, err := strconv.Atoi(port)
	if err != nil || number <= 0 || number > 65535 {
		return false
	}
	return true
}

// ParseByteSize parses human readable sizes such as "10M", "512K", "1G" or raw bytes.
func ParseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(strings.ToUpper(raw))
	if value == "" {
		return 0, ErrInvalidByteSize
	}

	multiplier := int64(1)
	switch {
	case strings.HasSuffix(value, "G") || strings.HasSuffix(value, "GB"):
		multiplier = 1024 * 1024 * 1024
		value = strings.TrimSuffix(strings.TrimSuffix(value, "B"), "G")
	case strings.HasSuffix(value, "M") || strings.HasSuffix(value, "MB"):
		multiplier = 1024 * 1024
		value = strings.TrimSuffix(strings.TrimSuffix(value, "B"), "M")
	case strings.HasSuffix(value, "K") || strings.HasSuffix(value, "KB"):
		multiplier = 1024
		value = strings.TrimSuffix(strings.TrimSuffix(value, "B"), "K")
	case strings.HasSuffix(value, "B"):
		value = strings.TrimSuffix(value, "B")
	}

	number, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || number <= 0 {
		return 0, ErrInvalidByteSize
	}

	return number * multiplier, nil
}

// FormatByteSize renders a byte count using the largest exact unit (G/M/K) or raw bytes.
func FormatByteSize(bytes int64) string {
	const (
		kilo = 1024
		mega = 1024 * kilo
		giga = 1024 * mega
	)

	switch {
	case bytes > 0 && bytes%giga == 0:
		return strconv.FormatInt(bytes/giga, 10) + "G"
	case bytes > 0 && bytes%mega == 0:
		return strconv.FormatInt(bytes/mega, 10) + "M"
	case bytes > 0 && bytes%kilo == 0:
		return strconv.FormatInt(bytes/kilo, 10) + "K"
	default:
		return strconv.FormatInt(bytes, 10)
	}
}

// String provides a compact, human readable summary of the configuration.
func (c *DomainConfig) String() string {
	return fmt.Sprintf("%s (%s, %s, max body %s)", c.domain, c.webServerKind, c.vhostModel, c.MaxBodySizeNginx())
}
