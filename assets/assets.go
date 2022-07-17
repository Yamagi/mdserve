// Wrapper package that holds the assets.
package assets

import (
	"embed"
)

//go:embed md.tmpl md-block.css md-left.css
var FS embed.FS
