package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

// DomainMetadata represents an optional .atlas.json file inside a domain folder
// for enriching domain metadata in both real and fake environments.
type DomainMetadata struct {
	Description string                `json:"description"`
	Environment string                `json:"environment"`
	Status      entities.DomainStatus `json:"status"`
}

// SeedSandbox creates a realistic fake Linux server tree (/var/www, /etc/nginx, /etc/letsencrypt)
// when running in sandbox/development mode so developers can test all features safely.
func SeedSandbox(resolver contracts.PathResolver) error {
	if !resolver.IsSandbox() {
		return nil
	}

	varWww := resolver.Resolve("/var/www")
	etcNginxSites := resolver.Resolve("/etc/nginx/sites-available")
	etcNginxSitesEnabled := resolver.Resolve("/etc/nginx/sites-enabled")
	etcSystemd := resolver.Resolve("/etc/systemd/system")
	varLogNginx := resolver.Resolve("/var/log/nginx")
	letsEncryptLive := resolver.Resolve("/etc/letsencrypt/live")

	for _, dir := range []string{varWww, etcNginxSites, etcNginxSitesEnabled, etcSystemd, varLogNginx, letsEncryptLive} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	sampleDomains := []struct {
		folder      string
		metadata    DomainMetadata
		html        string
		env         bool
		vhostLinked bool
		certificate bool
	}{
		{
			folder: "orion.example.com",
			metadata: DomainMetadata{
				Description: "Main SaaS control plane and customer portal",
				Environment: "production",
				Status:      entities.StatusActive,
			},
			html:        "<h1>Welcome to orion.example.com</h1>",
			env:         true,
			vhostLinked: true,
			certificate: true,
		},
		{
			folder: "vega.example.com",
			metadata: DomainMetadata{
				Description: "High-throughput payment gateway & webhook processor",
				Environment: "production",
				Status:      entities.StatusActive,
			},
			html:        "<h1>Payments API Gateway</h1>",
			env:         true,
			vhostLinked: true,
		},
		{
			folder: "aurora.example.com",
			metadata: DomainMetadata{
				Description: "Pre-release e-commerce storefront testing environment",
				Environment: "staging",
				Status:      entities.StatusActive,
			},
			html:        "<h1>Staging Storefront</h1>",
			env:         true,
			vhostLinked: true,
		},
		{
			folder: "titan.example.com",
			metadata: DomainMetadata{
				Description: "Internal SSO & Identity Provider (Scheduled Maintenance)",
				Environment: "production",
				Status:      entities.StatusMaintenance,
			},
			html:        "<h1>IAM Maintenance</h1>",
			env:         true,
			vhostLinked: false,
		},
		{
			folder: "phoenix.example.com",
			metadata: DomainMetadata{
				Description: "Deprecated v1 billing portal (Decommissioned)",
				Environment: "legacy",
				Status:      entities.StatusInactive,
			},
			html: "<h1>Legacy Billing</h1>",
		},
	}

	for _, sample := range sampleDomains {
		domainDir := filepath.Join(varWww, sample.folder)
		publicDir := filepath.Join(domainDir, "public")
		if err := os.MkdirAll(publicDir, 0o755); err != nil {
			return err
		}

		indexFile := filepath.Join(publicDir, "index.html")
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			if err := os.WriteFile(indexFile, []byte(sample.html+"\n"), 0o644); err != nil {
				return err
			}
		}

		metaFile := filepath.Join(domainDir, ".atlas.json")
		if _, err := os.Stat(metaFile); os.IsNotExist(err) {
			data, err := json.MarshalIndent(sample.metadata, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(metaFile, data, 0o644); err != nil {
				return err
			}
		}

		if sample.env {
			envFile := filepath.Join(domainDir, ".env")
			if _, err := os.Stat(envFile); os.IsNotExist(err) {
				if err := os.WriteFile(envFile, []byte("APP_ENV="+sample.metadata.Environment+"\n"), 0o600); err != nil {
					return err
				}
			}
		}

		if sample.metadata.Status != entities.StatusInactive {
			vhostFile := filepath.Join(etcNginxSites, sample.folder+".conf")
			if _, err := os.Stat(vhostFile); os.IsNotExist(err) {
				if err := os.WriteFile(vhostFile, []byte(seedVHostContent(sample.folder)), 0o644); err != nil {
					return err
				}
			}

			if sample.vhostLinked {
				vhostLink := filepath.Join(etcNginxSitesEnabled, sample.folder+".conf")
				if _, err := os.Lstat(vhostLink); os.IsNotExist(err) {
					if err := os.Symlink(filepath.Join("..", "sites-available", sample.folder+".conf"), vhostLink); err != nil {
						return err
					}
				}
			}
		}

		if sample.certificate {
			certDir := filepath.Join(letsEncryptLive, sample.folder)
			if err := os.MkdirAll(certDir, 0o755); err != nil {
				return err
			}
			certFile := filepath.Join(certDir, "fullchain.pem")
			if _, err := os.Stat(certFile); os.IsNotExist(err) {
				if err := os.WriteFile(certFile, []byte("# fake sandbox certificate for "+sample.folder+"\n"), 0o644); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// seedVHostContent renders the content of the single placeholder vhost model written into
// the sandbox sites-available directory.
//
// TODO(vhost-templates): replace this placeholder with the real, user-provided vhost models
// (nginx / apache presets) once they are available.
func seedVHostContent(domainName string) string {
	return fmt.Sprintf(`# vhost model: %s
server {
    listen 80;
    server_name %s;
    root /var/www/%s/public;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, entities.VHostTemplateStandard, domainName, domainName)
}
