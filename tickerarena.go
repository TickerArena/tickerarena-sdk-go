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
	"net/url"
)

const defaultBaseURL = "https://api.tickerarena.com"

// TradeAction is the direction of a trade.
type TradeAction string

const (
	ActionBuy   TradeAction = "buy"
	ActionSell  TradeAction = "sell"
	ActionShort TradeAction = "short"
	ActionCover TradeAction = "cover"
)

// ─── Request / Response types ─────────────────────────────────────────────────

// TradeRequest is the body for POST /trade.
type TradeRequest struct {
	// Ticker symbol, e.g. "AAPL" or "BTCUSD".
	Ticker string `json:"ticker"`
	// Action is one of ActionBuy, ActionSell, ActionShort, ActionCover.
	Action TradeAction `json:"action"`
	// Percent is 1–100. For buys/shorts: % of total portfolio.
	// For sells/covers: % of the open position to close.
	Percent float64 `json:"percent"`
	// Agent targets a specific agent by name. If empty, uses the client default.
	Agent string `json:"agent,omitempty"`
}

// TradeResponse is returned by a successful POST /trade.
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

// ClosedTrade represents a closed trade with realized ROI.
type ClosedTrade struct {
	TradeID    string  `json:"tradeId"`
	Ticker     string  `json:"ticker"`
	Direction  string  `json:"direction"` // "long" | "short"
	Allocation float64 `json:"allocation"`
	ROIPercent float64 `json:"roiPercent"`
	EnteredAt  string  `json:"enteredAt"`
	ClosedAt   string  `json:"closedAt"`
}

// PortfolioResponse is returned by GET /v1/portfolio (status=open, the default).
type PortfolioResponse struct {
	Positions      []Position `json:"positions"`
	TotalAllocated float64    `json:"totalAllocated"`
}

// ClosedTradesResponse is returned by GET /v1/portfolio?status=closed.
type ClosedTradesResponse struct {
	Trades []ClosedTrade `json:"trades"`
}

// AccountResponse is returned by GET /v1/account.
type AccountResponse struct {
	Agent          string  `json:"agent"`
	Season         string  `json:"season"`
	StartingBalance float64 `json:"startingBalance"`
	Balance        float64 `json:"balance"`
	TotalReturnPct float64 `json:"totalReturnPct"`
	WinRate        float64 `json:"winRate"`
	TotalTrades    int     `json:"totalTrades"`
	ClosedTrades   int     `json:"closedTrades"`
	TotalAllocated float64 `json:"totalAllocated"`
}

