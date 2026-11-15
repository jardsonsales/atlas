package entities

import (
	"errors"
	"strings"
)

var ErrInvalidWebServerKind = errors.New("web server kind cannot be empty")

// WebServerKind identifies a supported web server implementation.
type WebServerKind string

const (
	WebServerNginx  WebServerKind = "NGINX"
	WebServerApache WebServerKind = "APACHE"
	WebServerNone   WebServerKind = "NONE"
)

// WebServer is the core entity describing the web server detected on the host.
type WebServer struct {
	kind        WebServerKind
	version     string
	serviceName string
	configDir   string
	running     bool
}

// NewWebServer creates a validated instance of the WebServer entity.
func NewWebServer(kind WebServerKind, version, serviceName, configDir string, running bool) (*WebServer, error) {
	normalized := WebServerKind(strings.TrimSpace(string(kind)))
	if normalized == "" {
		return nil, ErrInvalidWebServerKind
	}

	return &WebServer{
		kind:        normalized,
		version:     strings.TrimSpace(version),
		serviceName: strings.TrimSpace(serviceName),
		configDir:   strings.TrimSpace(configDir),
		running:     running,
	}, nil
}

func (w *WebServer) Kind() WebServerKind { return w.kind }
func (w *WebServer) Version() string     { return w.version }
func (w *WebServer) ServiceName() string { return w.serviceName }
func (w *WebServer) ConfigDir() string   { return w.configDir }
func (w *WebServer) IsRunning() bool     { return w.running }

// IsInstalled reports whether a supported web server was found on the host.
func (w *WebServer) IsInstalled() bool { return w.kind != WebServerNone && w.kind != "" }

// DisplayName returns the human readable product name of the detected web server.
func (w *WebServer) DisplayName() string {
	switch w.kind {
	case WebServerNginx:
		return "Nginx"
	case WebServerApache:
		return "Apache"
	default:
		return "None"
	}
}
