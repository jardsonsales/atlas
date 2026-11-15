package fsrepo

import (
	"context"
	"os"
	"path/filepath"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

const (
	virtualWebRoot      = "/var/www"
	virtualLetsEncrypt  = "/etc/letsencrypt/live"
	vhostFileExtension  = ".conf"
	defaultSitesBaseDir = "/etc/nginx"
	apacheSitesBaseDir  = "/etc/apache2"
)

type fsDomainInspector struct {
	resolver contracts.PathResolver
}

// NewFSDomainInspector creates a DomainInspector that reads the provisioning state of a
// domain from the host filesystem (resolved through the PathResolver sandbox).
func NewFSDomainInspector(resolver contracts.PathResolver) contracts.DomainInspector {
	return &fsDomainInspector{
		resolver: resolver,
	}
}

func (i *fsDomainInspector) Inspect(
	_ context.Context,
	domain *entities.Domain,
	webServer *entities.WebServer,
) (*entities.DomainAudit, error) {
	webServerKind := entities.WebServerNone
	if webServer != nil {
		webServerKind = webServer.Kind()
	}

	envPresent := i.fileExists(filepath.Join(i.resolver.Resolve(virtualWebRoot), domain.ID(), ".env"))

	sitesAvailable, sitesEnabled := i.sitesDirs(webServerKind)
	vhostFileName := domain.Name() + vhostFileExtension
	vhostAvailable := i.fileExists(filepath.Join(sitesAvailable, vhostFileName))
	vhostLinked := i.isSymlink(filepath.Join(sitesEnabled, vhostFileName))

	certPresent := i.fileExists(i.resolver.Resolve(
		filepath.Join(virtualLetsEncrypt, domain.Name(), "fullchain.pem"),
	))

	return entities.NewDomainAudit(
		domain,
		webServerKind,
		envPresent,
		vhostLinked,
		vhostAvailable && !vhostLinked,
		certPresent,
	)
}

// sitesDirs returns the resolved sites-available and sites-enabled directories for the
// detected web server.
//
// TODO(vhost-templates): keep the vhost layout aligned with the detected web server and the
// pluggable vhost models once they are provided; today Nginx is the default layout.
func (i *fsDomainInspector) sitesDirs(kind entities.WebServerKind) (available string, enabled string) {
	base := defaultSitesBaseDir
	if kind == entities.WebServerApache {
		base = apacheSitesBaseDir
	}
	return i.resolver.Resolve(filepath.Join(base, "sites-available")),
		i.resolver.Resolve(filepath.Join(base, "sites-enabled"))
}

func (i *fsDomainInspector) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (i *fsDomainInspector) isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
