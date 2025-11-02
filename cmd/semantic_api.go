package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/labstack/echo/v4"
)

// SemanticTemplateAction represents a ReplaceAction for template rendering
// Uses Schema.org ReplaceAction to represent template substitution
type SemanticTemplateAction struct {
	Context    string                 `json:"@context,omitempty"`
	Type       string                 `json:"@type"`
	Identifier string                 `json:"identifier"`
	Name       string                 `json:"name,omitempty"`
	Object     *SemanticMediaObject   `json:"object,omitempty"`     // Template source
	TargetCollection interface{}       `json:"targetCollection,omitempty"` // Parameters
	Result     *SemanticMediaObject   `json:"result,omitempty"`     // Output
}

// SemanticMediaObject represents template or output
type SemanticMediaObject struct {
	Type           string                 `json:"@type,omitempty"`
	ContentURL     string                 `json:"contentUrl,omitempty"`     // File path or URL
	Text           string                 `json:"text,omitempty"`           // Inline content
	EncodingFormat string                 `json:"encodingFormat,omitempty"` // text/plain, application/sparql-query, etc.
	AdditionalType string                 `json:"additionalType,omitempty"` // "Template"
	Properties     map[string]interface{} `json:"properties,omitempty"`     // Template parameters
}

func handleSemanticAction(c echo.Context) error {
	// Parse raw JSON to detect action type
	var rawAction map[string]interface{}
	if err := c.Bind(&rawAction); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON-LD"})
	}

	actionType, ok := rawAction["@type"].(string)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "@type is required"})
	}

	switch actionType {
	case "ReplaceAction":
		return handleSemanticReplace(c, rawAction)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("unsupported action type: %s (expected ReplaceAction)", actionType),
		})
	}
}

func handleSemanticReplace(c echo.Context, rawAction map[string]interface{}) error {
	actionBytes, _ := json.Marshal(rawAction)
	var action SemanticTemplateAction
	if err := json.Unmarshal(actionBytes, &action); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action structure"})
	}

	// Get template content
	var templateContent string
	if action.Object != nil {
		if action.Object.Text != "" {
			// Inline template
			templateContent = action.Object.Text
		} else if action.Object.ContentURL != "" {
			// Load from file
			data, err := os.ReadFile(action.Object.ContentURL)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": fmt.Sprintf("failed to read template: %v", err),
				})
			}
			templateContent = string(data)
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "object.text or object.contentUrl is required",
			})
		}
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "object is required",
		})
	}

	// Get parameters from targetCollection (can be object or array of PropertyValue)
	var parameters map[string]interface{}

	if action.TargetCollection != nil {
		switch tc := action.TargetCollection.(type) {
		case map[string]interface{}:
			// Direct object with properties
			if props, ok := tc["properties"].(map[string]interface{}); ok {
				parameters = props
			} else {
				parameters = tc
			}
		case []interface{}:
			// Array of PropertyValue objects
			parameters = make(map[string]interface{})
			for _, item := range tc {
				if propVal, ok := item.(map[string]interface{}); ok {
					if name, hasName := propVal["name"].(string); hasName {
						if value, hasValue := propVal["value"]; hasValue {
							parameters[name] = value
						}
					}
				}
			}
		}
	}

	// Also check object.properties for parameters
	if action.Object != nil && action.Object.Properties != nil {
		if parameters == nil {
			parameters = action.Object.Properties
		} else {
			// Merge properties
			for k, v := range action.Object.Properties {
				parameters[k] = v
			}
		}
	}

	// Parse and execute template
	tmpl, err := template.New("semantic-template").Parse(templateContent)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to parse template: %v", err),
		})
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, parameters); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to execute template: %v", err),
		})
	}

	result := output.String()

	// Determine encoding format
	encodingFormat := "text/plain"
	if action.Object != nil && action.Object.EncodingFormat != "" {
		encodingFormat = action.Object.EncodingFormat
	}

	// Return action with result
	action.Result = &SemanticMediaObject{
		Type:           "MediaObject",
		Text:           result,
		EncodingFormat: encodingFormat,
	}

	return c.JSON(http.StatusOK, action)
}
