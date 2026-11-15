package contracts

// PathResolver resolves virtual system paths (like /var/www or /etc/nginx)
// to either the real root filesystem in production or a sandboxed root directory in development.
type PathResolver interface {
	Resolve(virtualPath string) string
	RootDir() string
	IsSandbox() bool
}
