// Package main implements the daily exit evaluator for graduated profit-taking.
//
// Run manually or via cron/systemd. The service:
//  1. Fetches all open positions from PostgreSQL
//  2. Gets current prices via DNSE REST API
//  3. Gets floor-hit probability via ML gRPC (P10 from PredictResponse)
//  4. Evaluates exit decisions (Target1/2/3 or Emergency)
//  5. Executes sell orders and updates the database
//
// Environment variables (from .env or system):
//
//	DNSE_API_BASE_URL, DNSE_USERNAME, DNSE_PASSWORD
//	DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
//	ML_GRPC_ADDR (default: localhost:50051)
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nonobeam/golang-stock-trading/internal/api"
	"github.com/nonobeam/golang-stock-trading/internal/config"
	"github.com/nonobeam/golang-stock-trading/internal/db"
	"github.com/nonobeam/golang-stock-trading/internal/position"
	"github.com/nonobeam/golang-stock-trading/internal/trading"
	"github.com/nonobeam/golang-stock-trading/internal/vn"
	mlpb "github.com/nonobeam/golang-stock-trading/proto/ml"
)

// ExitEvaluatorService orchestrates the daily exit evaluation workflow.
type ExitEvaluatorService struct {
	evaluator       *position.ExitEvaluator
	executor        *trading.ExitExecutor
	positionRepo    *ExitPositionRepository
	dnseClient      *api.DNSEClient
	mlClient        mlpb.MLPredictionServiceClient
	folMonitor      *vn.FOLMonitor
	ceilingDetector *vn.CeilingDetector
	vnIndexMonitor  *vn.VNIndexMonitor
}

func main() {
	// Load config from environment / .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config.Load: %v", err)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	dbCfg := &db.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            5432,
		User:            getEnv("DB_USER", "trading_user"),
		Password:        getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "trading"),
		Schema:          "stock-trading",
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxConnections:  10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	database, err := db.NewDatabase(dbCfg)
	if err != nil {
		log.Fatalf("db.NewDatabase: %v", err)
	}
	defer database.Close()

	// ── DNSE API ──────────────────────────────────────────────────────────────
	dnseAuth := api.NewDNSEAuthService(cfg.DnseApiBaseUrl, cfg.DnseUsername, cfg.DnsePassword)
	dnseClient := api.NewDNSEClient(cfg.DnseApiBaseUrl, dnseAuth)

	// ── ML gRPC ───────────────────────────────────────────────────────────────
	mlAddr := getEnv("ML_GRPC_ADDR", "localhost:50051")
	grpcConn, err := grpc.NewClient(mlAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient(%s): %v", mlAddr, err)
	}
	defer grpcConn.Close()
	mlClient := mlpb.NewMLPredictionServiceClient(grpcConn)

	// ── Service ───────────────────────────────────────────────────────────────
	service := &ExitEvaluatorService{
		evaluator:       position.NewExitEvaluator(position.DefaultExitEvaluatorConfig()),
		executor:        trading.NewExitExecutorWithClient(dnseClient),
		positionRepo:    NewExitPositionRepository(database.DB),
		dnseClient:      dnseClient,
		mlClient:        mlClient,
		folMonitor:      vn.NewFOLMonitor(),
		ceilingDetector: vn.NewCeilingDetector(),
		vnIndexMonitor:  vn.NewVNIndexMonitor(),
	}

	log.Println("=== Daily Exit Evaluator starting ===")
	ctx := context.Background()

	if err := service.EvaluateAllPositions(ctx); err != nil {
		log.Fatalf("EvaluateAllPositions: %v", err)
	}

	log.Println("=== Evaluation complete ===")
}

