# Implementation Tasks

## 1. Database Schema and Migrations

- [ ] 1.1 Create migration `000014_add_settlement_tracking.up.sql`
  - Add columns to positions table: settlement_status, purchase_date, settlement_date, can_sell_date, locked_capital, liquid_capital
  - Create settlement_status enum type: LOCKED_T0, LOCKED_T1, LOCKED_T2, LIQUID
  - Add exchange column to positions table (HOSE, HNX, UPCOM)

- [ ] 1.2 Create position_settlement_tracking table
  - Fields: tracking_id, position_id, check_date, settlement_status, days_until_liquid, locked_value, locked_risk, risk_classification
  - risk_classification enum: HIGH_RISK_LOCKED, MODERATE_RISK_NEAR_LIQUID, LOW_RISK_LIQUID
  - Indexes on position_id and check_date

- [ ] 1.3 Add locked_risk_threshold to user_config table
  - Field: locked_risk_threshold DECIMAL(3,2) DEFAULT 0.10
  - Add constraint: CHECK (locked_risk_threshold BETWEEN 0.05 AND 0.20)

- [ ] 1.4 Create theoretical_stop_breaches table
  - Fields: breach_id, position_id, breach_date, stop_price, actual_price, settlement_status, days_until_executable
  - Track stop losses that were triggered but not executable

- [ ] 1.5 Create down migration `000014_add_settlement_tracking.down.sql`
  - Drop theoretical_stop_breaches table
  - Drop position_settlement_tracking table
  - Remove added columns from positions and user_config
  - Drop settlement_status enum type

## 2. Go Backend - Settlement Status Management

- [ ] 2.1 Extend internal/vn/settlement.go
  - Add SettlementStatus enum constants
  - Add CalculateSettlementStatus(purchaseDate, currentDate) function
  - Add GetDaysUntilLiquid(purchaseDate, currentDate) function
  - Add GetCanSellDate(purchaseDate) function using existing T+2 logic

- [ ] 2.2 Update internal/db/repository/types.go
  - Add settlement fields to Position struct
  - Add Exchange field (string: "HOSE", "HNX", "UPCOM")
  - Add helper methods: IsLocked(), IsLiquid(), GetLockedRisk()

- [ ] 2.3 Update internal/db/repository/position_repository.go
  - Add CreatePositionWithSettlement method
  - Add UpdateSettlementStatus method
  - Add GetPositionsBySettlementStatus query
  - Add GetTotalLockedRisk query
  - Add GetLockedPositions query

- [ ] 2.4 Update internal/service/position/position_service.go
  - Add CalculateLockedRisk method with exchange-aware logic
  - Add CheckLockedRiskBudget method
  - Add UpdateDailySettlementStatuses method (for cron job)
  - Add RecordTheoreticalStopBreach method

## 3. Go Backend - Risk Calculation

- [ ] 3.1 Update internal/risk/position_sizing.go
  - Add CalculateLockedRisk function with exchange parameter
  - HOSE: shares * price * 0.20
  - HNX: shares * price * 0.30
  - UPCOM: shares * price * 0.40
  - Add CalculateLiquidRisk function: shares * (entry - stop)

- [ ] 3.2 Create internal/risk/locked_risk.go
  - Add GetTotalLockedRisk function (sums all locked positions)
  - Add GetAvailableLockedRiskBudget function
  - Add CanAffordLockedRisk function (checks budget before purchase)
  - Add GetRiskComposition function (returns locked vs liquid breakdown)

- [ ] 3.3 Update internal/risk/portfolio_manager.go
  - Integrate locked risk calculation into portfolio risk checks
  - Add CheckLockedRiskThreshold method
  - Update risk dashboard to show locked vs liquid risk

## 4. Go Backend - Signal Generation Integration

- [ ] 4.1 Update internal/signals/types.go
  - Add LockedRiskRejection rejection reason
  - Add EntryDayRestriction rejection reason
  - Add settlement info to signal metadata

- [ ] 4.2 Create internal/signals/settlement_validator.go
  - Add ValidateLockedRiskBudget function
  - Add ApplyEntryDayRestrictions function
  - Add GetPositionSizeMultiplier function (1.0 Mon-Wed, 0.5 Thu-Fri)

## 5. Go Backend - Stop Loss Execution

- [ ] 5.1 Create internal/service/position/stop_loss_validator.go
  - Add CanExecuteStopLoss function (checks settlement status)
  - Add RecordTheoreticalBreach function
  - Add GetExecutableStops query
  - Add GetTheoreticalStops query

- [ ] 5.2 Update existing stop loss execution logic
  - Add settlement status check before execution
  - Log theoretical breaches for locked positions
  - Send notifications for non-executable stop breaches

## 6. Go Backend - Daily Settlement Update Job

- [ ] 6.1 Create internal/jobs/settlement_update.go
  - Add RunDailySettlementUpdate function
  - Query all active positions
  - Calculate current settlement status based on purchase_date
  - Update settlement_status in database
  - Record daily snapshot in position_settlement_tracking

