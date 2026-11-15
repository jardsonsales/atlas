package fsrepo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

const (
	// ConfigFileName is the JSON file holding every per-domain provisioning configuration.
	ConfigFileName    = "domains.json"
	configFileVersion = 1
	configDirPerm     = 0o755
	configFilePerm    = 0o644
)

// domainConfigFile is the on-disk schema of domains.json.
type domainConfigFile struct {
	Version int               `json:"version"`
	Domains []domainConfigDTO `json:"domains"`
}

// domainConfigDTO mirrors entities.DomainConfig without exposing the entity internals.
type domainConfigDTO struct {
	Domain           string         `json:"domain"`
	WebServer        string         `json:"webServer"`
	ConnectionType   string         `json:"connectionType"`
	VHostModel       string         `json:"vhostModel"`
	MaxBodySizeBytes int64          `json:"maxBodySizeBytes"`
	EntryPoint       string         `json:"entryPoint"`
	OperationPort    int            `json:"operationPort"`
	BehindProxy      bool           `json:"behindProxy"`
	ServePublicDir   bool           `json:"servePublicDir"`
	PHPVersion       string         `json:"phpVersion"`
	Systemd          systemdUnitDTO `json:"systemd"`
}

type systemdUnitDTO struct {
	Enabled          bool   `json:"enabled"`
	Description      string `json:"description"`
	ExecStart        string `json:"execStart"`
	WorkingDirectory string `json:"workingDirectory"`
	EnvironmentFile  string `json:"environmentFile"`
	User             string `json:"user"`
	Group            string `json:"group"`
	Restart          string `json:"restart"`
	RestartSec       int    `json:"restartSec"`
	MemoryMax        string `json:"memoryMax"`
}

type fsDomainConfigRepository struct {
	dir string
}

// NewDomainConfigRepository creates a DomainConfigRepository that persists domains.json
// inside configDir (typically ~/.config/atlas). The directory is created on first write.
func NewDomainConfigRepository(configDir string) contracts.DomainConfigRepository {
	return &fsDomainConfigRepository{dir: configDir}
}

func (r *fsDomainConfigRepository) filePath() string {
	return filepath.Join(r.dir, ConfigFileName)
}

func (r *fsDomainConfigRepository) FindAll(_ context.Context) ([]*entities.DomainConfig, error) {
	file, err := r.load()
	if err != nil {
		return nil, err
	}

	configs := make([]*entities.DomainConfig, 0, len(file.Domains))
	for _, dto := range file.Domains {
		config, err := dto.toEntity()
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Domain() < configs[j].Domain()
	})

	return configs, nil
}

func (r *fsDomainConfigRepository) FindByDomain(_ context.Context, domain string) (*entities.DomainConfig, error) {
	file, err := r.load()
	if err != nil {
		return nil, err
	}

	for _, dto := range file.Domains {
		if dto.Domain == domain {
			return dto.toEntity()
		}
	}

	return nil, contracts.ErrDomainConfigNotFound
}

func (r *fsDomainConfigRepository) Save(_ context.Context, config *entities.DomainConfig) error {
	file, err := r.load()
	if err != nil {
		return err
	}

	dto := toDomainConfigDTO(config)
	replaced := false
	for i, existing := range file.Domains {
		if existing.Domain == dto.Domain {
			file.Domains[i] = dto
			replaced = true
			break
		}
	}
	if !replaced {
		file.Domains = append(file.Domains, dto)
	}
	file.Version = configFileVersion

	return r.persist(file)
}

// load reads domains.json, tolerating a missing file (empty store) and defaulting the version.
func (r *fsDomainConfigRepository) load() (*domainConfigFile, error) {
	data, err := os.ReadFile(r.filePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &domainConfigFile{Version: configFileVersion}, nil
		}
		return nil, err
	}

	file := &domainConfigFile{}
	if err := json.Unmarshal(data, file); err != nil {
		return nil, err
	}
	if file.Version == 0 {
		file.Version = configFileVersion
	}
	return file, nil
}

// persist writes domains.json atomically (temp file + rename) so a crash never corrupts the store.
func (r *fsDomainConfigRepository) persist(file *domainConfigFile) error {
	if err := os.MkdirAll(r.dir, configDirPerm); err != nil {
		return err
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(r.dir, ConfigFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(configFilePerm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, r.filePath())
}

func (d domainConfigDTO) toEntity() (*entities.DomainConfig, error) {
	systemd, err := entities.NewSystemdService(
		d.Systemd.Enabled,
		d.Systemd.Description,
		d.Systemd.ExecStart,
		d.Systemd.WorkingDirectory,
		d.Systemd.EnvironmentFile,
		d.Systemd.User,
		d.Systemd.Group,
		d.Systemd.Restart,
		d.Systemd.RestartSec,
		d.Systemd.MemoryMax,
	)
	if err != nil {
		return nil, err
	}

	return entities.NewDomainConfig(
		d.Domain,
		entities.WebServerKind(d.WebServer),
		entities.ConnectionType(d.ConnectionType),
		entities.VHostModel(d.VHostModel),
		d.MaxBodySizeBytes,
		d.EntryPoint,
		d.OperationPort,
		d.BehindProxy,
		d.ServePublicDir,
		d.PHPVersion,
		*systemd,
	)
}

func toDomainConfigDTO(config *entities.DomainConfig) domainConfigDTO {
	service := config.Systemd()
	return domainConfigDTO{
		Domain:           config.Domain(),
		WebServer:        string(config.WebServerKind()),
		ConnectionType:   string(config.ConnectionType()),
		VHostModel:       string(config.VHostModel()),
		MaxBodySizeBytes: config.MaxBodySizeBytes(),
		EntryPoint:       config.EntryPoint(),
		OperationPort:    config.OperationPort(),
		BehindProxy:      config.BehindProxy(),
		ServePublicDir:   config.ServePublicDir(),
		PHPVersion:       config.PHPVersion(),
		Systemd: systemdUnitDTO{
			Enabled:          service.Enabled(),
			Description:      service.Description(),
			ExecStart:        service.ExecStart(),
			WorkingDirectory: service.WorkingDirectory(),
			EnvironmentFile:  service.EnvironmentFile(),
			User:             service.User(),
			Group:            service.Group(),
			Restart:          service.Restart(),
			RestartSec:       service.RestartSec(),
			MemoryMax:        service.MemoryMax(),
		},
	}
}
