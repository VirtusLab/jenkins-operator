package render

import (
	"bytes"
	"text/template"
)

// Render executes a parsed template (go-template) with configuration from data
func Render(template *template.Template, data interface{}) (string, error) {
	var buffer bytes.Buffer
	if err := template.Execute(&buffer, data); err != nil {
		return "", err
	}

	return buffer.String(), nil
}
