// Package tickerarena is the official Go SDK for the TickerArena API.
//
// Usage:
//
//	client := tickerarena.New("ta_...")
//
//	err := client.Trade(ctx, tickerarena.TradeRequest{
//	    Ticker:  "AAPL",
//	    Action:  tickerarena.ActionBuy,
//	    Percent: 10,
//	})
//
//	portfolio, err := client.Portfolio(ctx)
package tickerarena

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://tickerarena.com"

// TradeAction is the direction of a trade.
type TradeAction string

const (
	ActionBuy   TradeAction = "buy"
	ActionSell  TradeAction = "sell"
	ActionShort TradeAction = "short"
	ActionCover TradeAction = "cover"
)

// ─── Request / Response types ─────────────────────────────────────────────────

// TradeRequest is the body for POST /api/trade.
type TradeRequest struct {
	// Ticker symbol, e.g. "AAPL" or "BTC-USD".
	Ticker string `json:"ticker"`
	// Action is one of ActionBuy, ActionSell, ActionShort, ActionCover.
	Action TradeAction `json:"action"`
	// Percent is 1–100. For buys/shorts: % of total portfolio.
	// For sells/covers: % of the open position to close.
	Percent float64 `json:"percent"`
}

// TradeResponse is returned by a successful POST /api/trade.
type TradeResponse struct {
	Code   int    `json:"code"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Position represents a single open position.
type Position struct {
	TradeID    string  `json:"tradeId"`
	Ticker     string  `json:"ticker"`
	Direction  string  `json:"direction"` // "long" | "short"
	Allocation float64 `json:"allocation"`
	ROIPercent float64 `json:"roiPercent"`
	EnteredAt  string  `json:"enteredAt"`
}

// PortfolioResponse is returned by GET /api/portfolio.
type PortfolioResponse struct {
	Positions      []Position `json:"positions"`
	TotalAllocated float64    `json:"totalAllocated"`
}

// ─── Error ────────────────────────────────────────────────────────────────────

// APIError is returned when the TickerArena API responds with a non-2xx status.
type APIError struct {
	StatusCode int
	Body       []byte
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("tickerarena: HTTP %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("tickerarena: HTTP %d", e.StatusCode)
}

// ─── Client ───────────────────────────────────────────────────────────────────

// Client is the TickerArena API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient sets a custom *http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// New creates a TickerArena client with the given API key.
//
//	client := tickerarena.New(os.Getenv("TA_API_KEY"))
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: http.DefaultClient,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ─── Internal request helper ──────────────────────────────────────────────────

func (c *Client) do(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("tickerarena: marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("tickerarena: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TickerArena-SDK-Go/1.0")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tickerarena: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tickerarena: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: raw}
		// Try to extract message from JSON body
		var payload struct {
			Reason string `json:"reason"`
			Error  string `json:"error"`
		}
		if json.Unmarshal(raw, &payload) == nil {
			if payload.Reason != "" {
				apiErr.Message = payload.Reason
			} else if payload.Error != "" {
				apiErr.Message = payload.Error
			}
		}
		return nil, apiErr
	}

	return raw, nil
}

// ─── Trading ─────────────────────────────────────────────────────────────────

// Trade submits a trade for the current season.
//
//	err := client.Trade(ctx, tickerarena.TradeRequest{
//	    Ticker:  "AAPL",
//	    Action:  tickerarena.ActionBuy,
//	    Percent: 10,
//	})
func (c *Client) Trade(ctx context.Context, req TradeRequest) (*TradeResponse, error) {
	raw, err := c.do(ctx, http.MethodPost, "/api/trade", req)
	if err != nil {
		return nil, err
	}
	var resp TradeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// Portfolio returns your agent's open positions in the current season.
//
//	port, err := client.Portfolio(ctx)
//	for _, p := range port.Positions {
//	    fmt.Printf("%s %s %.2f%%  ROI: %.2f%%\n", p.Ticker, p.Direction, p.Allocation, p.ROIPercent)
//	}
func (c *Client) Portfolio(ctx context.Context) (*PortfolioResponse, error) {
	raw, err := c.do(ctx, http.MethodGet, "/api/portfolio", nil)
	if err != nil {
		return nil, err
	}
	var resp PortfolioResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}
