package vn_test

import (
	"testing"

	"github.com/nonobeam/golang-stock-trading/internal/vn"
)

func TestVNIndexMonitor_Evaluate(t *testing.T) {
	monitor := vn.NewVNIndexMonitor()

	tests := []struct {
		name          string
		prevClose     float64
		currentValue  float64
		wantDrop      float64 // approximate
		wantWarning   bool
		wantCritical  bool
	}{
		{
			name:         "small gain: no alert",
			prevClose:    1200,
			currentValue: 1212, // +1%
			wantWarning:  false,
			wantCritical: false,
		},
		{
			name:         "minor drop below warning threshold: no alert",
			prevClose:    1200,
			currentValue: 1192, // -0.67%
			wantWarning:  false,
			wantCritical: false,
		},
		{
			name:         "warning drop (exactly 1.5%): warning only",
			prevClose:    1200,
			currentValue: 1182, // -1.5%
			wantWarning:  true,
			wantCritical: false,
		},
		{
			name:         "warning drop (1.8%): warning only",
			prevClose:    1200,
			currentValue: 1178.4, // -1.8%
			wantWarning:  true,
			wantCritical: false,
		},
		{
			name:         "critical drop (exactly 3.0%): both warning and critical",
			prevClose:    1200,
			currentValue: 1164, // -3.0%
			wantWarning:  true,
			wantCritical: true,
		},
		{
			name:         "severe crash (5%): both warning and critical",
			prevClose:    1200,
			currentValue: 1140, // -5.0%
			wantWarning:  true,
			wantCritical: true,
		},
		{
			name:         "zero previous close: no alert (fail-safe)",
			prevClose:    0,
			currentValue: 1000,
			wantWarning:  false,
			wantCritical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := monitor.Evaluate(tt.prevClose, tt.currentValue)

			if result == nil {
				t.Fatal("Evaluate returned nil")
			}
			if result.IsWarning != tt.wantWarning {
				t.Errorf("IsWarning = %v, want %v (drop=%.2f%%)", result.IsWarning, tt.wantWarning, result.DropPercent)
			}
			if result.IsCritical != tt.wantCritical {
				t.Errorf("IsCritical = %v, want %v (drop=%.2f%%)", result.IsCritical, tt.wantCritical, result.DropPercent)
			}
			if result.Recommendation == "" {
				t.Error("Recommendation should not be empty")
			}
		})
	}
}

func TestVNIndexMonitor_AdjustedEmergencyThreshold(t *testing.T) {
	monitor := vn.NewVNIndexMonitor()

	tests := []struct {
		name          string
		original      float64
		isCritical    bool
		wantThreshold float64 // expected result
	}{
		{
			name:          "non-critical: threshold unchanged",
			original:      30.0,
			isCritical:    false,
			wantThreshold: 30.0,
		},
		{
			name:          "critical: threshold halved",
			original:      30.0,
			isCritical:    true,
			wantThreshold: 15.0,
		},
		{
			name:          "critical with very low original: clamped to 5%",
			original:      8.0,
			isCritical:    true,
			wantThreshold: 5.0, // 8/2=4, clamped to 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drop := &vn.VNIndexDropInfo{IsCritical: tt.isCritical}
			got := monitor.AdjustedEmergencyThreshold(tt.original, drop)
			if got != tt.wantThreshold {
				t.Errorf("AdjustedEmergencyThreshold(%v, critical=%v) = %.1f, want %.1f",
					tt.original, tt.isCritical, got, tt.wantThreshold)
			}
		})
	}
}
