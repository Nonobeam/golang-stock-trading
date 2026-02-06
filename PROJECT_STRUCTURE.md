# Project Structure Documentation

This document provides a comprehensive overview of the stock trading system architecture for AI assistants to quickly understand the codebase organization and key components.

## Directory Structure

```
golang-stock-trading/
├── cmd/                        # Application entry points
│   ├── app/                   # Main API server
│   │   ├── main.go           # Server entry point
│   │   └── adapters.go       # Service adapters/wiring
│   └── tools/                # Utility tools
│       ├── ml-ping/          # ML service health checker
│       └── xlsx_importer/    # Excel data import tools
│
├── internal/                  # Private application code (Go)
│   ├── analysis/             # Technical analysis
│   │   ├── indicators/       # Technical indicators (SMA, RSI, etc.)
│   │   ├── manager.go        # Analysis orchestration
│   │   └── weekly_aggregator.go
│   │
│   ├── api/                  # External API clients
│   │   ├── dnse/             # DNSE broker integration
│   │   ├── dnse_auth.go      # Authentication
│   │   ├── dnse_client.go    # Main client
│   │   ├── info_api.go       # Market data API
│   │   ├── trading_api.go    # Trading API
│   │   └── ohlc.go          # Price data fetching
│   │
│   ├── backtest/             # Backtesting engine
│   │   ├── engine.go         # Core backtest logic
│   │   ├── simulator.go      # Market simulation
│   │   ├── metrics.go        # Performance calculations
│   │   └── position_tracker.go
│   │
│   ├── config/               # Configuration management
│   │   └── config.go         # App configuration
│   │
│   ├── data/                 # Data structures
│   │   ├── ohlcv.go         # Price bar structures
│   │   ├── series.go        # Time series data
│   │   └── aggregator.go    # Data aggregation
│   │
│   ├── db/                   # Database layer
│   │   └── repository/       # Data access layer
│   │       ├── types.go             # Common DB types
│   │       ├── position_repository.go    # Position CRUD
│   │       ├── daily_bars_repository.go  # Price data
│   │       ├── signal_history_repository.go
│   │       ├── stock_signal_pref_repository.go
│   │       ├── watchlist_repository.go
│   │       └── user_config_repository.go
│   │
│   ├── errors/               # Error handling
│   │   └── errors.go
│   │
│   ├── jobs/                 # Background jobs
│   │   └── settlement_update.go  # Daily settlement status updater
│   │
│   ├── logger/               # Logging utilities
│   │   └── logger.go
│   │
│   ├── notification/         # Notification systems
│   │   └── email.go         # Email notifications
│   │
│   ├── position/             # Position management (Go side)
│   │   ├── types.go         # Position types
│   │   ├── tracker.go       # Position tracking
│   │   ├── metrics.go       # P&L, returns calculations
│   │   ├── alerts.go        # Position alerts
│   │   ├── dashboard.go     # Position dashboard
│   │   ├── managed_tracker.go # Advanced tracking
│   │   ├── stop_rules.go    # Stop loss rules
│   │   ├── stop_adjustment.go
│   │   ├── stop_engine.go   # Stop loss engine
│   │   └── integration.go   # Integration helpers
│   │
│   ├── regime/               # Market regime detection
│   │   ├── types.go
│   │   ├── detector.go      # Regime identification
│   │   ├── vn_market.go     # Vietnam-specific logic
│   │   ├── transitions.go   # Regime transitions
│   │   ├── scoring.go
│   │   └── classifier.go
│   │
│   ├── risk/                 # Risk management
│   │   ├── params.go        # Risk parameters
│   │   ├── stop_loss.go     # Stop loss calculations
│   │   ├── locked_risk.go   # T+2 settlement locked risk
│   │   ├── targets.go       # Profit targets
│   │   ├── targets_types.go
│   │   ├── targets_r.go     # R-multiple targets
│   │   ├── targets_atr.go   # ATR-based targets
│   │   ├── targets_tech.go  # Technical targets
│   │   ├── targets_fib.go   # Fibonacci targets
│   │   ├── targets_measured.go
│   │   ├── targets_trailing.go
│   │   └── targets_comprehensive.go
│   │
│   ├── scoring/              # Signal scoring system
│   │   ├── types.go
│   │   ├── scorer.go        # Main scoring logic
│   │   ├── components.go    # Scoring components
│   │   └── volume.go        # Volume analysis
│   │
│   ├── service/              # Business logic services
│   │   ├── account/         # Account management
│   │   │   └── account_service.go
│   │   ├── auth/            # Authentication
│   │   │   └── auth_service.go
│   │   ├── jwt/             # JWT token handling
│   │   │   └── jwt_service.go
│   │   ├── market/          # Market data service
│   │   │   └── market_service.go
│   │   ├── monitor/         # Position monitoring
│   │   │   └── service.go
│   │   ├── otp/             # OTP service
│   │   │   └── otp_service.go
│   │   ├── position/        # Position service (orchestration)
│   │   │   ├── position_service.go
│   │   │   ├── monitor_service.go
│   │   │   └── stop_loss_validator.go  # T+2 stop loss validation
│   │   ├── recommendation/  # Trade recommendations
│   │   │   └── recommendation_service.go
│   │   ├── scanner/         # Real-time scanner
│   │   │   ├── live_scanner.go
│   │   │   ├── bar_cache.go
│   │   │   ├── data_provider_adapter.go
│   │   │   └── helpers.go
│   │   ├── signal/          # Signal management
│   │   │   └── signal_service.go
│   │   ├── telegram/        # Telegram bot integration
│   │   │   ├── bot_service.go          # Main bot
│   │   │   ├── bot_service_positions.go # /position command
│   │   │   ├── bot_service_status.go   # /status command
│   │   │   ├── bot_service_alerts.go   # Alert handling
│   │   │   ├── settlement_alerts.go    # T+2 settlement alerts
│   │   │   ├── settlement_reports.go   # T+2 settlement reports
│   │   │   ├── alert_service.go
│   │   │   └── interfaces.go
│   │   └── watchlist/       # Watchlist management
│   │       └── watchlist_service.go
│   │
│   ├── signals/              # Signal detection
│   │   ├── patterns.go      # Chart patterns
│   │   ├── support.go       # Support/resistance
│   │   ├── volume_analysis.go
│   │   ├── pullback.go      # Pullback detection
│   │   └── settlement_validator.go  # T+2 settlement signal validation
│   │
│   ├── statistics/           # Trading statistics
│   │   ├── calculator.go    # Main stats calculator
│   │   ├── winrate.go       # Win rate metrics
│   │   ├── expectancy.go    # Expected value
│   │   ├── drawdown.go      # Drawdown calculations
│   │   ├── distribution.go  # Return distribution
│   │   ├── equity.go        # Equity curve
│   │   ├── mae_mfe.go       # MAE/MFE analysis
│   │   ├── time_metrics.go  # Time-based metrics
│   │   ├── r_distribution.go # R-multiple distribution
│   │   ├── risk_adjusted.go # Sharpe, Sortino, etc.
│   │   └── helpers.go
│   │
│   ├── trading/              # Trading execution
│   │   └── trade.go         # Trade execution logic
│   │
│   ├── vn/                   # Vietnam market specifics
│   │   ├── limits.go        # Price limits, trading rules
│   │   └── settlement.go    # T+2 settlement calculations
│   │
│   └── websocket/            # WebSocket handling
│       ├── handlers.go
│       └── topics.go
│
├── ml-service/               # Python ML pipeline
│   ├── daily/               # Daily batch processes
│   │   ├── feature_updater.py       # Update feature database
│   │   ├── outcome_recorder.py      # Record trade outcomes
│   │   ├── prediction_generator.py  # Generate predictions
│   │   ├── retrainer.py            # Model retraining
│   │   ├── daily_signals.py        # Daily signal generation
│   │   ├── run_daily_features.py   # Feature runner
│   │   ├── run_daily_outcomes.py   # Outcome runner
│   │   ├── run_daily_predictions.py # Prediction runner
│   │   ├── run_daily_retrain.py    # Retrain runner
│   │   └── run_daily_equity.py     # Equity tracking runner
│   │
│   ├── monitoring/          # System monitoring
│   │   ├── drawdown_alerts.py      # Drawdown alert system
│   │   └── settlement_monitor.py   # T+2 settlement monitoring
│   │
│   ├── position_manager/    # Position tracking (Python side)
│   │   └── manager.py       # Position state management
│   │
│   ├── position_sizing/     # Position sizing logic
│   │   ├── kelly.py         # Kelly criterion implementation
│   │   ├── drawdown_manager.py     # Drawdown-based sizing
│   │   └── locked_risk.py          # T+2 locked risk calculator
│   │
│   ├── scripts/             # Analysis scripts
│   │   ├── generate_r_report.py    # R-multiple reporting
│   │   └── plot_equity_curve.py    # Equity visualization
│   │
│   ├── signals/             # Signal generation (Python)
│   │   ├── generator.py           # Main signal generator
│   │   └── enhanced_generator.py  # Enhanced signals (with T+2 validation)
│   │
│   ├── tests/               # Python test suite
│   │   ├── test_drawdown_integration.py
│   │   ├── test_equity_tracking.py
│   │   ├── test_integration_scenarios.py
│   │   ├── test_position_manager_avg_cost.py
│   │   ├── test_signal_capacity.py
│   │   └── test_settlement_tracking.py  # T+2 settlement tests
│   │
│   ├── validation/          # Validation utilities
│   │   ├── portfolio_metrics.py    # Portfolio metrics
│   │   └── r_multiple_analytics.py # R-multiple analytics
│   │
│   ├── main.py              # ML service entry point
│   ├── config.py            # ML configuration
│   ├── train.py             # Model training
│   └── backfill_features.py # Historical feature backfill
│
├── db/migrations/           # Database migrations
│   ├── 000001_initial_schema.up.sql
│   ├── 000004_create_market_data_tables.up.sql
│   ├── 000007_create_ml_tables.up.sql
│   ├── 000008_add_multi_horizon_tables.up.sql
│   ├── 000009_remove_unused_tables.up.sql
│   ├── 000010_add_position_indexes.up.sql
│   ├── 000011_validation_infrastructure.up.sql
│   ├── 000012_add_portfolio_equity_tracking.up.sql
│   ├── 000013_add_position_entries_tracking.up.sql
│   └── 000014_add_settlement_tracking.up.sql  # T+2 settlement tracking
│
├── openspec/                # Architecture specifications
│   ├── project.md          # Project conventions
│   ├── AGENTS.md           # OpenSpec workflow instructions
│   ├── specs/              # Current specifications
│   └── changes/            # Pending change proposals
│       ├── add-drawdown-risk-controls/
│       ├── add-purchase-tracking/
│       ├── enforce-stop-loss-preservation/
│       ├── implement-t2-settlement-risk/
│       ├── integrate-enhanced-daily-signals/
│       └── track-average-cost-capacity/
│
├── pkg/                     # Public libraries
│   └── httpclient/         # HTTP client utilities
│       └── client.go
│
├── test/                    # Go test files
│   ├── analysis_test.go
│   ├── position_tracker_test.go
│   ├── statistics_*.go
│   └── targets_*.go
│
├── examples/                # Example code
│   └── run_backtest.go     # Backtest example
│
├── bin/                     # Compiled binaries
│   └── api-server.exe
│
├── scripts/                 # Utility scripts
│   └── migrate_settlement_data.py  # T+2 settlement backfill
│
├── docs/                    # Documentation
│   └── SETTLEMENT_TRACKING.md      # T+2 settlement guide
│
├── .agent/                  # AI agent workflows
│   └── workflows/
│       ├── openspec-proposal.md
│       └── openspec-apply.md
│
├── go.mod                   # Go dependencies
├── CLAUDE.md               # Claude-specific instructions
├── AGENTS.md               # Agent documentation
├── NEXT_IMPLEMENT_PLAN.md  # Next implementation plan
└── PROJECT_STRUCTURE.md    # This file
```

