package test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/scoring"
)

// TestCheckLiquidity tests liquidity filter requirements.
func TestCheckLiquidity(t *testing.T) {
	tests := []struct {
		name           string
		volume         float64
		turnover       float64
		zeroVolDays    int
		expectPass     bool
		expectIssueLen int
	}{
		{
			name:           "Pass all filters",
			volume:         1_000_000,
			turnover:       50_000_000_000,
			zeroVolDays:    0,
			expectPass:     true,
			expectIssueLen: 0,
		},
		{
			name:           "Fail volume filter",
			volume:         300_000,
			turnover:       50_000_000_000,
			zeroVolDays:    0,
			expectPass:     false,
			expectIssueLen: 1,
		},
		{
			name:           "Fail turnover filter",
			volume:         1_000_000,
			turnover:       1_000_000_000,
			zeroVolDays:    0,
			expectPass:     false,
			expectIssueLen: 1,
		},
		{
			name:           "Fail zero-volume days",
			volume:         1_000_000,
			turnover:       50_000_000_000,
			zeroVolDays:    1,
			expectPass:     false,
			expectIssueLen: 1,
		},
		{
			name:           "Fail all filters",
			volume:         100_000,
			turnover:       500_000_000,
			zeroVolDays:    2,
			expectPass:     false,
			expectIssueLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoring.CheckLiquidity(tt.volume, tt.turnover, tt.zeroVolDays)

			if result.Passes != tt.expectPass {
				t.Errorf("expected Passes=%v, got %v", tt.expectPass, result.Passes)
			}
			if len(result.Issues) != tt.expectIssueLen {
				t.Errorf("expected %d issues, got %d: %v", tt.expectIssueLen, len(result.Issues), result.Issues)
			}
		})
	}
}

// TestScoreTrendAlignment tests trend alignment scoring.
func TestScoreTrendAlignment(t *testing.T) {
	tests := []struct {
		name            string
		price           float64
		ema20           float64
		ema50           float64
		weeklyPrice     float64
		weeklySMA200    float64
		weeklyStructure string
		expectedScore   int
	}{
		{
			name:            "Perfect trend alignment",
			price:           52_000,
			ema20:           50_500,
			ema50:           48_200,
			weeklyPrice:     52_500,
			weeklySMA200:    45_000,
			weeklyStructure: "higher_highs_lows",
			expectedScore:   3,
		},
		{
			name:            "Below 20 EMA only",
			price:           49_000,
			ema20:           50_000,
			ema50:           48_000,
			weeklyPrice:     52_000,
			weeklySMA200:    45_000,
			weeklyStructure: "higher_highs_lows",
			expectedScore:   2,
		},
		{
			name:            "No trend alignment",
			price:           48_000,
			ema20:           50_000,
			ema50:           51_000,
			weeklyPrice:     48_500,
			weeklySMA200:    50_000,
			weeklyStructure: "lower_highs_lows",
			expectedScore:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoring.ScoreTrendAlignment(
				tt.price, tt.ema20, tt.ema50,
				tt.weeklyPrice, tt.weeklySMA200, tt.weeklyStructure,
			)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %d, got %d", tt.expectedScore, result.Score)
			}
			if result.MaxScore != 3 {
				t.Errorf("expected MaxScore=3, got %d", result.MaxScore)
			}
		})
	}
}

// TestScoreRiskReward tests risk/reward scoring.
func TestScoreRiskReward(t *testing.T) {
	tests := []struct {
		name          string
		entry         float64
		stop          float64
		target        float64
		atr           float64
		expectedScore int
	}{
		{
			name:          "Excellent R:R 2:1, stop under 7%",
			entry:         52_000,
			stop:          49_000,
			target:        58_000,
			atr:           2_500,
			expectedScore: 2,
		},
		{
			name:          "Poor R:R and wide stop",
			entry:         52_000,
			stop:          47_000,
			target:        54_500,
			atr:           2_000,
			expectedScore: 0,
		},
		{
			name:          "Good R:R at exactly 2:1",
			entry:         50_000,
			stop:          48_000,
			target:        54_000,
			atr:           1_500,
			expectedScore: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoring.ScoreRiskReward(tt.entry, tt.stop, tt.target, tt.atr)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %d, got %d; details: %v", tt.expectedScore, result.Score, result.Details)
			}
		})
	}
}

// TestScoreMomentum tests momentum scoring.
func TestScoreMomentum(t *testing.T) {
	tests := []struct {
		name             string
		rsi              float64
		macd             float64
		macdSignal       float64
		macdHistogram    float64
		prevHistogram    float64
		trendDir         string
		expectedScore    int
	}{
		{
			name:          "Strong momentum",
			rsi:           48,
			macd:          150,
			macdSignal:    120,
			macdHistogram: 30,
			prevHistogram: 20,
			trendDir:      "up",
			expectedScore: 2,
		},
		{
			name:          "Weak momentum",
			rsi:           28,
			macd:          -50,
			macdSignal:    -30,
			macdHistogram: -20,
			prevHistogram: -15,
			trendDir:      "up",
			expectedScore: 0,
		},
		{
			name:          "RSI over 70",
			rsi:           75,
			macd:          100,
			macdSignal:    90,
			macdHistogram: 10,
			prevHistogram: 5,
			trendDir:      "up",
			expectedScore: 1, // MACD positive, but RSI overextended
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoring.ScoreMomentum(
				tt.rsi, tt.macd, tt.macdSignal, tt.macdHistogram, tt.prevHistogram, tt.trendDir,
			)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %d, got %d; details: %v", tt.expectedScore, result.Score, result.Details)
			}
		})
	}
}

