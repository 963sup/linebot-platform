// Package mcp provides the infrastructure adapter for Model Context Protocol clients.
package mcp

import sdk "github.com/modelcontextprotocol/go-sdk/mcp"

// NewClient constructs an MCP client without establishing a remote connection.
// Authentication and transport are configured by the caller at connection time.
func NewClient() *sdk.Client {
	return sdk.NewClient(&sdk.Implementation{
		Name: "linebot-platform",
		Version: "1.0.0",
	}, nil)
}