## Key Components

### Entry Points
- **API Server**: `cmd/app/main.go` - Main HTTP server with REST API
- **ML Service**: `ml-service/main.py` - Python ML prediction service
- **Tools**: `cmd/tools/*` - Utility scripts

### Core Technologies
- **Language**: Go 1.24.0 (backend), Python 3.x (ML)
- **Database**: PostgreSQL with golang-migrate
- **Cache**: Redis (mandatory since commit 64c5de6)
- **Broker API**: DNSE (Vietnam stock exchange)
- **Bot**: Telegram Bot API
- **ML Framework**: (Check ml-service/requirements.txt)

### Database Schema
Key tables (see migrations for full schema):
- `positions` - Open/closed trading positions (with settlement tracking)
- `position_entries` - Individual purchase records (T+2 tracking)
- `position_settlement_tracking` - Daily settlement status snapshots
- `theoretical_stop_breaches` - Stop breaches during settlement lock
- `portfolio_equity` - Daily equity snapshots
- `daily_bars` - OHLCV price data
- `signals` - Trading signals
- `signal_history` - Historical signals
- `stock_signal_preferences` - Per-stock signal settings
- `watchlists` - User watchlists
- ML tables (features, predictions, outcomes)

## Data Flow

### 1. Signal Generation Flow
```
ML Service (ml-service/signals/generator.py)
    ↓ Generates signals
Database (signals table)
    ↓ Fetched by
Go Service (internal/service/signal/signal_service.go)
    ↓ Evaluated by
Position Service (internal/service/position/position_service.go)
    ↓ Executed via
Trading API (internal/api/trading_api.go)
    ↓ Broker (DNSE)
```

