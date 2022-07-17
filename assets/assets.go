// Wrapper package that holds the assets.
package assets

import (
	"embed"
)

//go:embed md-alt.css md.css md.tmpl
var FS embed.FS
