package mcpgen

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
)

// Input Schema for the CreateTodo tool
const CreateTodoInputSchema = `{
  "properties": {
    "body": {
      "description": "A list of todo items.",
      "properties": {
        "list": {
          "items": {
            "properties": {
              "createdAt": {
                "description": "Timestamp of when the todo item was created.",
                "examples": [
                  "2025-05-09T18:12:54Z"
                ],
                "format": "date-time",
                "readOnly": true,
                "type": "string"
              },
              "id": {
                "description": "Unique identifier for the todo item.",
                "examples": [
                  "d290f1ee-6c54-4b01-90e6-d701748f0851"
                ],
                "format": "uuid",
                "type": "string"
              },
              "status": {
                "default": "pending",
                "description": "Current status of the todo item.",
                "enum": [
                  "pending",
                  "in-progress",
                  "completed"
                ],
                "examples": [
                  "pending"
                ],
                "type": "string"
              },
              "title": {
                "description": "The main content of the todo item.",
                "examples": [
                  "Buy groceries"
                ],
                "type": "string"
              }
            },
            "required": [
              "id",
              "title",
              "status",
              "createdAt",
              "updatedAt"
            ],
            "type": "object"
          },
          "type": "array"
        }
      },
      "type": "object"
    }
  },
  "required": [
    "body"
  ],
  "type": "object"
}`

// Response Template for the CreateTodo tool
const CreateTodoResponseTemplate = `# API Response Information

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

// URL: https://api.example.com/v1/todos
// Method: POST
// Headers:
//   Content-Type: application/json

// NewCreateTodoMCPTool creates the MCP Tool instance for CreateTodo
func NewCreateTodoMCPTool() mcp.Tool {
	return mcp.NewToolWithRawSchema(
		"CreateTodo",
		"Create a new todo item - Adds a new item to the todo list.",
		[]byte(CreateTodoInputSchema),
	)
}

// CreateTodoHandler is the handler function for the CreateTodo tool.
// This function is automatically generated. Users should implement the actual
// logic within this function body to integrate with backend APIs.
// You can generate types, http client and helpers for parsing request params to facilitate the implementation.
func CreateTodoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	return nil, fmt.Errorf("%s not implemented", "CreateTodo")
}
