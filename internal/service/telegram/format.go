package telegram

import (
	"fmt"
	"strings"
)

// formatVND formats Vietnamese Dong currency
func formatVND(amount float64) string {
	// Format with thousand separators
	str := fmt.Sprintf("%.0f", amount)
	n := len(str)
	if n <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	return result.String()
}
