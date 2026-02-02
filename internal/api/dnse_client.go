package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// DNSEClient is the REST API client for DNSE LightSpeed API.
type DNSEClient struct {
	baseURL string
	auth    *DNSEAuthService
	client  *http.Client
}

// NewDNSEClient creates a new DNSE API client.
func NewDNSEClient(baseURL string, auth *DNSEAuthService) *DNSEClient {
	return &DNSEClient{
		baseURL: baseURL,
		auth:    auth,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DailyBar represents a daily OHLCV bar from DNSE.
type DailyBar struct {
	Symbol   string    `json:"symbol"`
	Date     time.Time `json:"date"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Volume   int64     `json:"volume"`
	Turnover float64   `json:"turnover"`
}

// IntradayBar represents an intraday bar (1m, 5m, 15m, etc.).
type IntradayBar struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
}

// SymbolInfo represents current symbol information.
type SymbolInfo struct {
	Symbol        string    `json:"symbol"`
	LastPrice     float64   `json:"lastPrice"`
	Change        float64   `json:"change"`
	ChangePercent float64   `json:"changePercent"`
	Ceiling       float64   `json:"ceiling"`
	Floor         float64   `json:"floor"`
	Reference     float64   `json:"reference"`
	BidPrice      float64   `json:"bidPrice"`
	AskPrice      float64   `json:"askPrice"`
	Volume        int64     `json:"volume"`
	Timestamp     time.Time `json:"timestamp"`
}

// VNIndexBar represents VN-Index daily data.
type VNIndexBar struct {
	Date   time.Time `json:"date"`
	Value  float64   `json:"value"`
	Change float64   `json:"change"`
	Volume int64     `json:"volume"`
}

// GetHistoricalDailyBars fetches daily OHLCV bars for a symbol.
func (c *DNSEClient) GetHistoricalDailyBars(symbol string, from, to time.Time) ([]DailyBar, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/market-data/daily/%s?from=%s&to=%s",
		c.baseURL, symbol, from.Format("2006-01-02"), to.Format("2006-01-02"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var bars []DailyBar
	if err := json.NewDecoder(resp.Body).Decode(&bars); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return bars, nil
}

// GetHistoricalIntradayBars fetches intraday bars (1m, 5m, 15m).
func (c *DNSEClient) GetHistoricalIntradayBars(symbol, interval string, from, to time.Time) ([]IntradayBar, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/market-data/intraday/%s/%s?from=%s&to=%s",
		c.baseURL, interval, symbol,
		from.Format(time.RFC3339), to.Format(time.RFC3339))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var bars []IntradayBar
	if err := json.NewDecoder(resp.Body).Decode(&bars); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return bars, nil
}

// GetSymbolInfo fetches current symbol information including ceiling/floor prices.
func (c *DNSEClient) GetSymbolInfo(symbol string) (*SymbolInfo, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/market-data/symbol-info/%s", c.baseURL, symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var info SymbolInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// GetVNIndexDaily fetches VN-Index historical daily data.
func (c *DNSEClient) GetVNIndexDaily(from, to time.Time) ([]VNIndexBar, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/market-data/index/vnindex?from=%s&to=%s",
		c.baseURL, from.Format("2006-01-02"), to.Format("2006-01-02"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var bars []VNIndexBar
	if err := json.NewDecoder(resp.Body).Decode(&bars); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return bars, nil
}

// GetVNIndexCurrent fetches current VN-Index value.
func (c *DNSEClient) GetVNIndexCurrent() (float64, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return 0, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/market-data/index/vnindex/current", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		Value float64 `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Value, nil
}

// ConvertToOHLCV converts DailyBar to data.OHLCV format.
func (b *DailyBar) ToOHLCV() data.OHLCV {
	return data.OHLCV{
		Timestamp: b.Date,
		Open:      b.Open,
		High:      b.High,
		Low:       b.Low,
		Close:     b.Close,
		Volume:    float64(b.Volume),
	}
}

// ConvertToOHLCV converts IntradayBar to data.OHLCV format.
func (b *IntradayBar) ToOHLCV() data.OHLCV {
	return data.OHLCV{
		Timestamp: b.Timestamp,
		Open:      b.Open,
		High:      b.High,
		Low:       b.Low,
		Close:     b.Close,
		Volume:    float64(b.Volume),
	}
}
