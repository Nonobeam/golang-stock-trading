# Project Tools

This document lists all available tools and scripts in the project and how to use them.

## Core Applications

### Trading Bot (`cmd/app`)

The main application that connects to DNSE API and manages the trading logic and Telegram bot.

- **Run**: `go run cmd/app/main.go`
- **Build**: `go build -o bin/app cmd/app/main.go`
- **Config**: Relies on `.env` file for API credentials and bot tokens.

## Data Tools

### XLSX Importer (`cmd/tools/xlsx_importer`)

Import historical OHLCV data from Simplize XLSX exports into the database.

- **Location**: `cmd/tools/xlsx_importer/main.go`
- **Usage**:
  ```bash
  go run cmd/tools/xlsx_importer/main.go -symbol <SYMBOL> -file <PATH_TO_XLSX>
  ```
- **Optional Arguments**:
  - `-start`: Starting row (default: 7)
  - `-end`: Ending row (default: 1006)
  - `-date-col`: Date column (default: A)
  - `-open-col`, `-high-col`, etc.: Price columns (see defaults in tool)

### History API Check (`cmd/temp_history_check`)

A diagnostic tool to test various DNSE API endpoints to find valid historical data sources.

- **Run**: `go run cmd/temp_history_check/main.go`
- **Purpose**: Debugging why certain historical data requests might fail and testing new API versions.

## Infrastructure & Testing

### ML Ping (`cmd/tools/ml-ping`)

Tests the gRPC connection between the Go application and the Python ML service.

- **Run**: `go run cmd/tools/ml-ping/main.go`
- **Purpose**: Verification that the ML service is up and reachable.

### Deployment Script (`scripts/deploy.sh`)

Automates the deployment process.

- **Usage**: `./scripts/deploy.sh`
- **Steps**: Builds Docker images and restarts services.

## Development Workflows

### Database Migrations

Handled via `db/migrations` files. Use `migrate` tool or manual SQL execution as described in [README_SETUP.md](file:///d:/Program/Source/Nonobeam/stock-trading/golang-stock-trading/README_SETUP.md).
