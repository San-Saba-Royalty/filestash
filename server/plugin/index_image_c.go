//go:build cgo && (linux || freebsd)

package plugin

import _ "github.com/mickael-kerjean/filestash/server/plugin/plg_image_c"
