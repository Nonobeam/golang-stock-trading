package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
)

var (
	xlsxFilePath string
	symbol       string
	startRow     int
	endRow       int
	dateCol      = "A"
	openCol      = "B"
	highCol      = "C"
	lowCol       = "D"
	closeCol     = "E"
	volumeCol    = "H"
	autoConfirm  bool
)

type DailyBarData struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

func init() {
	flag.StringVar(&xlsxFilePath, "file", "", "Path to the XLSX file")
	flag.StringVar(&symbol, "symbol", "", "Stock symbol")
	flag.IntVar(&startRow, "start", 7, "Starting row (default: 7)")
	flag.IntVar(&endRow, "end", 1006, "Ending row (default: 1006)")
	flag.BoolVar(&autoConfirm, "y", false, "Auto-confirm without prompt")
}

func main() {
	flag.Parse()

	if xlsxFilePath == "" || symbol == "" {
		log.Fatal("Usage: go run main.go -file <path> -symbol <ticker> [-y]")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	f, err := excelize.OpenFile(xlsxFilePath)
	if err != nil {
		log.Fatalf("Failed to open XLSX file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Failed to close XLSX file: %v", err)
		}
	}()

	// Auto-detect sheet name (use first sheet)
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		log.Fatalf("No sheets found in XLSX file")
	}
	sheetName := sheetList[0]
	log.Printf("Using sheet: %s\n", sheetName)

	fmt.Println("===========================================")
	fmt.Println("XLSX Historical Data Import Tool")
	fmt.Println("===========================================")
	fmt.Printf("File: %s\n", xlsxFilePath)
	fmt.Printf("Symbol: %s\n", symbol)
	fmt.Printf("Rows: %d to %d (%d total rows)\n\n", startRow, endRow, endRow-startRow+1)

	if !autoConfirm {
		fmt.Println("TEST MODE: Validating start row and end row")
		fmt.Println("-------------------------------------------")

		testStartRow, err := parseRow(f, sheetName, startRow)
		if err != nil {
			log.Fatalf("Failed to parse start row %d: %v", startRow, err)
		}
		displayRow(fmt.Sprintf("Row %d", startRow), testStartRow)

		testEndRow, err := parseRow(f, sheetName, endRow)
		if err != nil {
			log.Fatalf("Failed to parse end row %d: %v", endRow, err)
		}
		displayRow(fmt.Sprintf("Row %d", endRow), testEndRow)

		fmt.Println("\n===========================================")
		fmt.Print("Data looks correct? Proceed with full import? (y/n): ")
		var confirm string
		fmt.Scanln(&confirm)

		if confirm != "y" && confirm != "Y" {
			fmt.Println("Import cancelled by user")
			return
		}
	}

	fmt.Println("\nStarting full import...")
	fmt.Println("===========================================")

	repo := repository.NewDailyBarsRepository(db)
	ctx := context.Background()

	inserted := 0
	errors := 0

	totalToProcess := endRow - startRow + 1
	for row := startRow; row <= endRow; row++ {
		data, err := parseRow(f, sheetName, row)
		if err != nil {
			log.Printf("Error parsing row %d: %v", row, err)
			errors++
			continue
		}

		err = repo.UpsertDailyBar(
			ctx,
			symbol,
			data.Date,
			data.Open,
			data.High,
			data.Low,
			data.Close,
			data.Volume,
		)

		if err != nil {
			log.Printf("Error upserting row %d: %v", row, err)
			errors++
			continue
		}

		inserted++

		if inserted%100 == 0 || inserted == totalToProcess {
			fmt.Printf("Progress: %d/%d rows processed\n", inserted, totalToProcess)
		}
	}

	fmt.Println("\n===========================================")
	fmt.Println("Import Complete")
	fmt.Println("===========================================")
	fmt.Printf("Total rows processed: %d\n", inserted+errors)
	fmt.Printf("Successfully imported: %d\n", inserted)
	fmt.Printf("Errors: %d\n", errors)
}

func parseRow(f *excelize.File, sheetName string, row int) (*DailyBarData, error) {
	dateStr, err := f.GetCellValue(sheetName, fmt.Sprintf("%s%d", dateCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get date: %w", err)
	}

	date, err := parseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}

	open, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", openCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get open price: %w", err)
	}

	high, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", highCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get high price: %w", err)
	}

	low, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", lowCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get low price: %w", err)
	}

	close, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", closeCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get close price: %w", err)
	}

	volumeStr, err := f.GetCellValue(sheetName, fmt.Sprintf("%s%d", volumeCol, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}

	// Remove thousand separators (commas) before parsing
	volumeStr = strings.ReplaceAll(volumeStr, ",", "")
	// Parse as float first (Excel stores it with decimals like 24233600.00)
	volumeFloat, err := strconv.ParseFloat(volumeStr, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume '%s': %w", volumeStr, err)
	}
	// Convert to int64
	volume := int64(volumeFloat)

	return &DailyBarData{
		Date:   date,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}, nil
}

func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("02/01/2006", dateStr)
}

func getCellFloat(f *excelize.File, sheetName, cell string) (float64, error) {
	valStr, err := f.GetCellValue(sheetName, cell)
	if err != nil {
		return 0, err
	}
	// Remove thousand separators (commas) before parsing
	valStr = strings.ReplaceAll(valStr, ",", "")
	return strconv.ParseFloat(valStr, 64)
}

func displayRow(label string, data *DailyBarData) {
	turnover := data.Close * float64(data.Volume)
	fmt.Printf("\n%s:\n", label)
	fmt.Printf("  Symbol:   %s\n", symbol)
	fmt.Printf("  Date:     %s\n", data.Date.Format("2006-01-02"))
	fmt.Printf("  Open:     %.2f\n", data.Open)
	fmt.Printf("  High:     %.2f\n", data.High)
	fmt.Printf("  Low:      %.2f\n", data.Low)
	fmt.Printf("  Close:    %.2f\n", data.Close)
	fmt.Printf("  Volume:   %d\n", data.Volume)
	fmt.Printf("  Turnover: %.2f (calculated)\n", turnover)
}
