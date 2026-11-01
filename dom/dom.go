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

// Ptr returns a pointer to value. Use it to pass optional pointer props from a
// view, where address-of expressions are not supported.
func Ptr[T any](value T) *T {
	return &value
}

// BoolPtr returns a pointer to value.
func BoolPtr(value bool) *bool {
	return &value
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
