package entities

// VHostCatalogEntry describes one selectable vhost model: which web server it targets,
// which connection type it implements, where its template lives and which extra inputs
// it requires.
type VHostCatalogEntry struct {
	Model              VHostModel
	Kind               WebServerKind
	Connection         ConnectionType
	DisplayName        string
	TemplateKey        string
	RequiresEntryPoint bool
	SupportsPublicDir  bool
	RequiresPHPVersion bool
	// SupportsBehindProxy reports whether the model can sit behind an upstream proxy that
	// already sets X-Forwarded-* (reverse proxy hops and FastCGI backends, but not mod_php).
	SupportsBehindProxy bool
}

// Catalog is the single source of truth for the vhost models shipped with Atlas.
// TemplateKey maps to assets/vhosts/<TemplateKey>.txt.
func Catalog() []VHostCatalogEntry {
	return []VHostCatalogEntry{
		{
			Model:               ModelNginxReverseProxy,
			Kind:                WebServerNginx,
			Connection:          ConnectionReverseProxy,
			DisplayName:         "Reverse proxy",
			TemplateKey:         "nginx/reverse_proxy",
			RequiresEntryPoint:  true,
			SupportsBehindProxy: true,
		},
		{
			Model:               ModelNginxReverseProxyPublic,
			Kind:                WebServerNginx,
			Connection:          ConnectionReverseProxy,
			DisplayName:         "Reverse proxy + public dir",
			TemplateKey:         "nginx/reverse_proxy_with_public",
			RequiresEntryPoint:  true,
			SupportsPublicDir:   true,
			SupportsBehindProxy: true,
		},
		{
			Model:               ModelNginxSocket,
			Kind:                WebServerNginx,
			Connection:          ConnectionSocket,
			DisplayName:         "PHP-FPM socket",
			TemplateKey:         "nginx/socket",
			RequiresPHPVersion:  true,
			SupportsBehindProxy: true,
		},
		{
			Model:               ModelApacheReverseProxy,
			Kind:                WebServerApache,
			Connection:          ConnectionReverseProxy,
			DisplayName:         "Reverse proxy",
			TemplateKey:         "apache2/reverse_proxy",
			RequiresEntryPoint:  true,
			SupportsBehindProxy: true,
		},
		{
			Model:       ModelApacheMod,
			Kind:        WebServerApache,
			Connection:  ConnectionModPHP,
			DisplayName: "mod_php / static",
			TemplateKey: "apache2/mod",
		},
	}
}

// FindModel returns the catalog entry for a web server kind and model.
func FindModel(kind WebServerKind, model VHostModel) (VHostCatalogEntry, bool) {
	for _, entry := range Catalog() {
		if entry.Kind == kind && entry.Model == model {
			return entry, true
		}
	}
	return VHostCatalogEntry{}, false
}

// ModelsFor returns the catalog entries available for a web server kind and connection type.
func ModelsFor(kind WebServerKind, connection ConnectionType) []VHostCatalogEntry {
	entries := make([]VHostCatalogEntry, 0)
	for _, entry := range Catalog() {
		if entry.Kind == kind && entry.Connection == connection {
			entries = append(entries, entry)
		}
	}
	return entries
}

// ConnectionsFor returns the connection types supported by a web server kind, in catalog order.
func ConnectionsFor(kind WebServerKind) []ConnectionType {
	seen := make(map[ConnectionType]bool)
	connections := make([]ConnectionType, 0)
	for _, entry := range Catalog() {
		if entry.Kind == kind && !seen[entry.Connection] {
			seen[entry.Connection] = true
			connections = append(connections, entry.Connection)
		}
	}
	return connections
}
