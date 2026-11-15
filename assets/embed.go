// Package assets embeds the vhost and systemd templates shipped with Atlas so the
// renderers never depend on the current working directory.
package assets

import "embed"

// FS holds the embedded vhost templates (vhosts/<key>.txt) and the systemd unit
// template (systemd/service.txt).
//
//go:embed vhosts systemd
var FS embed.FS
