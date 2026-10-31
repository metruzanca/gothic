// Package route matches URL paths against segment patterns.
//
// A pattern is a slash-separated path. A segment that starts with ":" captures
// one non-empty path segment. The matching rules are shared by the server and
// the generated client JavaScript.
//
//	params := route.Params("/todos/:id", r.URL.Path)
//	if params != nil {
//		id := params["id"]
//	}
package route

import "strings"

// Params matches path against pattern and returns the captured segments. It
// returns nil when the paths do not match.
func Params(pattern, path string) map[string]string {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	if len(patternParts) != len(pathParts) {
		return nil
	}
	params := map[string]string{}
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			name := part[1:]
			if name == "" || pathParts[i] == "" {
				return nil
			}
			params[name] = pathParts[i]
			continue
		}
		if part != pathParts[i] {
			return nil
		}
	}
	return params
}

// Match reports whether path matches pattern.
func Match(pattern, path string) bool {
	return Params(pattern, path) != nil
}
