package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"

	"github.com/nonobeam/golang-stock-trading/internal/db/repository"
)

type DailyBarData struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

var (
	symbolFlag    string
	filePathFlag  string
	startRowFlag  int
	endRowFlag    int
	dateColFlag   string
	openColFlag   string
	highColFlag   string
	lowColFlag    string
	closeColFlag  string
	volumeColFlag string
)

func init() {
	flag.StringVar(&symbolFlag, "symbol", "HPG", "Stock symbol to import")
	flag.StringVar(&filePathFlag, "file", "", "Path to the XLSX file")
	flag.IntVar(&startRowFlag, "start", 7, "Starting row number (1-indexed)")
	flag.IntVar(&endRowFlag, "end", 1006, "Ending row number (1-indexed)")

	flag.StringVar(&dateColFlag, "date-col", "A", "Column for Date")
	flag.StringVar(&openColFlag, "open-col", "B", "Column for Open price")
	flag.StringVar(&highColFlag, "high-col", "C", "Column for High price")
	flag.StringVar(&lowColFlag, "low-col", "D", "Column for Low price")
	flag.StringVar(&closeColFlag, "close-col", "E", "Column for Close price")
	flag.StringVar(&volumeColFlag, "volume-col", "H", "Column for Volume")
}

func main() {
	flag.Parse()

	if filePathFlag == "" {
		log.Fatal("Error: -file argument is required")
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

	f, err := excelize.OpenFile(filePathFlag)
	if err != nil {
		log.Fatalf("Failed to open XLSX file: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Failed to close XLSX file: %v", err)
		}
	}()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		log.Fatalf("No sheets found in XLSX file")
	}
	sheetName := sheetList[0]
	log.Printf("Using sheet: %s\n", sheetName)

	fmt.Println("===========================================")
	fmt.Printf("XLSX Historical Data Import Tool - %s\n", symbolFlag)
	fmt.Println("===========================================")
	fmt.Printf("File: %s\n", filePathFlag)
	fmt.Printf("Symbol: %s\n", symbolFlag)
	fmt.Printf("Rows: %d to %d (%d total rows)\n\n", startRowFlag, endRowFlag, endRowFlag-startRowFlag+1)

	fmt.Printf("TEST MODE: Validating row %d and row %d\n", startRowFlag, endRowFlag)
	fmt.Println("-------------------------------------------")

	testRowStart, err := parseRow(f, sheetName, startRowFlag)
	if err != nil {
		log.Fatalf("Failed to parse row %d: %v", startRowFlag, err)
	}
	displayRow(fmt.Sprintf("Row %d (Start)", startRowFlag), testRowStart)

	testRowEnd, err := parseRow(f, sheetName, endRowFlag)
	if err != nil {
		log.Fatalf("Failed to parse row %d: %v", endRowFlag, err)
	}
	displayRow(fmt.Sprintf("Row %d (End)", endRowFlag), testRowEnd)

	fmt.Println("\n===========================================")
	// fmt.Print("Data looks correct? Proceed with full import? (y/n): ") // Disable interactive prompt for bot usage
	// var confirm string
	// fmt.Scanln(&confirm)

	// if confirm != "y" && confirm != "Y" {
	// 	fmt.Println("Import cancelled by user")
	// 	return
	// }

	fmt.Println("Starting full import...")
	fmt.Println("===========================================")

	repo := repository.NewDailyBarsRepository(db)
	ctx := context.Background()

	inserted := 0
	errors := 0

	for row := startRowFlag; row <= endRowFlag; row++ {
		data, err := parseRow(f, sheetName, row)
		if err != nil {
			log.Printf("Error parsing row %d: %v", row, err)
			errors++
			continue
		}

		err = repo.UpsertDailyBar(
			ctx,
			symbolFlag,
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

		if inserted%100 == 0 {
			fmt.Printf("Progress: %d/%d rows processed\n", inserted, endRowFlag-startRowFlag+1)
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
	dateStr, err := f.GetCellValue(sheetName, fmt.Sprintf("%s%d", dateColFlag, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get date: %w", err)
	}

	date, err := parseDate(dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date '%s': %w", dateStr, err)
	}

	open, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", openColFlag, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get open: %w", err)
	}

	high, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", highColFlag, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get high: %w", err)
	}

	low, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", lowColFlag, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get low: %w", err)
	}

	close, err := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", closeColFlag, row))
	if err != nil {
		return nil, fmt.Errorf("failed to get close: %w", err)
	}

	volume, err := getCellInt(f, sheetName, fmt.Sprintf("%s%d", volumeColFlag, row))
	if err != nil {
		// Try to handle volume as float if int fails, sometimes formatted as 1000.00
		vFloat, errFloat := getCellFloat(f, sheetName, fmt.Sprintf("%s%d", volumeColFlag, row))
		if errFloat != nil {
			return nil, fmt.Errorf("failed to get volume: %w", err)
		}
		volume = int64(vFloat)
	}

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
	// Try Excel serial date format first
	// excelDate, err := strconv.ParseFloat(dateStr, 64)
	// if err == nil {
	// 	return excelDateToTime(excelDate), nil
	// }
	
	layouts := []string{"02/01/2006", "2006-01-02", "01/02/2006"}
	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown date format")
}

func getCellFloat(f *excelize.File, sheet, axis string) (float64, error) {
	val, err := f.GetCellValue(sheet, axis)
	if err != nil {
		return 0, err
	}
	// Handle empty strings or "-" as 0
	if val == "" || val == "-" {
		return 0, nil
	}
	
	// Remove commas if present
	val = strings.ReplaceAll(val, ",", "")
	
	var result float64
	_, err = fmt.Sscanf(val, "%f", &result)
	return result, err
}

func getCellInt(f *excelize.File, sheet, axis string) (int64, error) {
	val, err := f.GetCellValue(sheet, axis)
	if err != nil {
		return 0, err
	}
	if val == "" || val == "-" {
		return 0, nil
	}
	
	val = strings.ReplaceAll(val, ",", "")
	
	var result int64
	_, err = fmt.Sscanf(val, "%d", &result)
	return result, err
}

func displayRow(label string, data *DailyBarData) {
	fmt.Printf("%s: %s | O: %.2f | H: %.2f | L: %.2f | C: %.2f | V: %d\n",
		label, data.Date.Format("2006-01-02"),
		data.Open, data.High, data.Low, data.Close, data.Volume)
}
