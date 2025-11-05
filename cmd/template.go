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
type TemplateRequest struct {
	Template   string                 `json:"template"`   // Template content (inline)
	TemplateID string                 `json:"templateId"` // OR template file path/URL
	Parameters map[string]interface{} `json:"parameters"` // Template variables
}

// TemplateResponse returns the rendered output
type TemplateResponse struct {
	Output         string `json:"output"`
	EncodingFormat string `json:"encodingFormat"`
	ContentSize    int64  `json:"contentSize"`
}

// handleRender renders a template with provided parameters
func handleRender(c echo.Context) error {
	var req TemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Get template content
	var templateContent string
	if req.Template != "" {
		// Inline template
		templateContent = req.Template
	} else if req.TemplateID != "" {
		// Load from file
		data, err := os.ReadFile(req.TemplateID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("failed to read template file: %v", err),
			})
		}
		templateContent = string(data)
	} else {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "either template or templateId is required",
		})
	}

	// Parse and execute template
	tmpl, err := template.New("template").Parse(templateContent)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to parse template: %v", err),
		})
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, req.Parameters); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("failed to execute template: %v", err),
		})
	}

	result := output.String()

	response := TemplateResponse{
		Output:         result,
		EncodingFormat: "text/plain",
		ContentSize:    int64(len(result)),
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
