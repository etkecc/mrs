// Package template is a thin wrapper over html/template for the common case: parse a string, run
// it, get a string back. Three entry points for three tempers, Execute hands you the error, May
// swallows it and falls back to the input, Must panics.
//
// It's html/template, not text/template, so output is HTML-escaped by default and dropping user
// data into a page won't quietly hand you an XSS hole.
package template

import (
	"bytes"
	"html/template"
)

// Execute parses tplString and runs it with vars as the dot ("."). Returns the rendered string,
// or "" and an error when parsing or execution fails: a malformed template, or a reach for a
// field that isn't there.
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

// May runs the template and falls back to the original tplString when execution fails OR renders
// empty. Handy over config values, where a plain non-template string (or one that renders to
// nothing) should pass through untouched. The swallowed error is the whole point here, not a slip.
func May(tplString string, vars any) string {
	result, _ := Execute(tplString, vars) //nolint:errcheck // that's the point
	if result == "" {
		return tplString
	}
	return result
}

// Must runs the template and panics if anything goes wrong. Only for compile-time-constant
// templates you know are good, never user-supplied ones: a bad template takes the process with it.
func Must(tplString string, vars any) string {
	result, err := Execute(tplString, vars)
	if err != nil {
		panic(err)
	}
	return result
}
