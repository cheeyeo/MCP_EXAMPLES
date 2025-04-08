package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// HelloArgs represent arguments of hello tool
type HelloArgs struct {
	Name string `json:"name" jsonschema:"required,description=The name to say hello to"`
}

type BitcoinPriceArguments struct {
	Currency string `json:"currency" jsonschema:"required,description=The currency to get the Bitcoin price in (USD, EUR, GBP, JPY, AUD, CAD, CHF, CNY, KRW, RUB etc)"`
}

type CoinGeckoResponse struct {
	Bitcoin struct {
		USD float64 `json:"usd"`
		EUR float64 `json:"eur"`
		GBP float64 `json:"gbp"`
		JPY float64 `json:"jpy"`
		AUD float64 `json:"aud"`
		CAD float64 `json:"cad"`
		CHF float64 `json:"chf"`
		CNY float64 `json:"cny"`
		KRW float64 `json:"krw"`
		RUB float64 `json:"rub"`
	} `json:"bitcoin"`
}

type Content struct {
	Title       string  `json:"title" jsonschema:"required,description=The title to submit"`
	Description *string `json:"description" jsonschema:"description=The description to submit"`
}

func getBitcoinPrice(currency string) (float64, error) {
	log.Printf("INSIDE GET BITCOIN PRICE - CURRENCY - %s", currency)
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make request to CoinGecko API
	resp, err := client.Get("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd,eur,gbp,jpy,aud,cad,chf,cny,krw,rub")
	if err != nil {
		return 0, fmt.Errorf("error making request to CoinGecko API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("error reading response body: %w", err)
	}

	// Parse JSON response
	var coinGeckoResp CoinGeckoResponse
	err = json.Unmarshal(body, &coinGeckoResp)
	if err != nil {
		return 0, fmt.Errorf("error parsing JSON response: %w", err)
	}

	// Get price for requested currency
	var price float64
	switch currency {
	case "USD", "usd":
		price = coinGeckoResp.Bitcoin.USD
	case "EUR", "eur":
		price = coinGeckoResp.Bitcoin.EUR
	case "GBP", "gbp":
		price = coinGeckoResp.Bitcoin.GBP
	case "JPY", "jpy":
		price = coinGeckoResp.Bitcoin.JPY
	case "AUD", "aud":
		price = coinGeckoResp.Bitcoin.AUD
	case "CAD", "cad":
		price = coinGeckoResp.Bitcoin.CAD
	case "CHF", "chf":
		price = coinGeckoResp.Bitcoin.CHF
	case "CNY", "cny":
		price = coinGeckoResp.Bitcoin.CNY
	case "KRW", "krw":
		price = coinGeckoResp.Bitcoin.KRW
	case "RUB", "rub":
		price = coinGeckoResp.Bitcoin.RUB
	default:
		return 0, fmt.Errorf("unsupported currency: %s", currency)
	}

	return price, nil
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

	// Register the bitcoin_price tool
	err = server.RegisterTool("bitcoin_price", "Get the latest Bitcoin price in various currencies", func(arguments BitcoinPriceArguments) (*mcp_golang.ToolResponse, error) {
		log.Printf("received request for bitcoin_price tool with currency: %s", arguments.Currency)

		currency := arguments.Currency
		if currency == "" {
			currency = "USD"
		}

		// Call CoinGecko API to get latest Bitcoin price
		price, err := getBitcoinPrice(currency)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error fetching Bitcoin price: %v", err))), err
		}

		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("The current Bitcoin price in %s is %.2f (as of %s)",
			currency,
			price,
			time.Now().Format(time.RFC1123)))), nil
	})
	if err != nil {
		log.Fatalf("error registering bitcoin_price tool: %v", err)
	}

	err = server.RegisterPrompt("prompt_test", "This is a test prompt", func(arguments Content) (*mcp_golang.PromptResponse, error) {
		log.Println("Received request for prompt_test")

		return mcp_golang.NewPromptResponse("description", mcp_golang.NewPromptMessage(mcp_golang.NewTextContent(fmt.Sprintf("Hello, %s!", arguments.Title)), mcp_golang.RoleUser)), nil
	})
	if err != nil {
		log.Fatalf("error registering prompt_test prompt: %v", err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	select {}
}
