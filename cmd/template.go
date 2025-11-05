package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/template"

	"eve.evalgo.org/common"
	evehttp "eve.evalgo.org/http"
	"eve.evalgo.org/registry"
	"github.com/labstack/echo/v4"
)

// TemplateRequest represents a request to render a template
// Semantic representation as Schema.org CreativeWork (specifically a DigitalDocument or template)
type TemplateRequest struct {
	// JSON-LD semantic fields
	Context string `json:"@context,omitempty"` // https://schema.org
	Type    string `json:"@type,omitempty"`    // DigitalDocument or SoftwareSourceCode

	// Schema.org CreativeWork properties
	Text           string `json:"text,omitempty"`           // Template content (inline)
	Identifier     string `json:"identifier,omitempty"`     // Template ID or path
	EncodingFormat string `json:"encodingFormat,omitempty"` // Template format (e.g., "text/template")

	// Template-specific properties
	TemplateParameters map[string]interface{} `json:"templateParameters,omitempty"` // Template variables

	// Legacy fields (for backward compatibility)
	Template   string                 `json:"template,omitempty"`   // Deprecated: use text
	TemplateID string                 `json:"templateId,omitempty"` // Deprecated: use identifier
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Deprecated: use templateParameters
}

// TemplateResponse returns the rendered output
// Semantic representation as Schema.org CreativeWork (rendered document)
type TemplateResponse struct {
	// JSON-LD semantic fields
	Context string `json:"@context,omitempty"` // https://schema.org
	Type    string `json:"@type,omitempty"`    // DigitalDocument or Article

	// Schema.org CreativeWork properties
	Text           string `json:"text,omitempty"`           // Rendered output
	EncodingFormat string `json:"encodingFormat,omitempty"` // Output format (e.g., "text/plain", "text/html")
	ContentSize    int64  `json:"contentSize,omitempty"`    // Size in bytes

	// Legacy fields (for backward compatibility)
	Output string `json:"output,omitempty"` // Deprecated: use text
}

// handleRender renders a template with provided parameters
func handleRender(c echo.Context) error {
	var req TemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Normalize legacy fields to semantic fields for backward compatibility
	if req.Text == "" && req.Template != "" {
		req.Text = req.Template
	}
	if req.Identifier == "" && req.TemplateID != "" {
		req.Identifier = req.TemplateID
	}
	if req.TemplateParameters == nil && req.Parameters != nil {
		req.TemplateParameters = req.Parameters
	}

	// Get template content (prefer semantic fields)
	var templateContent string
	if req.Text != "" {
		// Inline template
		templateContent = req.Text
	} else if req.Identifier != "" {
		// Load from file
		data, err := os.ReadFile(req.Identifier)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("failed to read template file: %v", err),
			})
		}
		templateContent = string(data)
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "either text/template or identifier/templateId is required",
		})
	}

	// Parse and execute template
	tmpl, err := template.New("template").Parse(templateContent)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to parse template: %v", err),
		})
	}

	// Use template parameters (prefer semantic field)
	params := req.TemplateParameters
	if params == nil {
		params = req.Parameters
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, params); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to execute template: %v", err),
		})
	}

	result := output.String()

	response := TemplateResponse{
		// Semantic fields
		Context:        "https://schema.org",
		Type:           "DigitalDocument",
		Text:           result,
		EncodingFormat: "text/plain",
		ContentSize:    int64(len(result)),

		// Legacy fields (for backward compatibility)
		Output: result,
	}

	return c.JSON(http.StatusOK, response)
}

var logger *common.ContextLogger

func main() {
	// Initialize logger
	logger = common.ServiceLogger("templateservice", "1.0.0")

	e := echo.New()

	// REST API endpoint
	e.POST("/v1/api/render", handleRender)

	// Semantic API endpoint with EVE API key middleware
	apiKey := os.Getenv("TEMPLATE_API_KEY")
	apiKeyMiddleware := evehttp.APIKeyMiddleware(apiKey)
	e.POST("/v1/api/semantic/action", handleSemanticAction, apiKeyMiddleware)

	// EVE health check
	e.GET("/health", evehttp.HealthCheckHandler("templateservice", "1.0.0"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}

	// Auto-register with registry service if REGISTRYSERVICE_API_URL is set
	portInt, _ := strconv.Atoi(port)
	if _, err := registry.AutoRegister(registry.AutoRegisterConfig{
		ServiceID:    "templateservice",
		ServiceName:  "Template Rendering Service",
		Description:  "Go template rendering service with semantic action support",
		Port:         portInt,
		Directory:    "/home/opunix/templateservice",
		Binary:       "templateservice",
		Capabilities: []string{"template-rendering", "go-templates", "semantic-actions"},
	}); err != nil {
		logger.WithError(err).Error("Failed to register with registry")
	}

	// Start server in goroutine
	go func() {
		logger.Infof("templateservice starting on port %s", port)
		if err := e.Start(":" + port); err != nil {
			logger.WithError(err).Error("Server error")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Unregister from registry
	if err := registry.AutoUnregister("templateservice"); err != nil {
		logger.WithError(err).Error("Failed to unregister from registry")
	}

	// Shutdown server
	if err := e.Close(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}

	logger.Info("Server stopped")
}
