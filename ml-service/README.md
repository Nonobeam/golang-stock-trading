# ML Prediction Service

Python-based machine learning service for stock price prediction using XGBoost quantile regression.

## Features

- XGBoost quantile regression (p10, p50, p90 predictions)
- Feature engineering from OHLC data (40+ technical indicators)
- gRPC interface for Go trading system
- Model training and persistence
- Daily automation for feature updates and predictions

## Installation

```bash
# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Generate gRPC code
python -m grpc_tools.protoc -I../proto \
    --python_out=generated \
    --grpc_python_out=generated \
    ../proto/ml_service.proto
```

## Configuration

Set environment variables:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=nonobeam
export DB_USER=your_user
export DB_PASSWORD=your_password
```

## Running the Service

```bash
python main.py
```

The service will start on `localhost:50051` and accept gRPC requests.

```
ml-service/
├── main.py                 # Service entry point
├── requirements.txt        # Python dependencies
├── pyproject.toml         # Project metadata
├── db/                    # Database connection
│   ├── connection.py
│   └── queries.py
├── features/              # Feature engineering
│   └── calculator.py
├── data/                  # Data loading
│   └── loader.py
├── models/                # Model training
│   └── trainer.py
├── inference/             # Prediction engine
│   └── predictor.py
├── server/                # gRPC server
│   └── grpc_server.py
├── position_manager/      # Position tracking (NEW)
│   ├── __init__.py
│   └── manager.py
├── signals/               # Trading signals (ENHANCED)
│   └── generator.py       # Position-aware signal generation
├── position_sizing/       # Position sizing (ENHANCED)
│   └── kelly.py           # Portfolio-aware sizing
├── daily/                 # Daily automation
│   ├── feature_updater.py
│   ├── prediction_generator.py
│   ├── outcome_recorder.py
│   ├── daily_signals.py   # Position monitoring workflow (NEW)
│   └── README.md          # Workflow documentation
├── scripts/               # Helper scripts (NEW)
│   ├── insert_vci_position.sql
│   └── execute_trade.py   # Trade execution helper
└── generated/             # Generated gRPC code
    ├── ml_service_pb2.py
    └── ml_service_pb2_grpc.py
```

## Position Management

The ML service now integrates with the `positions` table for portfolio-aware trading signals and risk management.

### PositionManager Module

CRUD operations for position tracking:

```python
from position_manager.manager import PositionManager
from db.connection import get_connection

conn = get_connection()
pm = PositionManager(conn)

# Query active position
position = pm.get_position('VCI', user_id=1)

# Get all positions
all_positions = pm.get_all_positions(user_id=1)

# Add new position
position_id = pm.add_position(
    user_id=1,
    ticker='VCI',
    shares=100,
    entry_price=36850,
    entry_date='2026-01-27',
    stop_loss=35100,
    target_1=39500,
    target_2=42000,
    target_3=45000
)

# Update quantity (weighted average cost)
pm.update_position_quantity(position_id, additional_shares=50, new_price=37200)

# Close position
pm.close_position(position_id, exit_price=39500, exit_date='2026-02-05', exit_reason='target_1_reached')
```

### Position-Aware Signal Generation

Signal generator now supports 6 signal types based on position context:

- **BUY_NEW**: Initiate new position (no current holding)
- **BUY_MORE**: Add to existing position
- **SELL**: Close entire position
- **SELL_PARTIAL**: Reduce position (target reached)
- **HOLD**: Maintain position
- **HOLD_NONE**: Do nothing (no position, no opportunity)

```python
from signals.generator import SignalGenerator

sg = SignalGenerator(user_id=1)

# Position-aware signal (requires db_connection)
signal, strength, reason = sg.generate_signal(
    ticker='VCI',
    predictions={5: {'p10': -0.015, 'p50': 0.032, 'p90': 0.079, 'confidence': 0.65}},
    current_price=37200,
    db_connection=conn,
    user_id=1
)

