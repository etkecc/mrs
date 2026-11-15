// Package template wraps html/template; output is HTML-escaped by default, unlike text/template.
package template

import (
	"bytes"
	"html/template"
)

// Execute parses and runs tplString with vars as the dot ("."), returning "" and an error on failure.
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

// May runs the template, falling back to tplString when execution fails or renders empty; errors are swallowed.
func May(tplString string, vars any) string {
	result, _ := Execute(tplString, vars) //nolint:errcheck // that's the point
	if result == "" {
		return tplString
	}
	return result
}

// Must runs the template and panics on any error; use only for known-good templates, never user-supplied ones.
func Must(tplString string, vars any) string {
	result, err := Execute(tplString, vars)
	if err != nil {
		panic(err)
	}
	return result
}
