package main

import (
	"bytes"
	"net/http"
	"os"
	"text/template"

	"eve.evalgo.org/semantic"
	"github.com/labstack/echo/v4"
)

func handleSemanticAction(c echo.Context) error {
	// Parse semantic action
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(c.Request().Body); err != nil {
		return semantic.ReturnActionError(c, nil, "Failed to read request body", err)
	}
	bodyBytes := buf.Bytes()

	action, err := semantic.ParseSemanticAction(bodyBytes)
	if err != nil {
		return semantic.ReturnActionError(c, nil, "Failed to parse semantic action", err)
	}

	// Dispatch to registered handler using the ActionRegistry
	// No switch statement needed - handlers are registered at startup
	return semantic.Handle(c, action)
}

func handleSemanticReplace(c echo.Context, action *semantic.SemanticAction) error {
	// Get template content from object
	if action.Object == nil {
		return semantic.ReturnActionError(c, action, "object is required", nil)
	}

	var templateContent string

	// Try to get inline text
	if action.Object.Text != "" {
		templateContent = action.Object.Text
	} else if action.Object.ContentUrl != "" {
		// Load from file
		data, err := os.ReadFile(action.Object.ContentUrl)
		if err != nil {
			return semantic.ReturnActionError(c, action, "Failed to read template file", err)
		}
		templateContent = string(data)
	} else {
		return semantic.ReturnActionError(c, action, "object.text or object.contentUrl is required", nil)
	}

	// Get parameters from action.Properties (where template params should be)
	parameters := make(map[string]interface{})

	// Check for parameters in Properties map
	if action.Properties != nil {
		// Look for templateParameters or parameters key
		if templateParams, ok := action.Properties["templateParameters"].(map[string]interface{}); ok {
			parameters = templateParams
		} else if params, ok := action.Properties["parameters"].(map[string]interface{}); ok {
			parameters = params
		} else {
			// Use all properties as parameters
			parameters = action.Properties
		}
	}

	// Parse and execute template
	tmpl, err := template.New("semantic-template").Parse(templateContent)
	if err != nil {
		return semantic.ReturnActionError(c, action, "Failed to parse template", err)
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, parameters); err != nil {
		return semantic.ReturnActionError(c, action, "Failed to execute template", err)
	}

	result := output.String()

	// Determine encoding format
	encodingFormat := "text/plain"
	if action.Object.EncodingFormat != "" {
		encodingFormat = action.Object.EncodingFormat
	}

	// Set result in action properties
	action.Properties["result"] = map[string]interface{}{
		"@type":          "MediaObject",
		"text":           result,
		"encodingFormat": encodingFormat,
		"contentSize":    len(result),
	}

	semantic.SetSuccessOnAction(action)
	return c.JSON(http.StatusOK, action)
}
