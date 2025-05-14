package main

import (
	mcpgen "github.com/lyeslabs/mcpgen/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// This is your MCP server instance
var mcpServer = mcpgen.NewMCPServer()

func main() {
	sseHandler := server.NewSSEServer(mcpServer)
	sseHandler.Start(":8080")
}
