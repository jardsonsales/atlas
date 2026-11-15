package entities

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidDomainAudit = errors.New("domain audit requires a non-nil domain")

// VHostTemplateStandard is the single built-in vhost model currently applied to every domain.
//
// TODO(vhost-templates): replace this placeholder with pluggable, user-provided vhost models
// (one per web server flavour: nginx / apache) and allow selecting a template per domain.
// Until those real models are provided, every domain reports this single template.
const VHostTemplateStandard = "standard-vhost"

// DomainAudit is a read-only snapshot of the provisioning state of a single domain,
// covering readiness (.env), vhost linking (sites-enabled) and HTTPS (certbot).
type DomainAudit struct {
	domain        *Domain
	webServerKind WebServerKind
	envPresent    bool
	vhostLinked   bool
	vhostPending  bool
	certPresent   bool
}

// NewDomainAudit creates a validated DomainAudit snapshot.
func NewDomainAudit(
	domain *Domain,
	webServerKind WebServerKind,
	envPresent, vhostLinked, vhostPending, certPresent bool,
) (*DomainAudit, error) {
	if domain == nil {
		return nil, ErrInvalidDomainAudit
	}
	if strings.TrimSpace(string(webServerKind)) == "" {
		webServerKind = WebServerNone
	}

	return &DomainAudit{
		domain:        domain,
		webServerKind: webServerKind,
		envPresent:    envPresent,
		vhostLinked:   vhostLinked,
		vhostPending:  vhostPending,
		certPresent:   certPresent,
	}, nil
}

func (a *DomainAudit) Domain() *Domain              { return a.domain }
func (a *DomainAudit) WebServerKind() WebServerKind { return a.webServerKind }

// IsReadyForUse reports whether the domain has a .env file in its root.
func (a *DomainAudit) IsReadyForUse() bool { return a.envPresent }

// IsActive reports whether the domain vhost is symlinked into sites-enabled.
func (a *DomainAudit) IsActive() bool { return a.vhostLinked }

// IsVHostPending reports whether a vhost file exists in sites-available but is not linked yet.
func (a *DomainAudit) IsVHostPending() bool { return a.vhostPending }

// HasHTTPS reports whether a certificate already exists for the domain.
func (a *DomainAudit) HasHTTPS() bool { return a.certPresent }

// VHostTemplate returns the currently applied (single, placeholder) vhost model name.
func (a *DomainAudit) VHostTemplate() string { return VHostTemplateStandard }

// VHostFileName returns the conventional vhost file name for the domain.
func (a *DomainAudit) VHostFileName() string { return a.domain.Name() + ".conf" }

// CertbotCommand returns the certbot command that issues HTTPS for this specific domain,
// choosing the plugin that matches the detected web server.
func (a *DomainAudit) CertbotCommand() string {
	plugin := "certonly"
	switch a.webServerKind {
	case WebServerNginx:
		plugin = "--nginx"
	case WebServerApache:
		plugin = "--apache"
	}
	return fmt.Sprintf("certbot %s -d %s", plugin, a.domain.Name())
}

// PendingActions lists the provisioning gaps that still need to be resolved for the domain.
func (a *DomainAudit) PendingActions() []string {
	actions := make([]string, 0, 3)

	if !a.envPresent {
		actions = append(actions, "Add a .env file to the domain root")
	}
	if !a.vhostLinked {
		actions = append(actions, fmt.Sprintf("Link %s into sites-enabled", a.VHostFileName()))
	}
	if !a.certPresent {
		actions = append(actions, fmt.Sprintf("Run: %s", a.CertbotCommand()))
	}

	return actions
}
