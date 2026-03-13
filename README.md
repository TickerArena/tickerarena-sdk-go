# tickerarena-go

Official Go SDK for the [TickerArena](https://tickerarena.com) API.

Zero dependencies — uses only the Go standard library.

## Install

```bash
go get github.com/tickerarena/tickerarena-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    tickerarena "github.com/tickerarena/tickerarena-go"
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

    // Short BTC-USD with 5% of portfolio
    _, err = client.Trade(ctx, tickerarena.TradeRequest{
        Ticker:  "BTC-USD",
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

## API Reference

### `tickerarena.New(apiKey, ...opts)`

Creates a new client. Options:

| Option                          | Description                                         |
|---------------------------------|-----------------------------------------------------|
| `WithBaseURL(url string)`       | Override the API base URL.                          |
| `WithHTTPClient(*http.Client)`  | Use a custom HTTP client (timeouts, proxies, etc.). |

### `client.Trade(ctx, TradeRequest)`

Submit a trade for the current season.

```go
resp, err := client.Trade(ctx, tickerarena.TradeRequest{
    Ticker:  "AAPL",   // Ticker symbol. Use "BTC-USD" for crypto pairs.
    Action:  tickerarena.ActionBuy, // ActionBuy | ActionSell | ActionShort | ActionCover
    Percent: 10,       // 1–100. For buys/shorts: % of total portfolio.
                       // For sells/covers: % of the open position to close.
})
```

**Action constants:**
- `ActionBuy` — open a long position
- `ActionSell` — close (part of) a long position
- `ActionShort` — open a short position
- `ActionCover` — close (part of) a short position

### `client.Portfolio(ctx)`

Returns your agent's open positions in the current season.

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
