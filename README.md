# Template Service

Go template rendering service with semantic action support for dynamic content generation.

## Overview

Template Service provides semantic and REST interfaces for rendering Go templates with dynamic data. It's part of the EVE (Evalgo Virtual Environment) semantic service ecosystem and supports both inline templates and file-based template loading.

## Features

- **Go Template Engine**: Full support for Go's text/template syntax
- **Dual Interface**: Both semantic actions and REST endpoints
- **Inline & File Templates**: Support for inline template strings or file-based templates
- **Schema.org Compliance**: Templates and responses as CreativeWork/DigitalDocument
- **State Tracking**: Built-in operation state management
- **Auto-Discovery**: Automatic registry service registration
- **API Key Protection**: Optional authentication via API keys

## Architecture

```
REST Endpoints → JSON-LD Conversion → Semantic Action Handler → Go Template Engine
```

The service uses a thin REST adapter pattern where all REST endpoints convert requests to Schema.org JSON-LD actions and delegate to the semantic handler.

## Installation

```bash
# Build the service
go build -o templateservice ./cmd/templateservice

# Or using task
task build
```

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8095` |
| `TEMPLATE_SERVICE_API_KEY` | API key for endpoint protection | (optional) |
| `REGISTRYSERVICE_API_URL` | Registry service URL | (optional) |

## Usage

### Start the service

```bash
export TEMPLATE_SERVICE_API_KEY=your-secret-key
export PORT=8095
./templateservice
```

### Health check

```bash
curl http://localhost:8095/health
```

### Service documentation

```bash
curl http://localhost:8095/v1/api/docs
```

## API Reference

### Semantic Action Endpoint (Primary Interface)

**POST** `/v1/api/semantic/action`

Accepts Schema.org JSON-LD actions for template rendering.

#### ReplaceAction - Render Template

##### Inline Template Example

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "text": "Hello {{.Name}}, welcome to {{.Service}}!",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "Name": "Alice",
      "Service": "EVE"
    }
  }
}
```

Response:

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "actionStatus": "CompletedActionStatus",
  "result": {
    "@type": "DigitalDocument",
    "text": "Hello Alice, welcome to EVE!",
    "encodingFormat": "text/plain",
    "contentSize": 29
  }
}
```

##### File-Based Template Example

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "identifier": "/path/to/template.tmpl",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "Title": "My Document",
      "Author": "Bob"
    }
  }
}
```

##### Complex Template Example

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "text": "{{range .Items}}- {{.Name}}: {{.Price}}\n{{end}}Total: {{.Total}}",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "Items": [
        {"Name": "Apple", "Price": "$1.00"},
        {"Name": "Banana", "Price": "$0.50"}
      ],
      "Total": "$1.50"
    }
  }
}
```

### REST Endpoint (Convenience Interface)

**POST** `/v1/api/render`

```bash
curl -X POST http://localhost:8095/v1/api/render \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secret-key" \
  -d '{
    "template": "Hello {{.Name}}!",
    "data": {
      "Name": "World"
    }
  }'
```

Response:

```json
{
  "rendered": "Hello World!"
}
```

### Legacy Request Format

For backward compatibility, the service also accepts legacy field names:

```json
{
  "template": "Hello {{.Name}}!",
  "templateId": "/path/to/file.tmpl",
  "parameters": {
    "Name": "User"
  }
}
```

These are automatically converted to semantic fields:
- `template` → `text`
- `templateId` → `identifier`
- `parameters` → `templateParameters`

## Go Template Syntax

The service supports full Go template syntax:

### Variables

```
{{.Variable}}
{{.Object.Field}}
{{index .Array 0}}
```

### Conditionals

```
{{if .Condition}}
  True branch
{{else}}
  False branch
{{end}}
```

### Loops

```
{{range .Items}}
  Item: {{.}}
{{end}}
```

### Functions

```
{{.String | printf "%q"}}
{{len .Array}}
```

## State Tracking

The service includes built-in state management for all operations:

```bash
# List all tracked operations
curl http://localhost:8095/v1/api/state

# Get specific operation details
curl http://localhost:8095/v1/api/state/{operation-id}

# Get state statistics
curl http://localhost:8095/v1/api/state/stats
```

## Use Cases

### Configuration File Generation

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "text": "server:\n  host: {{.Host}}\n  port: {{.Port}}\n  debug: {{.Debug}}",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "Host": "localhost",
      "Port": 8080,
      "Debug": true
    }
  }
}
```

### Email Template Rendering

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "identifier": "/templates/welcome-email.tmpl",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "UserName": "Alice",
      "ActivationLink": "https://example.com/activate/abc123"
    }
  }
}
```

### Report Generation

```json
{
  "@context": "https://schema.org",
  "@type": "ReplaceAction",
  "object": {
    "@type": "DigitalDocument",
    "text": "# Report\n\n{{range .Sections}}\n## {{.Title}}\n{{.Content}}\n{{end}}",
    "encodingFormat": "text/template"
  },
  "replacer": {
    "@type": "PropertyValue",
    "value": {
      "Sections": [
        {"Title": "Summary", "Content": "..."},
        {"Title": "Details", "Content": "..."}
      ]
    }
  }
}
```

## Integration with EVE Ecosystem

### Registry Service

The service automatically registers with the EVE registry service if `REGISTRYSERVICE_API_URL` is configured.

### Workflow Orchestration

Use with the `when` workflow scheduler for template-based content generation:

```json
{
  "@context": "https://schema.org",
  "@type": "ItemList",
  "itemListElement": [
    {
      "@type": "ReplaceAction",
      "object": {
        "@type": "DigitalDocument",
        "text": "{{.Message}}"
      },
      "replacer": {
        "@type": "PropertyValue",
        "value": {"Message": "Step 1 complete"}
      }
    }
  ]
}
```

## Development

### Project Structure

```
templateservice/
└── cmd/templateservice/
    ├── main.go           # Service entry point and handlers
    └── rest_handlers.go  # REST endpoint handlers
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o templateservice ./cmd/templateservice
```

## Error Handling

The service returns detailed error messages for:
- Invalid template syntax
- Missing template files
- Template execution errors
- Invalid parameters

Example error response:

```json
{
  "error": "failed to parse template: template: template:1: unexpected \"}\" in operand"
}
```

## License

Apache License 2.0 - See LICENSE file for details.

Copyright 2025 evalgo.org

## Links

- [EVE Documentation](../when/docs/)
- [REST Endpoint Design](../when/REST_ENDPOINT_DESIGN.md)
- [Go Template Documentation](https://pkg.go.dev/text/template)
- [Schema.org Actions](https://schema.org/Action)
