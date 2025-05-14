package mcpgen

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// Input Schema for the ListTodos tool
const ListTodosInputSchema = `{
  "properties": {
    "limit": {
      "default": 20,
      "format": "int32",
      "minimum": 1,
      "type": "integer"
    },
    "offset": {
      "default": 0,
      "format": "int32",
      "minimum": 0,
      "type": "integer"
    },
    "status": {
      "enum": [
        "pending",
        "completed",
        "in-progress"
      ],
      "type": "string"
    }
  },
  "type": "object"
}`

// Response Template for the ListTodos tool
const ListTodosResponseTemplate = `# API Response Information

Below is the response from an API call. To help you understand the data, I've provided:

1. A detailed description of all fields in the response structure
2. The complete API response

## Success Response Structure

> Content-Type: application/json

- **items**: Array of items (Type: array)

## Error Response Structure

> Content-Type: application/json

- **code**: An application-specific error code. (Type: integer)
- **details**: Optional array of specific field validation errors. (Type: array)
  - **details[].field**:  (Type: string)
  - **details[].issue**:  (Type: string)
- **message**: A human-readable description of the error. (Type: string)

## Original Response

`

// URL: https://api.example.com/v1/todos
// Method: GET

// NewListTodosMCPTool creates the MCP Tool instance for ListTodos
func NewListTodosMCPTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"ListTodos",
		"List all todo items - Retrieves a list of todo items, optionally filtered by status.",
		[]byte(ListTodosInputSchema),
	)
}

// ListTodosHandler is the handler function for the ListTodos tool.
// This function is automatically generated. Users should implement the actual
// logic within this function body to integrate with backend APIs.
// You can generate types, http client and helpers for parsing request params to facilitate the implementation.
func ListTodosHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	return nil, fmt.Errorf("%s not implemented", "ListTodos")
}
