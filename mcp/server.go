package mcpgen

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPServer creates and returns an MCP server with all tools registered
func NewMCPServer() *server.MCPServer {
	// Create a new MCP server
	s := server.NewMCPServer(
		"MCP Server",
		"1.0.0",
	)

	// Register all tools
	s.AddTool(NewCreateTodoMCPTool(), CreateTodoHandler)
	s.AddTool(NewDeleteTodoByIdMCPTool(), DeleteTodoByIdHandler)
	s.AddTool(NewGetTodoByIdMCPTool(), GetTodoByIdHandler)
	s.AddTool(NewListTodosMCPTool(), ListTodosHandler)
	s.AddTool(NewUpdateTodoByIdMCPTool(), UpdateTodoByIdHandler)

	return s
}
