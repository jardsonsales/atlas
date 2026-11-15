package entities

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidDomainID   = errors.New("domain ID cannot be empty")
	ErrInvalidDomainName = errors.New("domain name cannot be empty")
	ErrInvalidDomainPath = errors.New("domain path cannot be empty")
	// ErrUnsafeDomainName rejects a domain name that is not hostname-safe. Because the name
	// is used as a directory segment (/var/www/<name>) and as a vhost file name
	// (sites-available/<name>.conf), anything able to escape those directories (slashes,
	// "..", whitespace, control characters) is refused.
	ErrUnsafeDomainName = errors.New("domain name must be a hostname-safe value (letters, digits, dots and hyphens only)")
)

// domainNamePattern matches a hostname-safe domain name: one or more dot-separated labels of
// letters, digits and hyphens, each starting and ending with an alphanumeric character. It
// intentionally rejects path separators, "..", whitespace and control characters.
var domainNamePattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)*$`)

// MaxDomainNameLength is the maximum length of a fully qualified domain name.
const MaxDomainNameLength = 253

// ValidateDomainName reports whether name is a safe, hostname-like domain name that can be
// used as a filesystem segment without escaping its parent directory.
func ValidateDomainName(name string) error {
	if name == "" {
		return ErrInvalidDomainName
	}
	if len(name) > MaxDomainNameLength || !domainNamePattern.MatchString(name) {
		return ErrUnsafeDomainName
	}
	return nil
}

// DomainStatus represents the operational state of a domain.
type DomainStatus string

const (
	StatusActive      DomainStatus = "ACTIVE"
	StatusMaintenance DomainStatus = "MAINTENANCE"
	StatusInactive    DomainStatus = "INACTIVE"
)

// Domain is the core entity representing a hosted domain directory under /var/www.
type Domain struct {
	id          string
	name        string
	path        string
	description string
	environment string
	status      DomainStatus
}

// NewDomain creates a validated instance of the Domain entity.
func NewDomain(id, name, path, description, environment string, status DomainStatus) (*Domain, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)

	if id == "" {
		return nil, ErrInvalidDomainID
	}
	if name == "" {
		return nil, ErrInvalidDomainName
	}
	if path == "" {
		return nil, ErrInvalidDomainPath
	}
	// The name (and its folder-derived ID) is used as a filesystem segment, so it must be
	// hostname-safe: this is what blocks path traversal once Atlas writes to the host.
	if err := ValidateDomainName(id); err != nil {
		return nil, err
	}
	if err := ValidateDomainName(name); err != nil {
		return nil, err
	}
	if status == "" {
		status = StatusActive
	}

	return &Domain{
		id:          id,
		name:        name,
		path:        path,
		description: strings.TrimSpace(description),
		environment: strings.TrimSpace(environment),
		status:      status,
	}, nil
}

func (d *Domain) ID() string           { return d.id }
func (d *Domain) Name() string         { return d.name }
func (d *Domain) Path() string         { return d.path }
func (d *Domain) Description() string  { return d.description }
func (d *Domain) Environment() string  { return d.environment }
func (d *Domain) Status() DomainStatus { return d.status }
func (d *Domain) IsSelectable() bool   { return d.status != StatusInactive }