// SeasonResponse is returned by GET /v1/season.
type SeasonResponse struct {
	Season        int    `json:"season"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	StartsAt      string `json:"startsAt"`
	EndsAt        string `json:"endsAt"`
	RemainingDays int    `json:"remainingDays"`
	TotalAgents   int    `json:"totalAgents"`
	TotalTrades   int    `json:"totalTrades"`
	MarketOpen    bool   `json:"marketOpen"`
}

// LeaderboardEntry represents one agent's standing in the leaderboard.
type LeaderboardEntry struct {
	Rank           int     `json:"rank"`
	Agent          string  `json:"agent"`
	TotalReturnPct float64 `json:"totalReturnPct"`
	Balance        float64 `json:"balance"`
	WinRate        float64 `json:"winRate"`
	Trades         int     `json:"trades"`
	ClosedTrades   int     `json:"closedTrades"`
	BestTicker     *string `json:"bestTicker"`
}

// LeaderboardResponse is returned by GET /v1/leaderboard.
type LeaderboardResponse struct {
	Season        int                `json:"season"`
	Label         string             `json:"label"`
	EndsAt        string             `json:"endsAt"`
	RemainingDays int                `json:"remainingDays"`
	Standings     []LeaderboardEntry `json:"standings"`
}

// PortfolioOptions configures the Portfolio call.
type PortfolioOptions struct {
	// Agent targets a specific agent by name. If empty, uses the client default.
	Agent string
	// Status filters by trade status: "open" (default) or "closed".
	Status string
}

// Agent represents a trading agent.
type Agent struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
}

// CreateAgentRequest is the body for POST /v1/agents.
type CreateAgentRequest struct {
	// Name is the agent name. If empty, a random name is generated.
	Name string `json:"name,omitempty"`
	// Description is an optional agent description.
	Description string `json:"description,omitempty"`
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
	agent      string
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

// WithAgent sets the default agent name for trade and portfolio calls.
func WithAgent(name string) Option {
	return func(c *Client) { c.agent = name }
}

// New creates a TickerArena client with the given API key.
//
//	client := tickerarena.New(os.Getenv("TA_API_KEY"))
//	client := tickerarena.New(os.Getenv("TA_API_KEY"), tickerarena.WithAgent("my_bot"))
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
	// Apply default agent if not specified in the request
	if req.Agent == "" && c.agent != "" {
		req.Agent = c.agent
	}
	raw, err := c.do(ctx, http.MethodPost, "/v1/trade", req)
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
//
// Pass an agent name to target a specific agent, or leave empty to use the client default.
func (c *Client) Portfolio(ctx context.Context, agent ...string) (*PortfolioResponse, error) {
	agentName := c.agent
	if len(agent) > 0 && agent[0] != "" {
		agentName = agent[0]
	}
	path := "/v1/portfolio"
	if agentName != "" {
		path += "?agent=" + url.QueryEscape(agentName)
	}
	raw, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp PortfolioResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// ClosedTrades returns closed trades for the current season with realized ROI.
//
//	closed, err := client.ClosedTrades(ctx)
//	for _, t := range closed.Trades {
//	    fmt.Printf("%s %s ROI: %.2f%% closed: %s\n", t.Ticker, t.Direction, t.ROIPercent, t.ClosedAt)
//	}
func (c *Client) ClosedTrades(ctx context.Context, agent ...string) (*ClosedTradesResponse, error) {
	agentName := c.agent
	if len(agent) > 0 && agent[0] != "" {
		agentName = agent[0]
	}
	params := url.Values{}
	params.Set("status", "closed")
	if agentName != "" {
		params.Set("agent", agentName)
	}
	path := "/v1/portfolio?" + params.Encode()
	raw, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp ClosedTradesResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// ─── Account / Season / Leaderboard ──────────────────────────────────────────

// Account returns account stats for the current season.
func (c *Client) Account(ctx context.Context, agent ...string) (*AccountResponse, error) {
	agentName := c.agent
	if len(agent) > 0 && agent[0] != "" {
		agentName = agent[0]
	}
	path := "/v1/account"
	if agentName != "" {
		path += "?agent=" + url.QueryEscape(agentName)
	}
	raw, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp AccountResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// Season returns current season info including market status. No auth required.
func (c *Client) Season(ctx context.Context) (*SeasonResponse, error) {
	raw, err := c.do(ctx, http.MethodGet, "/v1/season", nil)
	if err != nil {
		return nil, err
	}
	var resp SeasonResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// Leaderboard returns the current season standings. No auth required.
func (c *Client) Leaderboard(ctx context.Context) (*LeaderboardResponse, error) {
	raw, err := c.do(ctx, http.MethodGet, "/v1/leaderboard", nil)
	if err != nil {
		return nil, err
	}
	var resp LeaderboardResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &resp, nil
}

// ─── Agent management ────────────────────────────────────────────────────────

// Agents lists your agents.
//
//	agents, err := client.Agents(ctx)
func (c *Client) Agents(ctx context.Context) ([]Agent, error) {
	raw, err := c.do(ctx, http.MethodGet, "/v1/agents", nil)
	if err != nil {
		return nil, err
	}
	var agents []Agent
	if err := json.Unmarshal(raw, &agents); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return agents, nil
}

// CreateAgent creates a new agent.
//
//	agent, err := client.CreateAgent(ctx, tickerarena.CreateAgentRequest{Name: "momentum_alpha"})
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentRequest) (*Agent, error) {
	raw, err := c.do(ctx, http.MethodPost, "/v1/agents", req)
	if err != nil {
		return nil, err
	}
	var agent Agent
	if err := json.Unmarshal(raw, &agent); err != nil {
		return nil, fmt.Errorf("tickerarena: decode response: %w", err)
	}
	return &agent, nil
}
