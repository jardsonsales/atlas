package system

import (
	"context"
	"testing"

	"atlas/internal/domain/entities"
	"atlas/internal/infrastructure/filesystem"
)

func TestDetectPretendsNginxInSandbox(t *testing.T) {
	resolver := filesystem.NewPathResolver(t.TempDir())
	if !resolver.IsSandbox() {
		t.Fatal("expected resolver to be in sandbox mode")
	}

	probe := NewWebServerProbe(resolver)
	server, err := probe.Detect(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if server.Kind() != entities.WebServerNginx {
		t.Errorf("expected sandbox to pretend Nginx is installed, got %s", server.Kind())
	}
	if !server.IsInstalled() {
		t.Error("expected the sandbox web server to be reported as installed")
	}
	if !server.IsRunning() {
		t.Error("expected the sandbox web server to be reported as running")
	}
	if server.ServiceName() != "nginx" {
		t.Errorf("expected service name nginx, got %s", server.ServiceName())
	}
}

func TestParseServerVersion(t *testing.T) {
	tests := []struct {
		name string
		kind entities.WebServerKind
		raw  string
		want string
	}{
		{
			name: "nginx writes version to stderr style output",
			kind: entities.WebServerNginx,
			raw:  "nginx version: nginx/1.24.0\n",
			want: "1.24.0",
		},
		{
			name: "apache version line",
			kind: entities.WebServerApache,
			raw:  "Server version: Apache/2.4.68 (Debian)\nServer built: 2026-06-11T18:28:43\n",
			want: "2.4.68",
		},
		{
			name: "unparseable output",
			kind: entities.WebServerNginx,
			raw:  "command not found",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseServerVersion(tt.kind, tt.raw); got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
