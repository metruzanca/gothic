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

// Str returns the string behind value, or an empty string when it is nil.
func Str(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// StrOr returns the string behind value, or fallback when it is nil.
func StrOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

// True formats an optional bool as "true" or "false".
func True(value *bool) string {
	if value != nil && *value {
		return "true"
	}
	return "false"
}

// If returns yes when cond is true and no otherwise.
func If(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

// ID joins a base identifier and an index with a dash.
func ID(base string, n int) string {
	return base + "-" + strconv.Itoa(n)
}
