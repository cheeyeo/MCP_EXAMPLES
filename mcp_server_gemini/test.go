package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/generative-ai-go/genai"
)

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
	// map[$schema:https://json-schema.org/draft/2020-12/schema properties:map[currency:map[description:The currency to get the Bitcoin price in (USD type:string]] required:[currency] type:object]

	dict := make(map[string]interface{})
	dict["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	dict["properties"] = map[string]interface{}{"currency": map[string]interface{}{"description": "The currency to get the Bitcoin price in (USD, EUR, GBP, etc)", "type": "string"}}
	dict["required"] = []string{"currency"}
	dict["type"] = "object"

	jsonbody, err := json.Marshal(dict)
	if err != nil {
		panic(err)
	}
	log.Printf("JSONBODY: %+s\n", jsonbody)
	gschema := GSchema{}
	err = json.Unmarshal(jsonbody, &gschema)
	if err != nil {
		panic(err)
	}

	log.Printf("SCHEMA: %+v\n", gschema)

	res, err := gschema.Convert()
	fmt.Printf("RES: %+v\n", res)

}
