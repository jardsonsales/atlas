package entities_test

import (
	"testing"

	"atlas/internal/domain/entities"
)

func newAuditDomain(t *testing.T, name string) *entities.Domain {
	t.Helper()
	domain, err := entities.NewDomain(
		name, name, "/var/www/"+name, "description", "production", entities.StatusActive,
	)
	if err != nil {
		t.Fatalf("failed to build domain fixture: %v", err)
	}
	return domain
}

func TestDomainAuditReportsPendingProvisioning(t *testing.T) {
	domain := newAuditDomain(t, "vega.example.com")

	audit, err := entities.NewDomainAudit(domain, entities.WebServerNginx, false, false, true, false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if audit.IsReadyForUse() {
		t.Error("expected domain without .env to not be ready for use")
	}
	if audit.IsActive() {
		t.Error("expected domain without sites-enabled link to not be active")
	}
	if !audit.IsVHostPending() {
		t.Error("expected vhost file present in sites-available to be reported as pending")
	}
	if audit.HasHTTPS() {
		t.Error("expected domain without certificate to not have HTTPS")
	}
	if audit.VHostTemplate() != entities.VHostTemplateStandard {
		t.Errorf("expected the single placeholder template %q, got %q",
			entities.VHostTemplateStandard, audit.VHostTemplate())
	}
	if audit.VHostFileName() != "vega.example.com.conf" {
		t.Errorf("unexpected vhost file name: %s", audit.VHostFileName())
	}
	if got := len(audit.PendingActions()); got != 3 {
		t.Errorf("expected 3 pending actions (env, vhost link, certbot), got %d", got)
	}
}

func TestDomainAuditFullyProvisionedHasNoPendingActions(t *testing.T) {
	domain := newAuditDomain(t, "orion.example.com")

	audit, err := entities.NewDomainAudit(domain, entities.WebServerNginx, true, true, false, true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !audit.IsReadyForUse() || !audit.IsActive() || !audit.HasHTTPS() {
		t.Fatal("expected a fully provisioned domain to be ready, active and secured")
	}
	if got := len(audit.PendingActions()); got != 0 {
		t.Errorf("expected no pending actions, got %d", got)
	}
}

func TestDomainAuditCertbotCommandMatchesWebServer(t *testing.T) {
	domain := newAuditDomain(t, "vega.example.com")

	tests := []struct {
		kind entities.WebServerKind
		want string
	}{
		{entities.WebServerNginx, "certbot --nginx -d vega.example.com"},
		{entities.WebServerApache, "certbot --apache -d vega.example.com"},
		{entities.WebServerNone, "certbot certonly -d vega.example.com"},
	}

	for _, tt := range tests {
		audit, err := entities.NewDomainAudit(domain, tt.kind, true, true, false, false)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if got := audit.CertbotCommand(); got != tt.want {
			t.Errorf("kind %s: expected %q, got %q", tt.kind, tt.want, got)
		}
	}
}

func TestNewDomainAuditRejectsNilDomain(t *testing.T) {
	if _, err := entities.NewDomainAudit(nil, entities.WebServerNginx, true, true, false, true); err == nil {
		t.Fatal("expected an error when building an audit without a domain")
	}
}
