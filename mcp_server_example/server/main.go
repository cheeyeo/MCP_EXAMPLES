package main

import (
	"fmt"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// HelloArgs represent arguments of hello tool
type HelloArgs struct {
	Name string `json:"name" jsonschema:"required,description=The name to say hello to"`
}

func main() {
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	err := server.RegisterTool("hello", "Say hello to a person", func(args HelloArgs) (*mcp_golang.ToolResponse, error) {
		message := fmt.Sprintf("Hello %s!", args.Name)
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(message)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	select {}
}