- [ ] 6.2 Create cmd/jobs/settlement_cron.go
  - Add CLI command for manual execution: `go run cmd/jobs/settlement_cron.go`
  - Add scheduling logic (intended to run at 16:30 daily)
  - Add error handling and logging

- [ ] 6.3 Add settlement update to existing job scheduler
  - If using cron: add schedule for 16:30 weekdays
  - If using in-app scheduler: integrate into existing job system

## 7. Python ML Service - Locked Risk Integration

- [ ] 7.1 Create ml-service/position_sizing/locked_risk.py
  - Add calculate_locked_risk(shares, price, exchange) function
  - Add get_exchange_risk_multiplier(exchange) function
  - Add check_locked_risk_budget(ticker, shares, price, account_value) function

- [ ] 7.2 Update ml-service/signals/generator.py
  - Import locked_risk module
  - Add locked risk check before BUY_NEW and BUY_MORE signals
  - Add entry day restriction logic
  - Return rejection reason if locked risk budget exceeded

- [ ] 7.3 Update ml-service/position_manager/manager.py
  - Add get_total_locked_risk query
  - Add get_locked_risk_budget query
  - Add get_settlement_status query for position

- [ ] 7.4 Create ml-service/daily/update_settlement_status.py
  - Add script to update settlement statuses (Python equivalent of Go job)
  - Can be used as backup/validation of Go settlement updates

## 8. Reporting and Monitoring

- [ ] 8.1 Create internal/service/telegram/settlement_reports.go
  - Add FormatSettlementStatusMessage function
  - Add FormatLockedRiskReport function
  - Add FormatPositionLiquidNotification function

- [ ] 8.2 Update internal/service/telegram/bot_service_positions.go
  - Enhance /position command to show settlement status
  - Show days until liquid
  - Show locked vs liquid capital

- [ ] 8.3 Create internal/service/telegram/settlement_alerts.go
  - Add alert when position transitions to LIQUID
  - Add alert when locked risk exceeds 80% of threshold
  - Add alert when stop loss breached but not executable

- [ ] 8.4 Create ml-service/monitoring/settlement_monitor.py
  - Add daily report of settlement status distribution
  - Add locked risk utilization metrics
  - Add theoretical stop breach tracking

## 9. Testing

- [ ] 9.1 Create internal/vn/settlement_test.go
  - Test settlement date calculation with weekends
  - Test settlement date calculation with holidays
  - Test settlement status transitions
  - Test GetDaysUntilLiquid with various dates

- [ ] 9.2 Create internal/risk/locked_risk_test.go
  - Test locked risk calculation for each exchange
  - Test locked risk budget enforcement
  - Test risk composition calculations

- [ ] 9.3 Create ml-service/tests/test_locked_risk_integration.py
  - Test signal rejection due to locked risk budget
  - Test entry day restrictions
  - Test locked risk with multiple positions

- [ ] 9.4 Create integration test for settlement workflow
  - Test full lifecycle: purchase -> locked -> transition -> liquid
  - Test stop loss execution blocked during locked period
  - Test stop loss execution allowed after liquid

## 10. Data Migration and Backfill

- [ ] 10.1 Create migration script for existing positions
  - Set all existing positions to LIQUID (assume already settled)
  - Set settlement_date to entry_date + 3 days
  - Set can_sell_date to settlement_date
  - Infer exchange from ticker symbol pattern

- [ ] 10.2 Test migration on staging database
  - Verify all existing positions updated correctly
  - Verify no data loss
  - Verify risk calculations work with backfilled data

## 11. Documentation

- [ ] 11.1 Update API documentation
  - Document new position fields
  - Document settlement status enum values
  - Document locked risk calculation formulas

- [ ] 11.2 Create settlement risk user guide
  - Explain T+2 settlement period
  - Explain locked vs liquid positions
  - Explain locked risk budget
  - Explain entry day restrictions

- [ ] 11.3 Update configuration documentation
  - Document locked_risk_threshold setting
  - Document recommended values
  - Document how to override entry day restrictions

## 12. Deployment and Validation

- [ ] 12.1 Deploy to staging environment
  - Run database migrations
  - Deploy updated Go backend
  - Deploy updated Python ML service
  - Configure settlement cron job

- [ ] 12.2 Validation period (1 week)
  - Monitor settlement status transitions
  - Verify locked risk calculations
  - Verify stop loss execution logic
  - Collect metrics on signal rejections

- [ ] 12.3 Adjust thresholds based on validation
  - Tune locked_risk_threshold if too conservative
  - Adjust entry day restrictions if needed
  - Fine-tune exchange risk multipliers

- [ ] 12.4 Deploy to production
  - Run database migrations during maintenance window
  - Deploy backend and ML service updates
  - Monitor for 48 hours
  - Enable settlement cron job
