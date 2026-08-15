// Package web embeds the single-page app assets into the binary.
package web

import "embed"

//go:embed index.html app.js style.css
var FS embed.FS