// EvaluateAllPositions runs the full evaluation loop across every open position.
func (s *ExitEvaluatorService) EvaluateAllPositions(ctx context.Context) error {
	positions, err := s.positionRepo.GetOpenPositionsWithExitTracking(ctx)
	if err != nil {
		return fmt.Errorf("GetOpenPositionsWithExitTracking: %w", err)
	}

	log.Printf("Evaluating %d open position(s)", len(positions))

	// ── VN-Index broad market check ─────────────────────────────────────────
	// Fetch last 2 days of VN-Index data to compute the daily change.
	// If the call fails we proceed with normal thresholds (fail-open).
	vnDrop := s.checkVNIndexDrop(ctx)
	if vnDrop != nil {
		log.Printf("[VNINDEX] %s", vnDrop.Recommendation)
		if vnDrop.IsCritical {
			// Rebuild evaluator with a tighter emergency threshold.
			cfg := position.DefaultExitEvaluatorConfig()
			cfg.EmergencyFloorThreshold = s.vnIndexMonitor.AdjustedEmergencyThreshold(
				cfg.EmergencyFloorThreshold, vnDrop)
			s.evaluator = position.NewExitEvaluator(cfg)
			log.Printf("[VNINDEX] Emergency threshold tightened to %.1f%% (was %.1f%%)",
				cfg.EmergencyFloorThreshold, position.DefaultExitEvaluatorConfig().EmergencyFloorThreshold)
		}
	}
	// ────────────────────────────────────────────────────────────────────────

	exitCount, errCount := 0, 0
	today := time.Now()

	for _, pos := range positions {
		if err := s.evaluateOne(ctx, pos, today); err != nil {
			log.Printf("ERROR %s: %v", pos.Symbol, err)
			errCount++
			continue
		}
		exitCount++
	}

	log.Printf("Done – %d exit(s) executed, %d error(s)", exitCount, errCount)
	return nil
}

// checkVNIndexDrop fetches the last 2 days of VN-Index data and evaluates the drop.
// Returns nil on error so the caller can proceed with normal thresholds.
func (s *ExitEvaluatorService) checkVNIndexDrop(_ context.Context) *vn.VNIndexDropInfo {
	now := time.Now()
	bars, err := s.dnseClient.GetVNIndexDaily(now.AddDate(0, 0, -5), now) // 5-day window for weekends
	if err != nil || len(bars) < 2 {
		if err != nil {
			log.Printf("[VNINDEX] fetch failed: %v – using normal thresholds", err)
		}
		return nil
	}
	prevClose := bars[len(bars)-2].Value
	currentValue := bars[len(bars)-1].Value
	return s.vnIndexMonitor.Evaluate(prevClose, currentValue)
}

