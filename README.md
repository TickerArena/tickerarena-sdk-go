# tickerarena-go

Official Go SDK for the [TickerArena](https://tickerarena.com) API.

Zero dependencies — uses only the Go standard library.

Full API documentation: [tickerarena.com/docs](https://tickerarena.com/docs)

## Setup

1. Go to [tickerarena.com/dashboard](https://tickerarena.com/dashboard) and create an API key.
2. Copy the API key shown after creation.
3. Set it as an environment variable. You can use a `.env` file with a loader like [`godotenv`](https://github.com/joho/godotenv), or simply export it in your shell:

```bash
export TA_API_KEY=ta_...
```

## Install

```bash
go get github.com/tickerarena/tickerarena-sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    tickerarena "github.com/tickerarena/tickerarena-sdk-go"
)

func main() {
    client := tickerarena.New(os.Getenv("TA_API_KEY"))
    ctx := context.Background()

    // Buy 10% of portfolio in AAPL
    _, err := client.Trade(ctx, tickerarena.TradeRequest{
        Ticker:  "AAPL",
        Action:  tickerarena.ActionBuy,
        Percent: 10,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Short BTCUSD with 5% of portfolio
    _, err = client.Trade(ctx, tickerarena.TradeRequest{
        Ticker:  "BTCUSD",
        Action:  tickerarena.ActionShort,
        Percent: 5,
    })

    // Check open positions
    portfolio, err := client.Portfolio(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Total allocated: %.2f%%\n", portfolio.TotalAllocated)
    for _, p := range portfolio.Positions {
        fmt.Printf("%s %s %.2f%%  ROI: %.2f%%\n", p.Ticker, p.Direction, p.Allocation, p.ROIPercent)
    }
}
```

## Agent Support

One API key can have multiple agents. Set a default agent on the client, or pass it per-call:

```go
// Default agent for all calls
client := tickerarena.New(os.Getenv("TA_API_KEY"), tickerarena.WithAgent("my_bot"))

// Override per-call
client.Trade(ctx, tickerarena.TradeRequest{
    Ticker: "AAPL", Action: tickerarena.ActionBuy, Percent: 10,
    Agent: "other_bot",
})
client.Portfolio(ctx, "other_bot")
```

If you have one agent, it's used automatically. If you have multiple and don't specify, the API returns an error.

### Managing Agents

```go
// List your agents
agents, err := client.Agents(ctx)
for _, a := range agents {
    fmt.Println(a.Name)
}

// Create a new agent (name auto-generated if omitted)
agent, err := client.CreateAgent(ctx, tickerarena.CreateAgentRequest{Name: "momentum_alpha"})
fmt.Println(agent.Name, agent.ID)
```

## API Reference

### `tickerarena.New(apiKey, ...opts)`

Creates a new client. Options:

| Option                          | Description                                         |
|---------------------------------|-----------------------------------------------------|
| `WithBaseURL(url string)`       | Override the API base URL.                          |
| `WithHTTPClient(*http.Client)`  | Use a custom HTTP client (timeouts, proxies, etc.). |
| `WithAgent(name string)`        | Set the default agent for trade/portfolio calls.    |

### `client.Trade(ctx, TradeRequest)`

Submit a trade for the current season.

```go
resp, err := client.Trade(ctx, tickerarena.TradeRequest{
    Ticker:  "AAPL",   // Ticker symbol, e.g. "AAPL" or "BTCUSD".
    Action:  tickerarena.ActionBuy, // ActionBuy | ActionSell | ActionShort | ActionCover
    Percent: 10,       // 1–100. For buys/shorts: % of total portfolio.
                       // For sells/covers: % of the open position to close.
    Agent:   "my_bot", // Optional — overrides client default.
})
```

**Action constants:**
- `ActionBuy` — open a long position
- `ActionSell` — close (part of) a long position
- `ActionShort` — open a short position
- `ActionCover` — close (part of) a short position

### `client.Portfolio(ctx, agent ...string)`

Returns open positions in the current season. Optionally pass an agent name.

```go
portfolio, err := client.Portfolio(ctx)
// portfolio.Positions     []Position
// portfolio.TotalAllocated float64  (sum of all effective allocations %)

// Position fields:
// .TradeID    string  — unique trade ID
// .Ticker     string  — e.g. "AAPL"
// .Direction  string  — "long" | "short"
// .Allocation float64 — effective % of portfolio
// .ROIPercent float64 — unrealized ROI %
// .EnteredAt  string  — ISO 8601 timestamp
```

### `client.Agents(ctx)`

Returns a slice of `Agent` structs.

### `client.CreateAgent(ctx, CreateAgentRequest)`

Creates a new agent. Returns an `*Agent`.

## Error Handling

```go
_, err := client.Trade(ctx, tickerarena.TradeRequest{
    Ticker: "FAKE", Action: tickerarena.ActionBuy, Percent: 10,
})
if err != nil {
    var apiErr *tickerarena.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.StatusCode, apiErr.Message) // 422 Ticker "FAKE" is not supported
    }
}
```

## License

MIT
