package entities

import (
	"errors"
	"strings"
)

var (
	ErrSystemdMissingDescription = errors.New("systemd service requires a description")
	ErrSystemdInvalidRestartSec  = errors.New("systemd RestartSec cannot be negative")
)

// SystemdService is the backend service unit generated for a domain. It mirrors the
// essential directives of a systemd .service file (WorkingDirectory, EnvironmentFile,
// User/Group, Restart, MemoryMax...). The ExecStart command is not stored here: Atlas
// derives it from the DomainConfig entry point binary.
type SystemdService struct {
	enabled         bool
	description     string
	workingDir      string
	environmentFile string
	user            string
	group           string
	restart         string
	restartSec      int
	memoryMax       string
}

// NewSystemdService creates a validated SystemdService. When enabled, a description is
// mandatory so a renderable unit is always produced.
func NewSystemdService(
	enabled bool,
	description, workingDir, environmentFile, user, group, restart string,
	restartSec int,
	memoryMax string,
) (*SystemdService, error) {
	if restartSec < 0 {
		return nil, ErrSystemdInvalidRestartSec
	}

	service := &SystemdService{
		enabled:         enabled,
		description:     strings.TrimSpace(description),
		workingDir:      strings.TrimSpace(workingDir),
		environmentFile: strings.TrimSpace(environmentFile),
		user:            strings.TrimSpace(user),
		group:           strings.TrimSpace(group),
		restart:         strings.TrimSpace(restart),
		restartSec:      restartSec,
		memoryMax:       strings.TrimSpace(memoryMax),
	}

	if enabled {
		if service.description == "" {
			return nil, ErrSystemdMissingDescription
		}
	}

	if service.restart == "" {
		service.restart = "always"
	}

	return service, nil
}

// DefaultSystemdService returns the baseline unit defaults offered by the Atlas form.
func DefaultSystemdService() SystemdService {
	return SystemdService{
		enabled:    false,
		restart:    "always",
		restartSec: 5,
		user:       "www-data",
		group:      "www-data",
		memoryMax:  "512M",
	}
}

func (s SystemdService) Enabled() bool            { return s.enabled }
func (s SystemdService) Description() string      { return s.description }
func (s SystemdService) WorkingDirectory() string { return s.workingDir }
func (s SystemdService) EnvironmentFile() string  { return s.environmentFile }
func (s SystemdService) User() string             { return s.user }
func (s SystemdService) Group() string            { return s.group }
func (s SystemdService) Restart() string          { return s.restart }
func (s SystemdService) RestartSec() int          { return s.restartSec }
func (s SystemdService) MemoryMax() string        { return s.memoryMax }
