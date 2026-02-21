package vn_test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

func TestBoardLot_HOSE(t *testing.T) {
	tests := []struct {
		name     string
		shares   int
		expected int
	}{
		{"95 shares rounds to 90", 95, 90},
		{"100 shares stays 100", 100, 100},
		{"47 shares rounds to 40", 47, 40},
		{"5 shares rounds to 0", 5, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vn.RoundToHOSELot(tt.shares)
			if result != tt.expected {
				t.Errorf("RoundToHOSELot(%d) = %d, want %d", tt.shares, result, tt.expected)
			}
		})
	}
}

func TestBoardLot_HNX(t *testing.T) {
	tests := []struct {
		name     string
		shares   int
		expected int
	}{
		{"250 shares rounds to 200", 250, 200},
		{"300 shares stays 300", 300, 300},
		{"95 shares rounds to 0", 95, 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vn.RoundToHNXLot(tt.shares)
			if result != tt.expected {
				t.Errorf("RoundToHNXLot(%d) = %d, want %d", tt.shares, result, tt.expected)
			}
		})
	}
}

func TestCeilingDetector(t *testing.T) {
	detector := vn.NewCeilingDetector()
	
	tests := []struct {
		name          string
		entryPrice    float64
		currentPrice  float64
		dailyVolume   float64
		avgVolume     float64
		shouldExit    bool
	}{
		{
			name:         "at ceiling with volume spike",
			entryPrice:   30000,
			currentPrice: 32100, // 7% ceiling
			dailyVolume:  300000,
			avgVolume:    100000, // 3x spike
			shouldExit:   true,
		},
		{
			name:         "at ceiling without volume spike",
			entryPrice:   30000,
			currentPrice: 32100,
			dailyVolume:  200000,
			avgVolume:    100000, // Only 2x
			shouldExit:   false,
		},
		{
			name:         "below ceiling",
			entryPrice:   30000,
			currentPrice: 31000,
			dailyVolume:  300000,
			avgVolume:    100000,
			shouldExit:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := detector.DetectCeilingHit(tt.entryPrice, tt.currentPrice, tt.dailyVolume, tt.avgVolume)
			
			if tt.shouldExit {
				if !info.ShouldExitOnCeiling() {
					t.Error("Expected ceiling exit signal")
				}
			} else {
				if info.ShouldExitOnCeiling() {
					t.Error("Expected no ceiling exit signal")
				}
			}
		})
	}
}

func TestSOEClassifier(t *testing.T) {
	classifier := vn.NewSOEClassifier()
	
	tests := []struct {
		symbol string
		isSOE  bool
	}{
		{"VNM", true},   // Vinamilk
		{"GAS", true},   // PetroVietnam Gas
		{"VIC", false},  // Vingroup (private)
		{"HPG", false},  // Hoa Phat (private)
	}
	
	for _, tt := range tests {
		t.Run(tt.symbol, func(t *testing.T) {
			result := classifier.IsSOE(tt.symbol)
			if result != tt.isSOE {
				t.Errorf("IsSOE(%s) = %v, want %v", tt.symbol, result, tt.isSOE)
			}
			
			// Check allocation
			t1, t2, t3 := classifier.GetAllocation(tt.symbol)
			if tt.isSOE {
				if t1 != 30 || t2 != 40 || t3 != 30 {
					t.Errorf("SOE allocation should be 30/40/30, got %d/%d/%d", t1, t2, t3)
				}
			} else {
				if t1 != 30 || t2 != 30 || t3 != 40 {
					t.Errorf("Normal allocation should be 30/30/40, got %d/%d/%d", t1, t2, t3)
				}
			}
		})
	}
}