// TestFullTradeScoring tests the complete scoring system.
func TestFullTradeScoring(t *testing.T) {
	scorer := scoring.NewTradeScorer()

	t.Run("High quality trade", func(t *testing.T) {
		setup := scoring.TradeSetup{
			CurrentPrice:      52_000,
			EntryPrice:        52_000,
			StopLoss:          49_000,
			Target:            58_000,
			EMA20:             50_500,
			EMA50:             48_200,
			WeeklyPrice:       52_500,
			WeeklySMA200:      45_000,
			WeeklyStructure:   "higher_highs_lows",
			RSI:               48,
			MACD:              150,
			MACDSignal:        120,
			MACDHistogram:     30,
			PreviousHistogram: 20,
			ATR:               2_500,
			SupportLevel:      50_000,
			SupportType:       "previous_resistance",
			HasConsolidation:  true,
			ConsolidationBars: 8,
			VolumeConfirms:    true,
			VNIndexPrice:      1_250,
			VNIndexMA50:       1_200,
			SectorRS:          1.08,
			NewsSentiment:     "positive",
			AvgDailyVolume:    1_200_000,
			AvgDailyTurnover:  62_000_000_000,
			ZeroVolumeDays:    0,
		}

		result := scorer.Score(setup)

		if !result.ShouldTrade {
			t.Error("expected ShouldTrade=true for high quality setup")
		}
		if result.TotalScore < 9 {
			t.Errorf("expected score >= 9, got %d", result.TotalScore)
		}
		if result.RiskPercent < 1.0 {
			t.Errorf("expected risk >= 1.0%%, got %.1f%%", result.RiskPercent)
		}
	})

	t.Run("Failed liquidity", func(t *testing.T) {
		setup := scoring.TradeSetup{
			CurrentPrice:     52_000,
			AvgDailyVolume:   100_000, // Too low
			AvgDailyTurnover: 500_000_000,
			ZeroVolumeDays:   0,
		}

		result := scorer.Score(setup)

		if result.ShouldTrade {
			t.Error("expected ShouldTrade=false for failed liquidity")
		}
		if result.TotalScore != 0 {
			t.Errorf("expected score=0, got %d", result.TotalScore)
		}
	})

	t.Run("Low quality trade rejected", func(t *testing.T) {
		setup := scoring.TradeSetup{
			CurrentPrice:      48_000,
			EntryPrice:        48_000,
			StopLoss:          44_000, // Wide stop
			Target:            49_500, // Poor R:R
			EMA20:             50_000,
			EMA50:             51_000,
			WeeklyPrice:       48_500,
			WeeklySMA200:      50_000,
			WeeklyStructure:   "lower_highs_lows",
			RSI:               28,
			MACD:              -50,
			MACDSignal:        -30,
			MACDHistogram:     -20,
			PreviousHistogram: -15,
			ATR:               2_000,
			SupportType:       "none",
			HasConsolidation:  false,
			VolumeConfirms:    false,
			VNIndexPrice:      1_180,
			VNIndexMA50:       1_220,
			SectorRS:          0.95,
			NewsSentiment:     "negative",
			AvgDailyVolume:    1_000_000,
			AvgDailyTurnover:  50_000_000_000,
			ZeroVolumeDays:    0,
		}

		result := scorer.Score(setup)

		if result.ShouldTrade {
			t.Error("expected ShouldTrade=false for low quality setup")
		}
		if result.TotalScore >= 7 {
			t.Errorf("expected score < 7, got %d", result.TotalScore)
		}
	})
}

// TestVolumeConfirmation tests volume pattern analysis.
func TestVolumeConfirmation(t *testing.T) {
	t.Run("Volume confirms", func(t *testing.T) {
		volumes := []float64{1_000_000, 900_000, 800_000, 700_000, 600_000, 1_500_000}
		//                    pullback start=0, end=4, bounce=5
		result := scoring.CheckVolumeConfirmation(volumes, 0, 4)

		if !result.Confirms {
			t.Errorf("expected Confirms=true, got false; desc: %s", result.Description)
		}
		if !result.PullbackVolumeDeclined {
			t.Error("expected PullbackVolumeDeclined=true")
		}
		if !result.BounceVolumeIncreased {
			t.Error("expected BounceVolumeIncreased=true")
		}
	})

	t.Run("No volume decline", func(t *testing.T) {
		volumes := []float64{800_000, 900_000, 1_000_000, 1_100_000, 1_200_000, 1_400_000}
		result := scoring.CheckVolumeConfirmation(volumes, 0, 4)

		if result.Confirms {
			t.Error("expected Confirms=false when volume increases on pullback")
		}
	})

	t.Run("Insufficient data", func(t *testing.T) {
		volumes := []float64{1_000_000, 900_000}
		result := scoring.CheckVolumeConfirmation(volumes, 0, 2)

		if result.Confirms {
			t.Error("expected Confirms=false for insufficient data")
		}
	})
}
