// Package main provides a CLI tool for manually running settlement status updates
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/jobs"
)

func main() {
	// Parse command line flags
	force := flag.Bool("force", false, "Force update even if recently run")
	dryRun := flag.Bool("dry-run", false, "Show what would be updated without making changes")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	// Load configuration (for other settings if needed in future)
	_ = config.Get()

	// Connect to database using environment variables
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "trading"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost,
		dbPort,
		dbUser,
		dbPassword,
		dbName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("=== Settlement Status Update Tool ===")
	fmt.Printf("Run time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Force: %v, Dry-run: %v, Verbose: %v\n\n", *force, *dryRun, *verbose)

	ctx := context.Background()
	updater := jobs.NewSettlementUpdater(db)

	// Check if update should run
	if !*force {
		shouldRun, err := updater.ShouldRunUpdate(ctx)
		if err != nil {
			log.Fatalf("Failed to check update status: %v", err)
		}

		if !shouldRun {
			lastUpdate, _ := updater.GetLastUpdateTime(ctx)
			fmt.Printf("Settlement update was recently run at %s\n", lastUpdate.Format("2006-01-02 15:04:05"))
			fmt.Println("Use --force to run anyway")
			os.Exit(0)
		}
	}

	if *dryRun {
		fmt.Println("DRY RUN MODE - No changes will be made\n")
		// In a real implementation, we'd query and show what would change
		fmt.Println("Would update settlement statuses for all open positions")
		fmt.Println("Use without --dry-run to apply changes")
		os.Exit(0)
	}

	// Run the update
	result, err := updater.RunDailySettlementUpdate(ctx)
	if err != nil {
		log.Fatalf("Settlement update failed: %v", err)
	}

	// Display results
	fmt.Println("=== Update Results ===")
	fmt.Printf("Total positions processed: %d\n", result.TotalPositions)
	fmt.Printf("Positions updated: %d\n", result.UpdatedPositions)
	fmt.Printf("Transitioned to LIQUID: %d\n", result.TransitionedToLiquid)
	fmt.Printf("Still locked: %d\n", result.StillLocked)

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  Errors encountered: %d\n", len(result.Errors))
		if *verbose {
			for i, errMsg := range result.Errors {
				fmt.Printf("  %d. %s\n", i+1, errMsg)
			}
		} else {
			fmt.Println("Use --verbose to see error details")
		}
	}

	fmt.Printf("\nCompleted at: %s\n", result.UpdatedAt.Format("2006-01-02 15:04:05"))

	if result.TransitionedToLiquid > 0 {
		fmt.Printf("\n✓ %d position(s) became liquid and can now have stop losses executed\n", result.TransitionedToLiquid)
	}
}