// evaluateOne evaluates a single position and executes an exit if warranted.
func (s *ExitEvaluatorService) evaluateOne(ctx context.Context, pos position.DBPosition, today time.Time) error {
	// 1. Current price + floor/ceiling prices from the exchange
	symbolInfo, err := s.dnseClient.GetSymbolInfo(pos.Symbol)
	if err != nil {
		return fmt.Errorf("GetSymbolInfo: %w", err)
	}
	currentPrice := symbolInfo.LastPrice

	// 2. Consecutive floor counter persistence
	//    Detect floor hit, update in-memory counter, persist to DB.
	floorResult := position.CheckFloorHit(&pos, currentPrice, symbolInfo.Floor, today)
	if floorResult.IsFloorHit {
		if err := s.positionRepo.UpdateFloorCounter(ctx, &pos); err != nil {
			// Non-fatal – log and continue so the exit decision still runs.
			log.Printf("[FLOOR] UpdateFloorCounter(%s) failed: %v", pos.Symbol, err)
		}
		if floorResult.WasReset {
			log.Printf("[FLOOR] %s hit floor (day 1, counter reset after gap)", pos.Symbol)
		} else {
			log.Printf("[FLOOR] %s hit floor (consecutive day %d)", pos.Symbol, floorResult.FloorHitDays)
		}
	}

	// 3. Floor-hit probability from ML service (P10 = downside tail risk %)
	floorHitProb := s.getFloorHitProbability(ctx, pos.Symbol, currentPrice)

	// 4. Escalate if FOL emergency
	folRestriction, _ := s.folMonitor.CheckFOLRestriction(pos.Symbol)
	if folRestriction != nil && folRestriction.Level == "EMERGENCY" {
		log.Printf("[FOL] EMERGENCY for %s – forcing emergency exit", pos.Symbol)
		floorHitProb = 100.0
	}

	// 5. Log ceiling hits (informational; also handled by evaluator)
	ceilingInfo := s.ceilingDetector.DetectCeilingHit(
		pos.EntryPrice, currentPrice,
		float64(symbolInfo.Volume),
		float64(symbolInfo.Volume)*0.8, // Approximate avg volume
	)
	if ceilingInfo.ShouldExitOnCeiling() {
		log.Printf("[CEILING] Lock detected for %s (price +%.1f%%)", pos.Symbol,
			(currentPrice-pos.EntryPrice)/pos.EntryPrice*100)
	}

	// 6. Determine exit decision
	//    NOTE: pos.FloorHitDays was mutated by CheckFloorHit above, so the
	//    emergency check (3+ consecutive floor days) sees the updated value.
	decision := s.evaluator.EvaluatePosition(&pos, currentPrice, floorHitProb)
	if decision == nil {
		return nil // No exit warranted
	}

	log.Printf("[EXIT] %s: signal=%s target=%d pct=%d%% shares=%d @ %.2f",
		pos.Symbol, decision.SignalType, decision.TargetLevel,
		decision.ExitPercentage, decision.Shares, currentPrice)

	// 7. Execute the sell
	result, err := s.executor.ExecuteExit(ctx, decision, pos.Symbol, currentPrice)
	if err != nil {
		return fmt.Errorf("ExecuteExit: %w", err)
	}

	log.Printf("[FILLED] orderID=%s filled=%d/requested=%d avgPrice=%.2f",
		result.OrderID, result.SharesFilled, result.SharesOrdered, result.AveragePrice)

	// 8. Persist exit tracking
	if err := s.positionRepo.UpdateExitTracking(ctx, &pos, decision, result); err != nil {
		return fmt.Errorf("UpdateExitTracking: %w", err)
	}

	return nil
}

// getFloorHitProbability calls the ML service for floor-hit probability.
// Uses P10 (10th-percentile 1-day return) as a downside risk proxy.
// Returns 10.0 (low risk default) if the ML call fails.
func (s *ExitEvaluatorService) getFloorHitProbability(ctx context.Context, symbol string, _ float64) float64 {
	mlCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := s.mlClient.Predict(mlCtx, &mlpb.PredictRequest{
		Ticker: symbol,
		Date:   time.Now().Format("2006-01-02"),
	})
	if err != nil {
		log.Printf("[ML] Predict(%s) failed: %v – using default 10%%", symbol, err)
		return 10.0
	}
	if !resp.Success {
		log.Printf("[ML] Predict(%s) returned error: %s – using default 10%%", symbol, resp.ErrorMessage)
		return 10.0
	}

	// P10 is the 10th-percentile predicted return.
	// Convert to an absolute "floor risk" percentage: if P10 > -7 (ceiling hit),
	// downside risk is low; if P10 is very negative, risk is high.
	// We map: P10 = -7% → prob ≈ 80%, P10 = 0% → prob ≈ 20%.
	p10 := resp.P10
	prob := 20.0 - p10*10 // rough linear mapping
	if prob < 0 {
		prob = 0
	}
	if prob > 100 {
		prob = 100
	}
	return prob
}

// ──────────────────────────────────────────────────────────────────────────────
// ExitPositionRepository — database layer
// ──────────────────────────────────────────────────────────────────────────────

// ExitPositionRepository handles database reads/writes for the exit workflow.
type ExitPositionRepository struct {
	db *sql.DB
}

func NewExitPositionRepository(db *sql.DB) *ExitPositionRepository {
	return &ExitPositionRepository{db: db}
}

