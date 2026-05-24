// Package template provides a handy wrapper around the html/template package.
//
// Note: This package uses html/template (not text/template), which means output
// is HTML-escaped by default. This prevents XSS vulnerabilities when rendering
// user-supplied data into HTML templates.
package template

import (
	"bytes"
	"html/template"
)

// Execute parses the template string and executes it with the provided variables.
//
// The template string is parsed once and then executed with vars as the data
// context (accessible as "." in the template). It returns the rendered output
// as a string, or an empty string and an error if either parsing or execution
// fails.
//
// Failure modes:
//   - Parse error: malformed template syntax
//   - Execute error: accessing a missing field with no default, or other runtime errors
//
// vars is the data passed as "." to the template and can be any value.
func Execute(tplString string, vars any) (string, error) {
	var result bytes.Buffer
	tpl, err := template.New("template").Parse(tplString)
	if err != nil {
		return "", err
	}
	err = tpl.Execute(&result, vars)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

// May parses and executes the template string, returning the original string
// if execution fails or produces an empty result.
//
// May is useful for applying templates to configuration values where a
// non-template string (or a template that happens to be empty after rendering)
// should pass through unchanged. If either template parsing or execution fails,
// or if the result is empty, the original tplString is returned.
//
// The nolint:errcheck directive is intentional — errors are deliberately ignored
// as part of the fallback-to-original behavior.
func May(tplString string, vars any) string {
	result, _ := Execute(tplString, vars) //nolint:errcheck // that's the point
	if result == "" {
		return tplString
	}
	return result
}

// Must parses and executes the template string, panicking if parsing or
// execution fails.
//
// Must should only be used when the template is a compile-time constant and
// correctness is guaranteed. It is not appropriate for user-supplied templates.
// Any error during parsing or execution will cause a panic.
func Must(tplString string, vars any) string {
	result, err := Execute(tplString, vars)
	if err != nil {
		panic(err)
	}
	return result
}
