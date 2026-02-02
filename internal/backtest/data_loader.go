package backtest

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/data"
)

// HistoricalDataLoader loads and validates historical OHLCV data from CSV files
type HistoricalDataLoader struct {
	dataPath  string
	cache     *DataCache
	validator *DataValidator
}

// NewHistoricalDataLoader creates a new data loader
func NewHistoricalDataLoader(dataPath string) *HistoricalDataLoader {
	return &HistoricalDataLoader{
		dataPath:  dataPath,
		cache:     NewDataCache(),
		validator: NewDataValidator(),
	}
}

// LoadOHLCV loads historical OHLCV data for a symbol
// CSV format: timestamp,symbol,open,high,low,close,volume
// Example: 2023-01-03T09:00:00+07:00,VCB,80000,81500,79500,81000,2500000
func (loader *HistoricalDataLoader) LoadOHLCV(symbol string) ([]data.OHLCV, error) {
	// Check cache first
	if cached, exists := loader.cache.Get(symbol); exists {
		return cached, nil
	}

	// Construct file path
	filePath := filepath.Join(loader.dataPath, fmt.Sprintf("%s.csv", symbol))

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file %s: %w", filePath, err)
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate headers
	expectedHeaders := []string{"timestamp", "symbol", "open", "high", "low", "close", "volume"}
	if len(headers) < len(expectedHeaders) {
		return nil, fmt.Errorf("invalid CSV format: expected at least %d columns, got %d", len(expectedHeaders), len(headers))
	}

	// Parse rows
	var ohlcvData []data.OHLCV
	lineNum := 1 // header is line 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV line %d: %w", lineNum, err)
		}
		lineNum++

		if len(record) < len(expectedHeaders) {
			return nil, fmt.Errorf("invalid CSV format at line %d: expected at least %d columns, got %d", lineNum, len(expectedHeaders), len(record))
		}

		// Parse timestamp (ISO 8601 with timezone)
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp at line %d: %s (error: %w)", lineNum, record[0], err)
		}

		// Parse symbol (just verify it matches)
		rowSymbol := record[1]
		if rowSymbol != symbol {
			return nil, fmt.Errorf("symbol mismatch at line %d: expected %s, got %s", lineNum, symbol, rowSymbol)
		}

		// Parse OHLCV values
		open, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid open price at line %d: %s (error: %w)", lineNum, record[2], err)
		}

		high, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid high price at line %d: %s (error: %w)", lineNum, record[3], err)
		}

		low, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid low price at line %d: %s (error: %w)", lineNum, record[4], err)
		}

		close, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid close price at line %d: %s (error: %w)", lineNum, record[5], err)
		}

		volume, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid volume at line %d: %s (error: %w)", lineNum, record[6], err)
		}

		// Create OHLCV bar
		bar := data.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}

		ohlcvData = append(ohlcvData, bar)
	}

	if len(ohlcvData) == 0 {
		return nil, fmt.Errorf("no data found in file %s", filePath)
	}

	// Sort by timestamp (oldest first) to ensure chronological order
	sort.Slice(ohlcvData, func(i, j int) bool {
		return ohlcvData[i].Timestamp.Before(ohlcvData[j].Timestamp)
	})

	// Validate data
	if err := loader.validator.ValidateData(ohlcvData); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	// Cache for future use
	loader.cache.Set(symbol, ohlcvData)

	return ohlcvData, nil
}

// ClearCache clears the data cache (useful for testing)
func (loader *HistoricalDataLoader) ClearCache() {
	loader.cache.Clear()
}

// DataValidator validates historical data
type DataValidator struct{}

// NewDataValidator creates a new data validator
func NewDataValidator() *DataValidator {
	return &DataValidator{}
}

// ValidateData validates OHLCV data for consistency
func (v *DataValidator) ValidateData(ohlcv []data.OHLCV) error {
	if len(ohlcv) == 0 {
		return fmt.Errorf("empty data set")
	}

	for i, bar := range ohlcv {
		// Check all prices are positive
		if bar.Open <= 0 || bar.High <= 0 || bar.Low <= 0 || bar.Close <= 0 {
			return fmt.Errorf("invalid prices at index %d (date: %s): prices must be positive", i, bar.Timestamp.Format("2006-01-02"))
		}

		// Check volume is non-negative
		if bar.Volume < 0 {
			return fmt.Errorf("invalid volume at index %d (date: %s): volume cannot be negative", i, bar.Timestamp.Format("2006-01-02"))
		}

		// Check OHLCV relationships
		maxOC := bar.Open
		if bar.Close > maxOC {
			maxOC = bar.Close
		}
		minOC := bar.Open
		if bar.Close < minOC {
			minOC = bar.Close
		}

		if bar.High < maxOC {
			return fmt.Errorf("invalid OHLC at index %d (date: %s): High (%.2f) must be >= max(Open, Close) (%.2f)",
				i, bar.Timestamp.Format("2006-01-02"), bar.High, maxOC)
		}

		if bar.Low > minOC {
			return fmt.Errorf("invalid OHLC at index %d (date: %s): Low (%.2f) must be <= min(Open, Close) (%.2f)",
				i, bar.Timestamp.Format("2006-01-02"), bar.Low, minOC)
		}

		// Check chronological ordering
		if i > 0 {
			prevBar := ohlcv[i-1]
			if !bar.Timestamp.After(prevBar.Timestamp) {
				return fmt.Errorf("data not in chronological order at index %d: %s is not after %s",
					i, bar.Timestamp.Format("2006-01-02"), prevBar.Timestamp.Format("2006-01-02"))
			}
		}
	}

	return nil
}

// DataCache stores loaded historical data
type DataCache struct {
	cache map[string][]data.OHLCV
}

// NewDataCache creates a new data cache
func NewDataCache() *DataCache {
	return &DataCache{
		cache: make(map[string][]data.OHLCV),
	}
}

// Get retrieves cached data for symbol
func (dc *DataCache) Get(symbol string) ([]data.OHLCV, bool) {
	data, exists := dc.cache[symbol]
	return data, exists
}

// Set stores data for symbol
func (dc *DataCache) Set(symbol string, data []data.OHLCV) {
	dc.cache[symbol] = data
}

// Clear clears all cached data
func (dc *DataCache) Clear() {
	dc.cache = make(map[string][]data.OHLCV)
}