// GetOpenPositionsWithExitTracking returns all open positions with exit-tracking fields.
func (r *ExitPositionRepository) GetOpenPositionsWithExitTracking(ctx context.Context) ([]position.DBPosition, error) {
	const query = `
		SELECT
			id,
			symbol,
			entry_price,
			quantity                                    AS initial_shares,
			quantity                                    AS current_shares,
			target_1,
			target_2,
			target_3,
			COALESCE(target1_filled,      FALSE)        AS target1_filled,
			COALESCE(target2_filled,      FALSE)        AS target2_filled,
			COALESCE(trailing_stop_active, FALSE)       AS trailing_stop_active,
			target1_exit_price,
			target2_exit_price,
			target1_exit_date,
			target2_exit_date,
			ceiling_hit_date,
			COALESCE(ceiling_lock_days,   0)            AS ceiling_lock_days,
			COALESCE(floor_hit_days,      0)            AS floor_hit_days,
			last_floor_date
		FROM positions
		WHERE is_closed = FALSE
		ORDER BY entry_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []position.DBPosition
	for rows.Next() {
		var p position.DBPosition
		if err := rows.Scan(
			&p.PositionID, &p.Symbol, &p.EntryPrice, &p.InitialShares, &p.CurrentShares,
			&p.Target1, &p.Target2, &p.Target3,
			&p.Target1Filled, &p.Target2Filled, &p.TrailingStopActive,
			&p.Target1ExitPrice, &p.Target2ExitPrice,
			&p.Target1ExitDate, &p.Target2ExitDate,
			&p.CeilingHitDate, &p.CeilingLockDays, &p.FloorHitDays, &p.LastFloorDate,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateFloorCounter persists the updated consecutive floor counter for a position.
// It is called whenever a stock hits its daily -7% price floor.
func (r *ExitPositionRepository) UpdateFloorCounter(
	ctx context.Context,
	pos *position.DBPosition,
) error {
	const query = `
		UPDATE positions SET
			floor_hit_days  = $2,
			last_floor_date = $3,
			updated_at      = NOW()
		WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, pos.PositionID, pos.FloorHitDays, pos.LastFloorDate)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("position %s not found", pos.PositionID)
	}
	return nil
}

// UpdateExitTracking persists an exit result: updates quantity, fills, and closes when full.
func (r *ExitPositionRepository) UpdateExitTracking(
	ctx context.Context,
	pos *position.DBPosition,
	decision *position.ExitDecision,
	result *trading.ExitResult,
) error {
	var query string
	switch decision.TargetLevel {
	case 1:
		query = `
			UPDATE positions SET
				target1_filled     = TRUE,
				target1_exit_price = $2,
				target1_exit_date  = NOW(),
				quantity           = GREATEST(quantity - $3, 0),
				updated_at         = NOW()
			WHERE id = $1`
	case 2:
		query = `
			UPDATE positions SET
				target2_filled     = TRUE,
				target2_exit_price = $2,
				target2_exit_date  = NOW(),
				quantity           = GREATEST(quantity - $3, 0),
				updated_at         = NOW()
			WHERE id = $1`
	case 3:
		query = `
			UPDATE positions SET
				trailing_stop_active = TRUE,
				is_closed            = TRUE,
				exit_price           = $2,
				exit_date            = NOW(),
				exit_reason          = 'Target 3 trailing stop',
				quantity             = 0,
				pnl                  = ($2 - entry_price) * $3,
				pnl_percent          = (($2 - entry_price) / NULLIF(entry_price, 0)) * 100,
				updated_at           = NOW()
			WHERE id = $1`
	case 0:
		query = `
			UPDATE positions SET
				is_closed   = TRUE,
				exit_price  = $2,
				exit_date   = NOW(),
				exit_reason = 'Emergency exit',
				quantity    = 0,
				pnl         = ($2 - entry_price) * $3,
				pnl_percent = (($2 - entry_price) / NULLIF(entry_price, 0)) * 100,
				updated_at  = NOW()
			WHERE id = $1`
	default:
		return fmt.Errorf("unknown target level %d", decision.TargetLevel)
	}

	res, err := r.db.ExecContext(ctx, query, pos.PositionID, result.AveragePrice, result.SharesFilled)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("position %s not found", pos.PositionID)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