# Output: ('BUY_MORE', 0.32, 'Strong 5d outlook: 3.2% return - Add to position')
```

**Automatic risk management:**

- Stop-loss override: Triggers SELL if `current_price <= stop_loss`
- Target detection: Signals SELL_PARTIAL when T1/T2 reached, SELL when T3 reached
- Prevents BUY_MORE if price already at T1

### Daily Position Monitoring

Automate daily position monitoring and signal generation:

```bash
cd ml-service/daily
python daily_signals.py --date 2026-02-02
```

Generates comprehensive report with:

- Unrealized P&L for all positions
- Risk level checks (stop-loss distance, target distances)
- ML predictions and confidence
- Position-aware trading signals
- Alert summary (stop-loss triggers, targets reached)

See [daily/README.md](daily/README.md) for full documentation.

### Trade Execution Helper

Record manual trades with automatic position tracking:

```bash
# Buy shares
python scripts/execute_trade.py buy VCI 100 36850

# Sell partial
python scripts/execute_trade.py sell VCI 30 39500

# Close position
python scripts/execute_trade.py close VCI 35100 stop_loss_triggered
```

### Portfolio-Aware Position Sizing

Position sizer now considers current holdings:

```python
from position_sizing.kelly import PositionSizer

ps = PositionSizer(base_fraction=0.10, max_allocation=0.20)

# Portfolio-aware sizing
recommendation = ps.calculate_position_change(
    ticker='VCI',
    account_value=10000000,  # Total portfolio value
    current_price=37200,
    prediction_dict={'p10': -0.015, 'p50': 0.032, 'p90': 0.079},
    db_connection=conn,
    user_id=1
)

# Output:
# {
#   'action': 'BUY_MORE',
#   'current_shares': 100,
#   'recommended_shares': 268,
#   'delta_shares': 168,
#   'current_allocation': 0.037,
#   'recommended_allocation': 0.100,
#   'reason': 'Underweight by 6.3% - recommend buying 168 shares'
# }
```

## ML Integration Status

### ✅ Implemented Features

**ML Service (Python):**

- ✅ gRPC server with prediction endpoint (`Predict`)
- ✅ Model training via `TriggerTraining` RPC
- ✅ XGBoost quantile regression (p10, p50, p90)
- ✅ Feature backfilling from `daily_bars`
- ✅ Model persistence to `models/saved/`
- ✅ Database registration in `model_metadata` table

**Telegram Bot Integration:**

- ✅ `/train <symbol>` - Triggers ML training (backfill + train)
- ✅ Training status notifications
- ✅ Model version reporting

**Recent Fixes (2026-01-28):**

- ✅ Fixed XGBoost `n_estimators` warning by using `num_boost_round` parameter
- ✅ Fixed missing return statement in `train_all_quantiles()`
- ✅ Added `models/saved` directory creation in Dockerfile for proper permissions

### ❌ Missing Features (TODO)

**Telegram Bot Commands:**

- ❌ `/predict <symbol>` - Get ML predictions (p10/p50/p90) for a stock
- ❌ `/forecast <symbol>` - Show detailed forecast with confidence
- ❌ ML-based recommendations in signals/alerts

**Integration:**

- ❌ Recommendation service doesn't use ML predictions (uses hardcoded logic)
- ❌ Price alerts don't incorporate ML forecasts
- ❌ Signal detection doesn't use ML (only technical indicators)
- ❌ Position management doesn't use ML risk predictions

**ML Enhancements:**

- ❌ Multi-ticker batch predictions
- ❌ Model performance monitoring
- ❌ Automatic model retraining based on performance degradation
- ❌ Feature importance tracking

### 🎯 Quick Training

**Train a model locally:**

```bash
cd ml-service
source venv/bin/activate  # Windows: venv\Scripts\Activate.ps1
python train.py --ticker VCI
```

**Train via Telegram Bot:**

```
/train VCI
```

**Verify trained models:**

```sql
SELECT model_id, ticker, quantile, in_production, training_date
FROM model_metadata
WHERE ticker = 'VCI'
ORDER BY training_date DESC;
```

**Check saved models on disk:**

```bash
ls -la ml-service/models/saved/VCI/
```

## Development

```bash
# Run tests
pytest tests/

# Type checking
mypy ml-service/

# Linting
pylint ml-service/
```
