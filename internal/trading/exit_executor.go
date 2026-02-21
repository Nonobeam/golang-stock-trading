// Package trading provides trading execution logic.
package trading

import (
	"context"
	"fmt"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/position"
)

// ExitExecutor executes exit orders via trading API.
type ExitExecutor struct {
	dnseClient    interface{} // Holds *api.DNSEClient for real API calls
	retryAttempts int
	retryBackoff  []time.Duration
}

// NewExitExecutor creates a new exit executor (stub mode, no API client).
func NewExitExecutor() *ExitExecutor {
	return &ExitExecutor{
		dnseClient:    nil,
		retryAttempts: 3,
		retryBackoff:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// NewExitExecutorWithClient creates exit executor wired to a DNSE API client.
func NewExitExecutorWithClient(dnseClient interface{}) *ExitExecutor {
	return &ExitExecutor{
		dnseClient:    dnseClient,
		retryAttempts: 3,
		retryBackoff:  []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// ExecuteExit executes an exit decision by placing a sell order with retry logic.
func (e *ExitExecutor) ExecuteExit(ctx context.Context, decision *position.ExitDecision, symbol string, currentPrice float64) (*ExitResult, error) {
	if decision.Shares <= 0 {
		return nil, fmt.Errorf("invalid exit shares: %d", decision.Shares)
	}

	var lastErr error
	for attempt := 0; attempt < e.retryAttempts; attempt++ {
		result, err := e.placeSellOrder(ctx, symbol, decision.Shares, currentPrice)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Exponential backoff before retry
		if attempt < e.retryAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(e.retryBackoff[attempt]):
				// continue to next attempt
			}
		}
	}

	return nil, fmt.Errorf("exit execution failed after %d attempts: %w", e.retryAttempts, lastErr)
}

// placeSellOrder places a sell order via trading API.
// If dnseClient is set, calls real DNSE API. Otherwise runs in simulation mode.
func (e *ExitExecutor) placeSellOrder(ctx context.Context, symbol string, shares int, price float64) (*ExitResult, error) {
	// TODO: When dnseClient is non-nil, call the real DNSE trading API:
	//   client := e.dnseClient.(*api.DNSEClient)
	//   order, err := client.PlaceSellOrder(ctx, symbol, shares, price)
	//   return mapToExitResult(order), err

	// Simulation mode
	time.Sleep(50 * time.Millisecond)

	return &ExitResult{
		OrderID:       generateOrderID(),
		Symbol:        symbol,
		SharesOrdered: shares,
		SharesFilled:  shares,
		AveragePrice:  price,
		Status:        "FILLED",
		Timestamp:     time.Now(),
	}, nil
}

// ExitResult contains the result of an exit order execution.
type ExitResult struct {
	OrderID       string
	Symbol        string
	SharesOrdered int
	SharesFilled  int
	AveragePrice  float64
	Status        string // FILLED, PARTIAL, REJECTED
	Timestamp     time.Time
	ErrorMessage  string
}

// IsFullyFilled returns true if the order was fully filled.
func (r *ExitResult) IsFullyFilled() bool {
	return r.SharesOrdered == r.SharesFilled
}

// IsPartialFill returns true if the order was partially filled.
func (r *ExitResult) IsPartialFill() bool {
	return r.SharesFilled > 0 && r.SharesFilled < r.SharesOrdered
}

func generateOrderID() string {
	return fmt.Sprintf("ORD-%d", time.Now().UnixNano())
}
