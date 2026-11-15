package entities

// ProvisioningPreview is the rendered (not yet applied) provisioning output for a domain:
// the web server vhost and, when enabled, the backend systemd unit.
type ProvisioningPreview struct {
	vhost     string
	systemd   string
	vhostPath string
	unitName  string
}

// NewProvisioningPreview creates a ProvisioningPreview value object.
func NewProvisioningPreview(vhost, systemd, vhostPath, unitName string) *ProvisioningPreview {
	return &ProvisioningPreview{
		vhost:     vhost,
		systemd:   systemd,
		vhostPath: vhostPath,
		unitName:  unitName,
	}
}

func (p *ProvisioningPreview) VHost() string              { return p.vhost }
func (p *ProvisioningPreview) Systemd() string            { return p.systemd }
func (p *ProvisioningPreview) SuggestedVHostPath() string { return p.vhostPath }
func (p *ProvisioningPreview) SuggestedUnitName() string  { return p.unitName }
