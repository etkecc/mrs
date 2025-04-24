// Package template provides a handy wrapper around text/template package.
package template

import (
	"bytes"
	"html/template"
)

// Execute parses and executes template
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

// May parses and executes template, returns original string if parsing fails
func May(tplString string, vars any) string {
	result, _ := Execute(tplString, vars) //nolint:errcheck // that's the point
	if result == "" {
		return tplString
	}
	return result
}

// Must parses and executes template, panics if parsing fails
func Must(tplString string, vars any) string {
	result, err := Execute(tplString, vars)
	if err != nil {
		panic(err)
	}
	return result
}