### 2. Position Tracking Flow
```
Trade Execution
    ↓ Records entry
Position Repository (internal/db/repository/position_repository.go)
    ↓ Synced to
Position Manager (ml-service/position_manager/manager.py)
    ↓ Updates
Portfolio Equity (ml-service/daily/run_daily_equity.py)
    ↓ Monitored by
Monitor Service (internal/service/position/monitor_service.go)
    ↓ Alerts via
Telegram Bot (internal/service/telegram/bot_service.go)
```

### 3. Daily ML Pipeline
```
1. Feature Update: ml-service/daily/run_daily_features.py
   ↓ Fetches OHLCV from daily_bars
2. Prediction: ml-service/daily/run_daily_predictions.py
   ↓ Generates predictions using trained model
3. Signal Generation: ml-service/daily/daily_signals.py
   ↓ Creates actionable signals
4. Outcome Recording: ml-service/daily/run_daily_outcomes.py
   ↓ Records actual trade results
5. Retraining: ml-service/daily/run_daily_retrain.py
   ↓ Periodic model updates
6. Equity Tracking: ml-service/daily/run_daily_equity.py
   ↓ Daily portfolio snapshots
```

## Critical Constraints & Domain Rules

### Market Rules (Vietnam Stock Exchange)
- **T+2 Settlement**: Cash/shares settle 2 business days after trade, sellable on T+3
  - Status Lifecycle: LOCKED_T0 → LOCKED_T1 → LOCKED_T2 → LIQUID
  - Implementation: `internal/vn/settlement.go`, `internal/risk/locked_risk.go`
  - Database: Migration 000014, `position_settlement_tracking` table
  - Daily Job: `internal/jobs/settlement_update.go`
  - User Guide: `docs/SETTLEMENT_TRACKING.md`
