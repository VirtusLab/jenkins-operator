package resources

import (
	"bytes"
	"text/template"
)

// render executes a parsed template (go-template) with configuration from data
func render(template *template.Template, data interface{}) (string, error) {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, data); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
