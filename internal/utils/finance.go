package utils

// CalculateTurnover calculates the turnover (total value traded) from close price and volume.
// Turnover = Close Price × Volume
//
// This is a common financial metric representing the total monetary value of shares traded.
//
// Parameters:
//   - closePrice: The closing price of the stock
//   - volume: The number of shares traded
//
// Returns:
//   - The calculated turnover as a float64
func CalculateTurnover(closePrice float64, volume float64) float64 {
	if closePrice < 0 || volume < 0 {
		return 0
	}
	return closePrice * volume
}
