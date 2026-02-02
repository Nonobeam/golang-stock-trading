# Project Context

## Purpose
A Golang stock trading automation system that integrates with the DNSE (Vietnam stock exchange) API. The project enables automated stock trading operations with real-time market data streaming, authentication flows, technical analysis, risk management, and notifications.

## Module
`github.com/nonobeam/golang-stock-trading`

## Tech Stack
- **Language**: Go 1.21
- **Config Management**: github.com/spf13/viper - Environment variable configuration
- **Logging**: github.com/rs/zerolog - Structured logging with console/JSON output
- **HTTP**: Standard net/http with custom reusable client in `pkg/httpclient/`
- **WebSocket**: github.com/gorilla/websocket - Real-time market data streaming
- **Notifications**: 
  - gopkg.in/gomail.v2 - SMTP email for alerts
  - github.com/go-telegram-bot-api/telegram-bot-api/v5 - Telegram bot for trading token input
- **Environment**: github.com/joho/godotenv - .env file loading

## Project Conventions

### Code Style
- **Package Organization**: Feature-based under `internal/` with 17 packages:
  - **Core**: `api`, `config`, `errors`, `logger`, `notification`, `service`, `websocket`
  - **Analysis**: `analysis`, `analysis/indicators`, `data`, `regime`, `signals`, `scoring`
  - **Trading**: `risk`, `position`, `statistics`, `trading`, `vn`
- **File Naming**: `snake_case.go`
- **JSON Fields**: `lowerCamelCase` for all API payloads (matches DNSE API convention)
- **Error Handling**: Centralized error codes in `internal/errors/errors.go` with categories (AUTH, SYS, NOTIF, API, WS)
- **Documentation**: Google-style docstrings for all exported functions

### Architecture Patterns
- **API Separation**: Split between `info_api.go` (public, no auth) and `trading_api.go` (requires tradingToken)
- **Service Layer**: Business logic in `internal/service/` (auth, telegram)
- **Analysis Layer**: Technical indicators in `internal/analysis/indicators/`, market regime in `internal/regime/`
- **Signal Detection**: Entry patterns (pullback, breakout) in `internal/signals/`, scoring in `internal/scoring/`
- **Risk Management**: Position sizing, stops, and targets in `internal/risk/`
- **Position Tracking**: Active position monitoring in `internal/position/`
- **Statistics**: Performance analytics in `internal/statistics/`
- **Vietnam Market Rules**: Daily limits (±7%) and settlement in `internal/vn/`
- **Data Layer**: OHLCV time series in `internal/data/`
- **Reusable Components**: Generic HTTP client in `pkg/httpclient/`
- **Configuration**: Single config struct loaded via Viper at startup

### Testing Strategy
- Standard Go testing with `go test ./...`
- **Test Location**: Tests placed in `test/` directory (not alongside source files)
- **Naming**: `*_test.go` with descriptive test file names (e.g., `position_sizing_test.go`, `scoring_test.go`)
- **Table-driven tests**: Preferred for testing multiple scenarios

### Git Workflow
- Feature branches for new functionality
- Commits should be descriptive (conventional commits preferred)

### OpenSpec Change Proposals
- **Location**: `openspec/changes/{change-name}/proposal.md`
- **Workflow**: Proposal → Review → Implementation → Archive
- **Documentation**: See `openspec/AGENTS.md` for full workflow details

## Domain Context

### DNSE API
- DNSE is a Vietnam stock exchange broker API
- Authentication: Login → Trading Token (via Telegram)
- APIs are split: Info APIs (no token) vs Trading APIs (tradingToken required)

### Key Authentication Flow
1. `POST /auth-service/login` → Returns `accessToken`
2. User provides trading token via Telegram bot
3. Trading token is set for all trading API requests

### WebSocket Topics
Real-time market data via WebSocket with topic patterns:
- Stock Info: `quotes/krx/mdds/stockinfo/v1/roundlot/symbol/{symbol}`
- Top Price: `quotes/krx/mdds/topprice/v1/roundlot/symbol/{symbol}`
- OHLC: `quotes/krx/mdds/v2/ohlc/{type}/{resolution}/{symbol}`
- Tick Data: `quotes/krx/mdds/tick/v1/roundlot/symbol/{symbol}`
- Market Index: `quotes/krx/mdds/index/{indexName}`

## Important Constraints
- Trading token must be provided by user via Telegram
- Trading token timeout: 5 minutes (configurable)
- Trading APIs require valid tradingToken
- WebSocket connection requires authentication headers

## External Dependencies
- **DNSE API**: Primary broker API for authentication, trading, and market data
- **SMTP Server**: For email notifications/alerts
- **Telegram Bot API**: For trading token input from users
- **DNSE WebSocket**: Real-time market data streams
