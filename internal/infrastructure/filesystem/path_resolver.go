package filesystem

import (
	"path/filepath"
	"strings"

	"atlas/internal/domain/contracts"
)

type pathResolver struct {
	rootDir   string
	isSandbox bool
}

// NewPathResolver creates a PathResolver.
// If rootDir is "/" or empty, it maps directly to the real host filesystem.
// If rootDir is e.g. "./dev/fake_root", Resolve("/var/www") returns "./dev/fake_root/var/www".
func NewPathResolver(rootDir string) contracts.PathResolver {
	cleaned := strings.TrimSpace(rootDir)
	if cleaned == "" || cleaned == "/" {
		return &pathResolver{
			rootDir:   "/",
			isSandbox: false,
		}
	}

	return &pathResolver{
		rootDir:   filepath.Clean(cleaned),
		isSandbox: true,
	}
}

func (r *pathResolver) Resolve(virtualPath string) string {
	cleanVirtual := filepath.Clean("/" + strings.TrimSpace(virtualPath))
	if !r.isSandbox {
		return cleanVirtual
	}
	return filepath.Join(r.rootDir, cleanVirtual)
}

func (r *pathResolver) RootDir() string {
	return r.rootDir
}

func (r *pathResolver) IsSandbox() bool {
	return r.isSandbox
}
