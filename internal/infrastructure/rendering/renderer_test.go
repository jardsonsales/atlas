package rendering_test

import (
	"strings"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/rendering"
)

const tenMegabytes = 10 * 1024 * 1024

func disabledSystemd(t *testing.T) entities.SystemdService {
	t.Helper()
	service, err := entities.NewSystemdService(false, "", "", "", "", "", "always", 5, "")
	if err != nil {
		t.Fatalf("failed to build systemd fixture: %v", err)
	}
	return *service
}

func newConfig(
	t *testing.T,
	domain string,
	kind entities.WebServerKind,
	connection entities.ConnectionType,
	model entities.VHostModel,
	entryPoint string,
	phpVersion string,
	systemd entities.SystemdService,
) *entities.DomainConfig {
	t.Helper()
	config, err := entities.NewDomainConfig(
		domain, kind, connection, model, tenMegabytes, entryPoint, 4001, false, false, phpVersion, systemd,
	)
	if err != nil {
		t.Fatalf("failed to build domain config fixture: %v", err)
	}
	return config
}

func TestVHostRendererNginxReverseProxy(t *testing.T) {
	config := newConfig(
		t,
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		"bin/server",
		"",
		disabledSystemd(t),
	)

	out, err := rendering.NewVHostRenderer().Render(config)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	assertContains(t, out, "server_name nexus.example.com;")
	assertContains(t, out, "client_max_body_size 10M;")
	assertContains(t, out, "proxy_pass http://127.0.0.1:4001;")
}

func TestVHostRendererNginxWebSocketMapAndForwardedHeaders(t *testing.T) {
	disabled := disabledSystemd(t)

	notBehind, err := entities.NewDomainConfig(
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		tenMegabytes,
		"bin/server",
		4000,
		false,
		false,
		"",
		disabled,
	)
	if err != nil {
		t.Fatalf("failed to build domain config fixture: %v", err)
	}

	out, err := rendering.NewVHostRenderer().Render(notBehind)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	// The websocket upgrade map is unique per domain and drives the Connection header.
	mapVar := "connection_upgrade_nexus_example_com"
	assertContains(t, out, "map $http_upgrade $"+mapVar+" {")
	assertContains(t, out, "proxy_set_header Connection $"+mapVar+";")
	// Not behind a proxy: any client-supplied X-Forwarded-For is ignored.
	assertContains(t, out, "proxy_set_header X-Forwarded-For $remote_addr;")
	assertContains(t, out, `add_header X-Frame-Options "SAMEORIGIN" always;`)

	behind, err := entities.NewDomainConfig(
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		tenMegabytes,
		"bin/server",
		4000,
		true,
		false,
		"",
		disabled,
	)
	if err != nil {
		t.Fatalf("failed to build domain config fixture: %v", err)
	}

	out, err = rendering.NewVHostRenderer().Render(behind)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}
	assertContains(t, out, "proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;")
}

func TestVHostRendererNginxSocket(t *testing.T) {
	config := newConfig(
		t,
		"solaris.example.com",
		entities.WebServerNginx,
		entities.ConnectionSocket,
		entities.ModelNginxSocket,
		"",
		"8.3",
		disabledSystemd(t),
	)

	out, err := rendering.NewVHostRenderer().Render(config)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	assertContains(t, out, "root /var/www/solaris.example.com/public;")
	assertContains(t, out, "fastcgi_pass unix:/var/run/php/php8.3-fpm.sock;")
}

func TestVHostRendererApacheReverseProxy(t *testing.T) {
	config := newConfig(
		t,
		"nexus.example.com",
		entities.WebServerApache,
		entities.ConnectionReverseProxy,
		entities.ModelApacheReverseProxy,
		"bin/server",
		"",
		disabledSystemd(t),
	)

	out, err := rendering.NewVHostRenderer().Render(config)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	assertContains(t, out, "ServerName nexus.example.com")
	assertContains(t, out, "LimitRequestBody 10485760")
	assertContains(t, out, "ProxyPass / http://127.0.0.1:4001/")
	assertContains(t, out, `ws://127.0.0.1:4001/$1`)
}

func TestVHostRendererApacheMod(t *testing.T) {
	config := newConfig(
		t,
		"maximus.example.com",
		entities.WebServerApache,
		entities.ConnectionModPHP,
		entities.ModelApacheMod,
		"",
		"",
		disabledSystemd(t),
	)

	out, err := rendering.NewVHostRenderer().Render(config)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	assertContains(t, out, "DocumentRoot /var/www/maximus.example.com/public")
	assertContains(t, out, "LimitRequestBody 10485760")
}

func TestSystemdRenderer(t *testing.T) {
	service, err := entities.NewSystemdService(
		true,
		"4001 - nexus",
		"/opt/nexus",
		"/etc/nexus.env",
		"www-data",
		"www-data",
		"always",
		5,
		"50M",
	)
	if err != nil {
		t.Fatalf("failed to build systemd fixture: %v", err)
	}

	config := newConfig(
		t,
		"nexus.example.com",
		entities.WebServerNginx,
		entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy,
		"/usr/local/bin/nexus",
		"",
		*service,
	)

	out, err := rendering.NewSystemdRenderer().Render(config)
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	assertContains(t, out, "Description=4001 - nexus")
	assertContains(t, out, "ExecStart=/usr/local/bin/nexus")
	assertContains(t, out, "WorkingDirectory=/opt/nexus")
	assertContains(t, out, "EnvironmentFile=/etc/nexus.env")
	assertContains(t, out, "Restart=always")
	assertContains(t, out, "RestartSec=5")
	assertContains(t, out, "MemoryMax=50M")
	assertContains(t, out, "WantedBy=multi-user.target")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected rendered output to contain %q, got:\n%s", needle, haystack)
	}
}
