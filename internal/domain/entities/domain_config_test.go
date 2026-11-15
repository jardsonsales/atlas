package entities_test

import (
	"errors"
	"testing"

	"atlas/internal/domain/entities"
)

func TestNewDomainConfigValidatesInvariants(t *testing.T) {
	disabled := entities.DefaultSystemdService()

	tests := []struct {
		name        string
		domain      string
		kind        entities.WebServerKind
		connection  entities.ConnectionType
		model       entities.VHostModel
		maxBody     int64
		entryPoint  string
		port        int
		servePublic bool
		phpVersion  string
		wantErr     error
	}{
		{
			name:       "valid nginx reverse proxy",
			domain:     "nexus.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "127.0.0.1:4001",
			port:       4000,
		},
		{
			name:       "empty domain",
			domain:     "  ",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "127.0.0.1:4001",
			port:       4000,
			wantErr:    entities.ErrInvalidDomainConfigDomain,
		},
		{
			name:       "missing entry point for reverse proxy",
			domain:     "nexus.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "",
			port:       4000,
			wantErr:    entities.ErrInvalidDomainConfigEntryPoint,
		},
		{
			name:       "operation port above the maximum",
			domain:     "nexus.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "127.0.0.1:4001",
			port:       70000,
			wantErr:    entities.ErrInvalidDomainConfigPort,
		},
		{
			name:       "model incompatible with connection type",
			domain:     "nexus.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionSocket,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "127.0.0.1:4001",
			port:       4000,
			wantErr:    entities.ErrInvalidDomainConfigModel,
		},
		{
			name:       "model incompatible with web server",
			domain:     "nexus.example.com",
			kind:       entities.WebServerApache,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    10 * 1024 * 1024,
			entryPoint: "127.0.0.1:4001",
			port:       4000,
			wantErr:    entities.ErrInvalidDomainConfigModel,
		},
		{
			name:       "non positive max body size",
			domain:     "nexus.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionReverseProxy,
			model:      entities.ModelNginxReverseProxy,
			maxBody:    0,
			entryPoint: "127.0.0.1:4001",
			port:       4000,
			wantErr:    entities.ErrInvalidDomainConfigMaxBody,
		},
		{
			name:       "socket without php version",
			domain:     "solaris.example.com",
			kind:       entities.WebServerNginx,
			connection: entities.ConnectionSocket,
			model:      entities.ModelNginxSocket,
			maxBody:    10 * 1024 * 1024,
			phpVersion: "",
			wantErr:    entities.ErrInvalidDomainConfigPHPVersion,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := entities.NewDomainConfig(
				tc.domain, tc.kind, tc.connection, tc.model,
				tc.maxBody, tc.entryPoint, tc.port, false, tc.servePublic, tc.phpVersion, disabled,
			)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got: %v", tc.wantErr, err)
			}
		})
	}
}

func TestDomainConfigFormatsMaxBodySizePerServer(t *testing.T) {
	disabled := entities.DefaultSystemdService()

	nginx, err := entities.NewDomainConfig(
		"nexus.example.com", entities.WebServerNginx, entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy, 10*1024*1024, "127.0.0.1:4001", 0, false, false, "", disabled,
	)
	if err != nil {
		t.Fatalf("failed to build nginx config: %v", err)
	}
	if got := nginx.MaxBodySizeForServer(); got != "10M" {
		t.Errorf("expected nginx max body size 10M, got %s", got)
	}

	apache, err := entities.NewDomainConfig(
		"nexus.example.com", entities.WebServerApache, entities.ConnectionReverseProxy,
		entities.ModelApacheReverseProxy, 10*1024*1024, "127.0.0.1:4001", 0, false, false, "", disabled,
	)
	if err != nil {
		t.Fatalf("failed to build apache config: %v", err)
	}
	if got := apache.MaxBodySizeForServer(); got != "10485760" {
		t.Errorf("expected apache max body size 10485760, got %s", got)
	}
	if got := apache.SuggestedVHostPath(); got != "/etc/apache2/sites-available/nexus.example.com.conf" {
		t.Errorf("unexpected apache vhost path: %s", got)
	}
}

func TestDomainConfigOperationPort(t *testing.T) {
	disabled := entities.DefaultSystemdService()

	proxy, err := entities.NewDomainConfig(
		"nexus.example.com", entities.WebServerNginx, entities.ConnectionReverseProxy,
		entities.ModelNginxReverseProxy, 10*1024*1024, "127.0.0.1:4000", 0, false, false, "", disabled,
	)
	if err != nil {
		t.Fatalf("failed to build proxy config: %v", err)
	}
	if !proxy.UsesOperationPort() {
		t.Errorf("expected reverse proxy to use an operation port")
	}

	withPort, err := proxy.WithOperationPort(4001)
	if err != nil {
		t.Fatalf("failed to set operation port: %v", err)
	}
	if withPort.OperationPort() != 4001 {
		t.Errorf("expected operation port 4001, got %d", withPort.OperationPort())
	}

	socket, err := entities.NewDomainConfig(
		"solaris.example.com", entities.WebServerNginx, entities.ConnectionSocket,
		entities.ModelNginxSocket, 10*1024*1024, "", 4001, false, false, "8.3", disabled,
	)
	if err != nil {
		t.Fatalf("failed to build socket config: %v", err)
	}
	if socket.UsesOperationPort() {
		t.Errorf("expected socket model not to use an operation port")
	}
	if socket.OperationPort() != 0 {
		t.Errorf("expected socket operation port to be cleared, got %d", socket.OperationPort())
	}
}

func TestNextOperationPort(t *testing.T) {
	cases := []struct {
		used []int
		want int
	}{
		{nil, 4000},
		{[]int{}, 4000},
		{[]int{4000}, 4001},
		{[]int{4001, 4000}, 4002},
		{[]int{4000, 4002}, 4003},
	}
	for _, tc := range cases {
		if got := entities.NextOperationPort(tc.used); got != tc.want {
			t.Errorf("NextOperationPort(%v) = %d, want %d", tc.used, got, tc.want)
		}
	}
}

func TestParseAndFormatByteSize(t *testing.T) {
	cases := map[string]int64{
		"10M":   10 * 1024 * 1024,
		"512K":  512 * 1024,
		"1G":    1024 * 1024 * 1024,
		"2048":  2048,
		"10mb":  10 * 1024 * 1024,
		"512kb": 512 * 1024,
	}

	for raw, want := range cases {
		got, err := entities.ParseByteSize(raw)
		if err != nil {
			t.Fatalf("ParseByteSize(%q) returned error: %v", raw, err)
		}
		if got != want {
			t.Errorf("ParseByteSize(%q) = %d, want %d", raw, got, want)
		}
	}

	if _, err := entities.ParseByteSize("nope"); !errors.Is(err, entities.ErrInvalidByteSize) {
		t.Errorf("expected ErrInvalidByteSize, got: %v", err)
	}

	if got := entities.FormatByteSize(10 * 1024 * 1024); got != "10M" {
		t.Errorf("FormatByteSize(10M) = %s, want 10M", got)
	}
	if got := entities.FormatByteSize(1500); got != "1500" {
		t.Errorf("FormatByteSize(1500) = %s, want 1500", got)
	}
}
