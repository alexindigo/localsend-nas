// Package web embeds the single-page app assets into the binary.
package web

import "embed"

//go:embed index.html app.js style.css favicon.png
var FS embed.FS