- **Locked Risk Budget**: Max 10% of account in locked capital risk (configurable 5-20%)
  - Exchange multipliers: HOSE 20%, HNX 30%, UPCOM 40%
  - Signal validation: `internal/signals/settlement_validator.go`
  - Python integration: `ml-service/position_sizing/locked_risk.py`
- **Entry Day Restrictions**: Thursday/Friday entries reduced to 50% size
  - Rationale: Extends settlement over weekend, increases locked duration
- **Price Limits**: Daily price movement limits (±7% typical)
  - Implementation: `internal/vn/limits.go`
- **Trading Hours**: 9:00-11:30, 13:00-14:30 Vietnam time
- **Lot Size**: Minimum 100 shares per order

### Risk Management
- **Kelly Criterion**: Position sizing via Kelly formula
  - Implementation: `ml-service/position_sizing/kelly.py`
- **Drawdown Control**: Reduce exposure during drawdowns
  - Implementation: `ml-service/position_sizing/drawdown_manager.py`
  - Alerts: `ml-service/monitoring/drawdown_alerts.py`
- **Stop Loss Preservation**: Stop losses cannot be moved further from entry
  - Spec: `openspec/changes/enforce-stop-loss-preservation/`
  - Implementation: `internal/position/stop_engine.go`
- **R-Multiple Tracking**: Risk-reward ratio tracking
  - Analytics: `ml-service/validation/r_multiple_analytics.py`
  - Go side: `internal/risk/targets_r.go`

