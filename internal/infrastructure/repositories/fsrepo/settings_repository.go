package fsrepo

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"atlas/internal/domain/contracts"
	"atlas/internal/domain/entities"
)

const (
	// SettingsFileName is the JSON file holding the global Atlas settings.
	SettingsFileName    = "settings.json"
	settingsFileVersion = 1
)

// settingsFile is the on-disk schema of settings.json.
type settingsFile struct {
	Version      int    `json:"version"`
	CertbotEmail string `json:"certbotEmail"`
}

type fsSettingsRepository struct {
	dir string
}

// NewSettingsRepository creates a SettingsRepository that persists settings.json inside
// configDir (typically ~/.config/atlas). The directory is created on first write.
func NewSettingsRepository(configDir string) contracts.SettingsRepository {
	return &fsSettingsRepository{dir: configDir}
}

func (r *fsSettingsRepository) filePath() string {
	return filepath.Join(r.dir, SettingsFileName)
}

// Load reads the persisted settings, falling back to defaults when nothing has been saved yet.
func (r *fsSettingsRepository) Load(_ context.Context) (*entities.Settings, error) {
	file, err := r.load()
	if err != nil {
		return nil, err
	}
	return entities.NewSettings(file.CertbotEmail)
}

// Save persists the settings atomically (temp file + rename).
func (r *fsSettingsRepository) Save(_ context.Context, settings *entities.Settings) error {
	file := &settingsFile{
		Version:      settingsFileVersion,
		CertbotEmail: settings.CertbotEmail(),
	}
	return r.persist(file)
}

// load reads settings.json, tolerating a missing file (default settings) and defaulting the version.
func (r *fsSettingsRepository) load() (*settingsFile, error) {
	data, err := os.ReadFile(r.filePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &settingsFile{Version: settingsFileVersion}, nil
		}
		return nil, err
	}

	file := &settingsFile{}
	if err := json.Unmarshal(data, file); err != nil {
		return nil, err
	}
	if file.Version == 0 {
		file.Version = settingsFileVersion
	}
	return file, nil
}

// persist writes settings.json atomically (temp file + rename) so a crash never corrupts the store.
func (r *fsSettingsRepository) persist(file *settingsFile) error {
	if err := os.MkdirAll(r.dir, configDirPerm); err != nil {
		return err
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(r.dir, SettingsFileName+".*.tmp")
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
