package serve

import "net/http"

// Package plugin allows the web UI to be optionally included in the jd
// binary. The compiled WASM UI files are generated by the Makefile and
// are not included in source control. However `go get` will not generate
// them so the binary has to build with or without the UI.

var Handle func(http.ResponseWriter, *http.Request)