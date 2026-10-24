// Package runtimeclient embeds the gothic client runtime.
package runtimeclient

import _ "embed"

// RuntimeJS is the client runtime served at /_gothic/runtime.js.
//
//go:embed gothic.js
var RuntimeJS []byte