### Position Management
- **Average Cost Tracking**: Track average entry price per symbol
  - Spec: `openspec/changes/track-average-cost-capacity/`
  - Go: `internal/service/position/position_service.go`
  - Python: `ml-service/position_manager/manager.py`
- **Capacity Limits**: Max positions per symbol/portfolio
  - Tests: `ml-service/tests/test_signal_capacity.py`

## Telegram Bot Commands
Recent implementations (see git history):
- `/position` - Display detailed position info (includes settlement status)
- `/status` - System status
- `/watch` - Add symbol to watchlist
- `/unwatch` - Remove from watchlist
- Automatic alerts:
  - Token expiration
  - Position becomes liquid (T+2 → T+3 transition)
  - Locked risk threshold warnings (>80%)
  - Theoretical stop breaches (stop hit but non-executable)
  - Entry day warnings (Thursday/Friday half-size)

## Service Architecture

### Go Services (internal/service/)
- **account**: User account management
- **auth**: Authentication and authorization
- **jwt**: JWT token generation/validation
- **market**: Market data aggregation
- **monitor**: Position monitoring (legacy)
- **otp**: One-time password for broker
- **position**: Position lifecycle management
- **recommendation**: Trade recommendation engine
- **scanner**: Real-time market scanning
- **signal**: Signal evaluation and filtering
- **telegram**: Bot commands and notifications
- **watchlist**: Watchlist CRUD operations

