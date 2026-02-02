package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OHLCCandle represents a single OHLC candle from DNSE API.
type OHLCCandle struct {
	Date   string  `json:"date"` // Format: "YYYY-MM-DD" or timestamp
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// OHLCResponse represents the DNSE OHLC API response.
type OHLCResponse struct {
	Symbol string        `json:"symbol"`
	Data   []OHLCCandle `json:"data"`
}

// GetOHLC fetches OHLC candle data from DNSE API.
// resolution: "1" for daily, "60" for hourly, etc.
// Returns last N candles (typically 14+ for ATR calculation).
func (c *DNSEClient) GetOHLC(symbol, resolution string) ([]OHLCCandle, error) {
	token, err := c.auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	url := fmt.Sprintf("%s/quotes/krx/mdss/v2/ohlc/stock/%s/%s",
		c.baseURL, resolution, symbol)

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

	var result OHLCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// ParseOHLCDate parses DNSE OHLC date string to time.Time.
func ParseOHLCDate(dateStr string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
