// Package dom formats values used by DOM attributes.
//
// The functions exist so server-rendered markup and generated client code
// produce the same strings.
package dom

import "strconv"

// Px formats an integer pixel length, for example "36px".
func Px(n int) string {
	return strconv.Itoa(n) + "px"
}
