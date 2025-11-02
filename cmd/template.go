package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

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

	log.Printf("Rendered template (size: %d bytes)", len(result))

	return c.JSON(http.StatusOK, response)
}

func main() {
	e := echo.New()

	// REST API endpoint
	e.POST("/v1/api/render", handleRender)

	// Semantic API endpoint (will be added)
	e.POST("/v1/api/semantic/action", handleSemanticAction)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}

	log.Printf("templateservice starting on port %s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
