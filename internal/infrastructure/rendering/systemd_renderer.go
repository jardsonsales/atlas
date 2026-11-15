package rendering

import (
	"bytes"
	"strings"
	"text/template"

	"atlas/assets"
	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

const systemdTemplatePath = "systemd/service.txt"

type systemdRenderer struct{}

// NewSystemdRenderer creates a SystemdRenderer backed by the embedded systemd unit template.
func NewSystemdRenderer() contracts.SystemdRenderer {
	return &systemdRenderer{}
}

// systemdTemplateData is the placeholder context of the systemd unit template.
type systemdTemplateData struct {
	Description      string
	ExecStart        string
	WorkingDirectory string
	EnvironmentFile  string
	User             string
	Group            string
	Restart          string
	RestartSec       int
	MemoryMax        string
}

func (r *systemdRenderer) Render(config *entities.DomainConfig) (string, error) {
	source, err := assets.FS.ReadFile(systemdTemplatePath)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("systemd").Option("missingkey=error").Parse(string(source))
	if err != nil {
		return "", err
	}

	service := config.Systemd()
	data := systemdTemplateData{
		Description:      service.Description(),
		ExecStart:        config.ResolvedExecStart(),
		WorkingDirectory: service.WorkingDirectory(),
		EnvironmentFile:  service.EnvironmentFile(),
		User:             service.User(),
		Group:            service.Group(),
		Restart:          service.Restart(),
		RestartSec:       service.RestartSec(),
		MemoryMax:        service.MemoryMax(),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return strings.TrimRight(buf.String(), "\n") + "\n", nil
}
