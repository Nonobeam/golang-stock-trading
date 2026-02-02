package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/backtest"
)

func main() {
	fmt.Println("=== Backtesting System Demo ===")

	// Configure backtest
	config := &backtest.BacktestConfig{
		Symbol:         "VCB",
		DataPath:       "d:\\Program\\Source\\Nonobeam\\stock-trading\\golang-stock-trading\\test\\fixtures",
		StartDate:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:        time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
		InitialCapital: 100_000_000, // 100M VND
		MinSignalScore: 7,
		MaxPositions:   3,
		RiskPerTrade:   0.01,   // 1%
		Commission:     0.0015, // 0.15%
		Slippage:       0.001,  // 0.1%
		DryRun:         true,   // Start with dry run
		Verbose:        true,
	}

	// Create engine
	engine, err := backtest.NewBacktestEngine(config)
	if err != nil {
		log.Fatalf("Failed to create backtest engine: %v", err)
	}

	// Run backtest
	result, err := engine.Run()
	if err != nil {
		log.Fatalf("Backtest failed: %v", err)
	}

	// Print results
	fmt.Println("\n=== Backtest Results ===")
	fmt.Printf("Trading Days: %d\n", result.TradingDays)
	fmt.Printf("Initial Capital: %.2f VND\n", result.InitialCapital)
	fmt.Printf("Final Capital: %.2f VND\n", result.FinalCapital)

	if result.Metrics != nil {
		fmt.Printf("\nTrades: %d\n", result.Metrics.TotalTrades)
		fmt.Printf("Win Rate: %.2f%%\n", result.Metrics.WinRate)
		fmt.Printf("Profit Factor: %.2f\n", result.Metrics.ProfitFactor)
		fmt.Printf("Avg R-Multiple: %.2fR\n", result.Metrics.AvgRMultiple)
	}

	fmt.Println("\n✅ Backtesting system is working!")
}
