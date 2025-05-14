package mcpgen

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// Input Schema for the DeleteTodoById tool
const DeleteTodoByIdInputSchema = `{
  "properties": {
    "todoId": {
      "format": "uuid",
      "type": "string"
    }
  },
  "required": [
    "todoId"
  ],
  "type": "object"
}`

// Response Template for the DeleteTodoById tool
const DeleteTodoByIdResponseTemplate = `# API Response Information

Below is the response from an API call. To help you understand the data, I've provided:

1. A detailed description of all fields in the response structure
2. The complete API response

## Error Response Structure

> Content-Type: application/json

- **code**: An application-specific error code. (Type: integer)
- **details**: Optional array of specific field validation errors. (Type: array)
  - **details[].field**:  (Type: string)
  - **details[].issue**:  (Type: string)
- **message**: A human-readable description of the error. (Type: string)

## Original Response

`

// URL: https://api.example.com/v1/todos/{todoId}
// Method: DELETE

// NewDeleteTodoByIdMCPTool creates the MCP Tool instance for DeleteTodoById
func NewDeleteTodoByIdMCPTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"DeleteTodoById",
		"Delete a todo item - Removes a todo item by its ID.",
		[]byte(DeleteTodoByIdInputSchema),
	)
}

// DeleteTodoByIdHandler is the handler function for the DeleteTodoById tool.
// This function is automatically generated. Users should implement the actual
// logic within this function body to integrate with backend APIs.
// You can generate types, http client and helpers for parsing request params to facilitate the implementation.
func DeleteTodoByIdHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	return nil, fmt.Errorf("%s not implemented", "DeleteTodoById")
}
