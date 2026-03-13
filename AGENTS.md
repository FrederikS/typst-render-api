# typst-render-api

## Project Overview

Generic HTTP API that wraps the Typst typesetting system. Accepts a template name and JSON data, renders a PDF, and returns it.

## Technology Stack

- Go 1.21+
- chi router (lightweight)
- Docker + Kubernetes

## Configuration

| Env Variable | Required | Default | Description |
|--------------|----------|---------|-------------|
| `TEMPLATE_DIR` | Yes | - | Path to templates directory |
| `TYPST_PATH` | No | `/usr/local/bin/typst` | Path to typst binary |
| `PORT` | No | `8080` | HTTP server port |

## API Design

### POST /api/templates/{name}/render

Renders a template with the provided data.

**Path Parameter:**
- `name` - Template name without extension (e.g., `letter` finds `letter.typ`)

**Request Body:**
```json
{
  "company": { "name": "Acme Corp" },
  "position": "Software Engineer"
}
```

**Response:** PDF binary (`application/pdf`)

**Status Codes:**
- `200` - Success
- `400` - Invalid request
- `404` - Template not found
- `500` - Render error

### GET /health

Returns `{"status": "ok"}` with `200 OK`.

## Template Resolution

Given `name` from URL:
1. Template file: `{TEMPLATE_DIR}/{name}.typ` (required)
2. Static data file: `{TEMPLATE_DIR}/{name}.json` (optional)

## Data Merging

If `{name}.json` exists, merge it with request body:
- Both objects are combined
- Request body values override static file values for duplicate keys

## Typst Invocation

```bash
typst compile <template_path> <output.pdf> \
  --input data='<merged_json>' \
  --root <TEMPLATE_DIR>
```

## Project Structure

```
typst-render-api/
├── main.go
├── config.go
├── handler.go
├── handler_test.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yaml
├── .dockerignore
├── .gitignore
└── k8s/
    ├── deployment.yaml
    └── service.yaml
```

## Implementation

### config.go
- Read and validate `TEMPLATE_DIR` at startup
- Fail if directory doesn't exist

### handler.go
- Parse `{name}` from URL path
- Check `{TEMPLATE_DIR}/{name}.typ` exists
- Read optional `{TEMPLATE_DIR}/{name}.json`
- Merge static + request data
- Spawn typst subprocess
- Return PDF or error

### main.go
- Setup chi router
- Mount handlers
- Start server on `PORT`

## Docker

Multi-stage build:
- Builder: Go 1.21-alpine
- Runtime: alpine:3.19 with typst installed

Expose port 8080.

## Kubernetes

- Deployment with `TEMPLATE_DIR: /templates` env
- Volume mount for templates (ConfigMap or PVC)
- ClusterIP Service (internal only)
