package rendering

import (
	"bytes"
	"path"
	"strings"
	"text/template"

	"atlas/assets"
	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

type vhostRenderer struct{}

// NewVHostRenderer creates a VHostRenderer backed by the embedded vhost templates.
func NewVHostRenderer() contracts.VHostRenderer {
	return &vhostRenderer{}
}

// vhostTemplateData is the placeholder context shared by every vhost template.
type vhostTemplateData struct {
	Domain      string
	ServerName  string
	RootPath    string
	DomainRoot  string
	EntryPoint  string
	MaxBodySize string
	PHPVersion  string
	ServePublic bool
	BehindProxy bool
	// ConnectionUpgradeVar is a per-domain nginx map variable name, so enabling several
	// domains never produces a duplicate "map" directive.
	ConnectionUpgradeVar string
}

func (r *vhostRenderer) Render(config *entities.DomainConfig) (string, error) {
	entry, ok := entities.FindModel(config.WebServerKind(), config.VHostModel())
	if !ok {
		return "", entities.ErrInvalidDomainConfigModel
	}

	source, err := assets.FS.ReadFile(path.Join("vhosts", entry.TemplateKey+".txt"))
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("vhost").Option("missingkey=error").Parse(string(source))
	if err != nil {
		return "", err
	}

	data := vhostTemplateData{
		Domain:               config.Domain(),
		ServerName:           config.Domain(),
		RootPath:             config.PublicRoot(),
		DomainRoot:           config.DomainRoot(),
		EntryPoint:           config.EntryPoint(),
		MaxBodySize:          config.MaxBodySizeForServer(),
		PHPVersion:           config.PHPVersion(),
		ServePublic:          config.ServePublicDir(),
		BehindProxy:          config.BehindProxy(),
		ConnectionUpgradeVar: connectionUpgradeVariable(config.Domain()),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// connectionUpgradeVariable builds a valid, domain-unique nginx variable name used by the
// "$connection_upgrade" map idiom (nginx variable names allow only letters, digits and "_").
func connectionUpgradeVariable(domain string) string {
	var b strings.Builder
	b.WriteString("connection_upgrade_")
	for _, r := range domain {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
