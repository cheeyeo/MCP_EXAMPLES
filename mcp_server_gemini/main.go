package main

// Example of using MCP with Gemini via Function Calls

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"google.golang.org/api/option"
)

func printResponse(resp *genai.GenerateContentResponse) {
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Println(part)
			}
		}
	}
	fmt.Println("---")
}

type Property struct {
	Description string `json:"description"`
	Type        string `json:"type"`
}

type GSchema struct {
	Schema     string              `json:"$schema"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
	Type       string              `json:"type"`
}

func getType(kind string) (genai.Type, error) {
	var gType genai.Type
	switch kind {
	case "object":
		gType = genai.TypeObject
	case "array":
		gType = genai.TypeArray
	case "string":
		gType = genai.TypeString
	case "number":
		gType = genai.TypeNumber
	case "integer":
		gType = genai.TypeInteger
	case "boolean":
		gType = genai.TypeBoolean
	default:
		return 0, fmt.Errorf("type not found in gemini Type: %s", kind)
	}

	return gType, nil
}

func (g GSchema) Convert() (*genai.Schema, error) {
	var parseErr error
	res := &genai.Schema{}

	gType, parseErr := getType(g.Type)
	if parseErr != nil {
		return nil, parseErr
	}
	res.Type = gType
	res.Required = g.Required

	// Convert properties to map of genai.Schema
	schemaProperties := map[string]*genai.Schema{}
	for k, v := range g.Properties {
		gType, parseErr := getType(v.Type)
		if parseErr != nil {
			return nil, parseErr
		}
		schemaProperties[k] = &genai.Schema{
			Description: v.Description,
			Type:        gType,
		}
	}
	res.Properties = schemaProperties

	return res, nil
}

func main() {
	// Load dotenv file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading dotenv file")
	}

	// Start the server process
	cmd := exec.Command("go", "run", "./server/main.go")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer cmd.Process.Kill()

	clientTransport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	client := mcp_golang.NewClient(clientTransport)
	if _, err := client.Initialize(context.Background()); err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	// List available tools on MCP server
	tools, err := client.ListTools(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v\n", err)
	}

	log.Println("Available tools:")
	// Create list of gemini tools
	geminiTools := []*genai.Tool{}

	for _, tool := range tools.Tools {
		desc := ""
		if tool.Description != nil {
			desc = *tool.Description
		}
		log.Printf("Tool: %s. Description: %s, Schema: %+v", tool.Name, desc, tool.InputSchema)

		// Cast inputschema from interface to map[string]any
		inputDict := tool.InputSchema.(map[string]any)
		jsonbody, err := json.Marshal(inputDict)
		if err != nil {
			log.Fatalf("error with converting tool.InputSchema - %s", err)
		}

		gschema := GSchema{}
		err = json.Unmarshal(jsonbody, &gschema)
		if err != nil {
			log.Fatalf("error with converting tool.InputSchema - %s", err)
		}

		geminiProperties, err := gschema.Convert()
		geminiTool := &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{{
				Name:        tool.Name,
				Description: *tool.Description,
				Parameters:  geminiProperties,
			}},
		}
		geminiTools = append(geminiTools, geminiTool)
	}

	ctx := context.Background()
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer geminiClient.Close()

	model := geminiClient.GenerativeModel("gemini-2.5-pro-preview-03-25")
	model.Tools = geminiTools
	model.SetTemperature(0.1)

	session := model.StartChat()
	// prompt := "Can you say hello to Col444 using my custom tool?"
	prompt := "What's the current Bitcoin price in RUB?"

	res, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("session.SendMessage: %v", err)
	}

	part := res.Candidates[0].Content.Parts[0]
	funcall, ok := part.(genai.FunctionCall)
	log.Printf("gemini funcall: %+v\n", funcall)
	if !ok {
		log.Fatalf("expected functioncall but received error:\n%v", part)
	}

	// Make actual call in MCP
	var geminiFunctionResponse map[string]any
	helloResp, err := client.CallTool(context.Background(), funcall.Name, funcall.Args)
	if err != nil {
		log.Printf("failed to call tool: %v\n", err)
		geminiFunctionResponse = map[string]any{"error": err}
	} else {
		log.Printf("Response: %v\n", helloResp.Content[0].TextContent.Text)
		geminiFunctionResponse = map[string]any{"response": helloResp.Content[0].TextContent.Text}
	}

	// Send resp back to gemini
	res, err = session.SendMessage(ctx, genai.FunctionResponse{
		Name:     funcall.Name,
		Response: geminiFunctionResponse,
	})
	if err != nil {
		log.Fatal(err)
	}

	printResponse(res)

	// Try testing of calling prompt
	promptArgs := map[string]interface{}{
		"Title": "Hello MCP",
	}
	resp, err := client.GetPrompt(ctx, "prompt_test", promptArgs)
	log.Printf("Prompt resp: %s", resp.Messages[0].Content.TextContent.Text)
}
