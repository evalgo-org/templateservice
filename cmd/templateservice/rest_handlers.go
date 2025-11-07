package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
)

// REST endpoint request types

type RenderRequest struct {
	Template   string                 `json:"template"`
	TemplateID string                 `json:"templateId,omitempty"`
	Parameters map[string]interface{} `json:"parameters"`
}

// registerRESTEndpoints adds REST endpoints that convert to semantic actions
func registerRESTEndpoints(apiGroup *echo.Group, apiKeyMiddleware echo.MiddlewareFunc) {
	// POST /v1/api/render - Render template
	apiGroup.POST("/render", renderTemplateREST, apiKeyMiddleware)
}

// renderTemplateREST handles REST POST /v1/api/render
// Converts to ReplaceAction and delegates to semantic handler
func renderTemplateREST(c echo.Context) error {
	var req RenderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid request: %v", err)})
	}

	// Validate required fields
	if req.Template == "" && req.TemplateID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "template or templateId is required"})
	}

	// Build object (template content)
	object := map[string]interface{}{
		"@type": "MediaObject",
	}
	if req.Template != "" {
		object["text"] = req.Template
	}
	if req.TemplateID != "" {
		object["contentUrl"] = req.TemplateID
	}

	// Convert to JSON-LD ReplaceAction
	action := map[string]interface{}{
		"@context": "https://schema.org",
		"@type":    "ReplaceAction",
		"object":   object,
	}

	// Add parameters if provided
	if req.Parameters != nil {
		action["additionalProperty"] = req.Parameters
	}

	return callSemanticHandler(c, action)
}

// callSemanticHandler converts action to JSON and calls the semantic action handler
func callSemanticHandler(c echo.Context, action map[string]interface{}) error {
	// Marshal action to JSON
	actionJSON, err := json.Marshal(action)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to marshal action: %v", err)})
	}

	// Create new request with JSON-LD body
	newReq := c.Request().Clone(c.Request().Context())
	newReq.Body = io.NopCloser(bytes.NewReader(actionJSON))
	newReq.Header.Set("Content-Type", "application/json")

	// Create new context with modified request
	newCtx := c.Echo().NewContext(newReq, c.Response())
	newCtx.SetPath(c.Path())
	newCtx.SetParamNames(c.ParamNames()...)
	newCtx.SetParamValues(c.ParamValues()...)

	// Call the existing semantic action handler
	return handleSemanticAction(newCtx)
}
