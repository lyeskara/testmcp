package mcpgen

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// Input Schema for the UpdateTodoById tool
const UpdateTodoByIdInputSchema = `{
  "properties": {
    "body": {
      "properties": {
        "description": {
          "description": "Optional detailed description of the todo item.",
          "examples": [
            "Confirm bookings and pack."
          ],
          "type": [
            "string",
            "null"
          ]
        },
        "dueDate": {
          "description": "Optional due date for the todo item.",
          "examples": [
            "2025-06-14"
          ],
          "format": "date",
          "type": [
            "string",
            "null"
          ]
        },
        "status": {
          "description": "Current status of the todo item.",
          "enum": [
            "pending",
            "in-progress",
            "completed"
          ],
          "examples": [
            "in-progress"
          ],
          "type": "string"
        },
        "title": {
          "description": "The main content of the todo item.",
          "examples": [
            "Finalize weekend trip plans"
          ],
          "type": "string"
        }
      },
      "type": "object"
    },
    "todoId": {
      "format": "uuid",
      "type": "string"
    }
  },
  "required": [
    "todoId",
    "body"
  ],
  "type": "object"
}`

// Response Template for the UpdateTodoById tool
const UpdateTodoByIdResponseTemplate = `# API Response Information

Below is the response from an API call. To help you understand the data, I've provided:

1. A detailed description of all fields in the response structure
2. The complete API response

## Success Response Structure

> Content-Type: application/json

- **createdAt**: Timestamp of when the todo item was created. (Type: string)
- **id**: Unique identifier for the todo item. (Type: string)
- **status**: Current status of the todo item. (Type: string)
- **title**: The main content of the todo item. (Type: string)

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
// Method: PUT
// Headers:
//   Content-Type: application/json

// NewUpdateTodoByIdMCPTool creates the MCP Tool instance for UpdateTodoById
func NewUpdateTodoByIdMCPTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"UpdateTodoById",
		"Update an existing todo item - Modifies an existing todo item by its ID.",
		[]byte(UpdateTodoByIdInputSchema),
	)
}

// UpdateTodoByIdHandler is the handler function for the UpdateTodoById tool.
// This function is automatically generated. Users should implement the actual
// logic within this function body to integrate with backend APIs.
// You can generate types, http client and helpers for parsing request params to facilitate the implementation.
func UpdateTodoByIdHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	return nil, fmt.Errorf("%s not implemented", "UpdateTodoById")
}
