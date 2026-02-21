// Package position provides position scaling logic for graduated exits.
package position

import "github.com/nonobeam/golang-stock-trading/internal/vn"

// PositionScaler calculates exit share quantities for graduated profit-taking.
type PositionScaler struct{}

// NewPositionScaler creates a new position scaler.
func NewPositionScaler() *PositionScaler {
	return &PositionScaler{}
}

// CalculateExitShares calculates the number of shares to exit based on target level and position details.
// Allocation strategy: 30% (T1), 30% (T2), 40% (T3)
func (s *PositionScaler) CalculateExitShares(initialShares int, targetLevel int, symbol string) int {
	var rawShares int
	
	switch targetLevel {
	case 1: // Target 1: 30%
		rawShares = (initialShares * 30) / 100
	case 2: // Target 2: 30% of initial
		rawShares = (initialShares * 30) / 100
	case 3: // Target 3: Remaining (40% of initial, but equal to current shares)
		rawShares = (initialShares * 40) / 100
	default:
		return 0
	}
	
	// Round to exchange board lot
	exchange := vn.GetExchange(symbol)
	return vn.RoundToBoardLot(rawShares, exchange)
}

// CalculateRemainingShares calculates shares remaining after exits.
func (s *PositionScaler) CalculateRemainingShares(initialShares int, exitedShares int) int {
	return initialShares - exitedShares
}

// ValidateExitShares validates that exit shares don't exceed available shares.
func (s *PositionScaler) ValidateExitShares(exitShares int, availableShares int) error {
	if exitShares > availableShares {
		return &ExitValidationError{
			ExitShares:      exitShares,
			AvailableShares: availableShares,
			Message:         "Exit quantity exceeds available shares",
		}
	}
	return nil
}

// ExitValidationError represents a validation error for exit quantities.
type ExitValidationError struct {
	ExitShares      int
	AvailableShares int
	Message         string
}

func (e *ExitValidationError) Error() string {
	return e.Message
}
