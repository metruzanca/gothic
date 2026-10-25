package parser

import (
	"fmt"
	"go/token"
)

// clientReservedNames are identifiers that generated JavaScript already uses.
var clientReservedNames = map[string]bool{
	"readState": true,
	"signal":    true,
	"each":      true,
	"on":        true,
	"rpc":       true,
	"hydrate":   true,
	"state":     true,
	"root":      true,
	"undefined": true,
	"null":      true,
	"true":      true,
	"false":     true,
	"arguments": true,
	"eval":      true,
}

// jsReservedWords are JavaScript keywords that are valid Go identifiers.
var jsReservedWords = map[string]bool{
	"class":      true,
	"delete":     true,
	"export":     true,
	"extends":    true,
	"finally":    true,
	"function":   true,
	"import":     true,
	"in":         true,
	"instanceof": true,
	"let":        true,
	"new":        true,
	"super":      true,
	"this":       true,
	"throw":      true,
	"try":        true,
	"typeof":     true,
	"void":       true,
	"while":      true,
	"with":       true,
	"yield":      true,
	"enum":       true,
	"await":      true,
	"implements": true,
	"interface":  true,
	"package":    true,
	"private":    true,
	"protected":  true,
	"public":     true,
	"static":     true,
}

// Validate checks names that would collide in generated Go or JavaScript.
func Validate(f *File) error {
	procedures := map[string]bool{}
	for _, p := range f.Procedures {
		if err := validateName("procedure", p.Name); err != nil {
			return err
		}
		if procedures[p.Name] {
			return fmt.Errorf("duplicate procedure %q", p.Name)
		}
		procedures[p.Name] = true
	}

	for _, t := range f.Templates {
		states := map[string]bool{}
		for _, sd := range t.State {
			if err := validateName("state", sd.Name); err != nil {
				return fmt.Errorf("template %s: %w", t.Name, err)
			}
			if states[sd.Name] {
				return fmt.Errorf("template %s: duplicate state %q", t.Name, sd.Name)
			}
			if procedures[sd.Name] {
				return fmt.Errorf("template %s: state %q collides with an rpc procedure", t.Name, sd.Name)
			}
			states[sd.Name] = true
		}
	}
	return nil
}

func validateName(kind, name string) error {
	if !token.IsIdentifier(name) {
		return fmt.Errorf("%s name %q is not a valid Go identifier", kind, name)
	}
	if clientReservedNames[name] {
		return fmt.Errorf("%s name %q is reserved by the gothic client runtime", kind, name)
	}
	if jsReservedWords[name] {
		return fmt.Errorf("%s name %q is a reserved JavaScript word", kind, name)
	}
	return nil
}
