package cmd

import (
	"mime"
	"path/filepath"
)

type Mime struct {
	ContentType string
	Category    string
}

func mimeTypeForFile(file string) Mime {
	ext := filepath.Ext(file)

	switch ext {
	case "htm", "html":
		return Mime{ContentType: "text/html", Category: "html"}

	case "css":
		return Mime{ContentType: "text/css", Category: "asset"}

	case "js":
		return Mime{ContentType: "application/javascript", Category: "asset"}

	case "json":
		return Mime{ContentType: "application/json", Category: "asset"}

	default:
		return Mime{ContentType: mime.TypeByExtension(ext), Category: ""}
	}
}