### Python Services (ml-service/)
All Python services are batch/scheduled jobs, not real-time:
- **daily/**: Daily batch processes (features, predictions, signals)
- **position_manager/**: Position state tracking
- **position_sizing/**: Kelly criterion and drawdown-based sizing
- **signals/**: ML-based signal generation
- **monitoring/**: System health monitoring

## OpenSpec Workflow

When planning new features, always follow the OpenSpec workflow:

1. **Before Planning**: Read `openspec/AGENTS.md` and `openspec/project.md`
2. **Check Existing Work**:
   - `openspec list` - Active changes
   - `openspec list --specs` - Existing specs
3. **Create Proposal**:
   - Directory: `openspec/changes/[change-id]/`
   - Files: `proposal.md`, `tasks.md`, `specs/[capability]/spec.md`
4. **Validate**: `openspec validate [change-id] --strict`
5. **Get Approval**: Wait for user approval before implementation
6. **Implement**: Follow `tasks.md` checklist
7. **Archive**: `openspec archive [change-id]` after deployment

## Common Patterns

### Repository Pattern
All database access goes through repositories in `internal/db/repository/`:
```go
// Example: Getting positions
posRepo := repository.NewPositionRepository(db)
positions, err := posRepo.GetOpenPositions(ctx)
```

### Service Layer
Business logic lives in `internal/service/`:
```go
// Example: Position service using repository
type PositionService struct {
    repo repository.PositionRepository
    // ... other dependencies
}
```

### Error Handling
Centralized error types in `internal/errors/errors.go`:
```go
import "github.com/nonobeam/golang-stock-trading/internal/errors"
// Use custom error types with context
```

### Logging
Structured logging via zerolog:
```go
import "github.com/nonobeam/golang-stock-trading/internal/logger"
logger.Info().Str("symbol", symbol).Msg("Processing signal")
```

## Testing Strategy

### Go Tests (`test/`)
- Unit tests for components (indicators, statistics, risk)
- Integration tests for position tracking
- Test file pattern: `*_test.go`

### Python Tests (`ml-service/tests/`)
- Integration tests for ML pipeline
- Position manager tests
- Signal capacity tests

## Configuration

### Environment Variables
Key env vars (check `.env` or config files):
- Database: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- Redis: `REDIS_ADDR`, `REDIS_PASSWORD` (mandatory)
- Broker: `DNSE_API_KEY`, `DNSE_USERNAME`, `DNSE_PASSWORD`
- Telegram: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`
- ML Service: `ML_SERVICE_URL`

### Configuration Files
- Go: `internal/config/config.go` (uses Viper)
- Python: `ml-service/config.py`

## Recent Major Changes

Check git log for details:
1. `64c5de6` - Fix Telegram bot deadlock, make Redis mandatory
2. `44e2162` - Implement /watch and /unwatch commands
3. `1d1ccde` - Add trading token expiration notifications
4. `7325643` - Implement /position command
5. `eee20e7` - Migrate from psycopg2 to psycopg3

## Completed Changes
- `implement-t2-settlement-risk/` - ✅ T+2 settlement risk management (Feb 2026)

## Pending Changes (openspec/changes/)
- `add-drawdown-risk-controls/` - Enhanced drawdown management
- `add-purchase-tracking/` - T+2 settlement tracking
- `enforce-stop-loss-preservation/` - Stop loss rules
- `integrate-enhanced-daily-signals/` - Improved signal generation
- `track-average-cost-capacity/` - Average cost tracking

## Dependencies

### Go (go.mod)
Key dependencies:
- `github.com/gorilla/mux` - HTTP routing
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram bot
- `github.com/rs/zerolog` - Logging
- `github.com/spf13/viper` - Configuration
- `google.golang.org/grpc` - gRPC support
- `github.com/gorilla/websocket` - WebSocket support

### Python (ml-service/)
Check `ml-service/requirements.txt` for ML dependencies

## Cross-Platform Notes

This codebase runs on Windows (current environment). Be aware:
- Use `Glob` tool instead of `ls` commands
- Path separators: Use `filepath.Join()` in Go
- Line endings: CRLF on Windows
- Shell: Git Bash available for Unix-style commands

## Quick Navigation Guide

**Need to find...**
- API endpoints? → `internal/api/`
- Database queries? → `internal/db/repository/`
- Business logic? → `internal/service/`
- ML pipeline? → `ml-service/daily/`
- Position tracking? → `internal/position/` (Go), `ml-service/position_manager/` (Python)
- Risk calculations? → `internal/risk/`
- T+2 settlement? → `internal/vn/settlement.go`, `docs/SETTLEMENT_TRACKING.md`
- Technical indicators? → `internal/analysis/indicators/`
- Telegram bot? → `internal/service/telegram/`
- Database schema? → `db/migrations/`
- Architecture specs? → `openspec/specs/`

## File Naming Conventions

- Go packages: lowercase, single word (avoid underscores)
- Go files: snake_case (e.g., `position_service.go`)
- Python files: snake_case (e.g., `kelly.py`)
- OpenSpec change IDs: kebab-case, verb-led (e.g., `add-feature-name`)
- Test files: `*_test.go` (Go), `test_*.py` (Python)

## Development Workflow

1. **Planning**: Use OpenSpec for features (see above)
2. **Database Changes**: Create migration in `db/migrations/`
3. **Go Code**: Follow repository → service → API layers
4. **Python Code**: Add to appropriate ml-service module
5. **Testing**: Add tests in `test/` or `ml-service/tests/`
6. **Commit**: Descriptive commit messages
7. **No push to main**: Create branches/PRs

---

**Last Updated**: 2026-02-06 (Auto-generated from codebase exploration)
