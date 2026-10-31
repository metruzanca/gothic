package gothic

import _ "embed"

// The framework default assets served under /_gothic/. The PNGs are generated
// from the SVG sources with headless Chrome. See docs/ITERATION_21.md.
var (
	//go:embed assets/gothic-mark.svg
	defaultIconSVG []byte

	//go:embed assets/icon-512.png
	defaultIconPNG []byte

	//go:embed assets/apple-touch-icon.png
	defaultAppleTouchIcon []byte

	//go:embed assets/og.png
	defaultOGImage []byte
)
