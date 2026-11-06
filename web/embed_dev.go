//go:build !embedweb

// Package web exposes the built interface to the Go binary.
package web

import "io/fs"

// Dist is nil unless built with -tags embedweb; Vite serves the UI in development.
func Dist() fs.FS { return nil }
