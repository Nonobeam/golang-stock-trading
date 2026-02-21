// Package vn provides Vietnamese market utilities.
package vn

import "fmt"

// VNIndexMonitor evaluates the VN-Index daily change and signals broad market stress.
//
// During a WARNING (>= 1.5% drop) callers should avoid opening new positions.
// During a CRITICAL (>= 3.0% drop) callers should lower the emergency floor
// threshold to exit positions more defensively.
type VNIndexMonitor struct {
	// WarningThreshold is the negative drop % that triggers a WARNING (default -1.5).
	WarningThreshold float64
	// CriticalThreshold is the negative drop % that triggers CRITICAL (default -3.0).
	CriticalThreshold float64
}

// VNIndexDropInfo holds the result of a single VN-Index evaluation.
type VNIndexDropInfo struct {
	// DropPercent is the index change in %, e.g. -2.5 means the index is down 2.5%.
	DropPercent float64
	// IsWarning is true when the index dropped >= WarningThreshold (default 1.5%).
	IsWarning bool
	// IsCritical is true when the index dropped >= CriticalThreshold (default 3.0%).
	IsCritical bool
	// Recommendation is a human-readable action hint.
	Recommendation string
}

// DefaultWarningThreshold – index down 1.5% or more.
const DefaultWarningThreshold = 1.5

// DefaultCriticalThreshold – index down 3.0% or more (broad market panic).
const DefaultCriticalThreshold = 3.0

// NewVNIndexMonitor creates a VNIndexMonitor with default thresholds.
func NewVNIndexMonitor() *VNIndexMonitor {
	return &VNIndexMonitor{
		WarningThreshold:  DefaultWarningThreshold,
		CriticalThreshold: DefaultCriticalThreshold,
	}
}

// Evaluate computes the VN-Index drop from previousClose to currentValue and
// returns a VNIndexDropInfo describing the severity.
//
// previousClose must be > 0; if it is 0 the result will have IsCritical=false
// and a neutral recommendation (prevents divide-by-zero in simulation mode).
func (m *VNIndexMonitor) Evaluate(previousClose, currentValue float64) *VNIndexDropInfo {
	if previousClose <= 0 {
		return &VNIndexDropInfo{
			DropPercent:    0,
			IsWarning:      false,
			IsCritical:     false,
			Recommendation: "VN-Index previous close unavailable – assuming normal conditions",
		}
	}

	dropPercent := (currentValue - previousClose) / previousClose * 100 // negative = drop

	info := &VNIndexDropInfo{DropPercent: dropPercent}

	switch {
	case -dropPercent >= m.CriticalThreshold:
		info.IsCritical = true
		info.IsWarning = true
		info.Recommendation = fmt.Sprintf(
			"CRITICAL: VN-Index down %.2f%% – tighten emergency threshold, defend all positions",
			-dropPercent)
	case -dropPercent >= m.WarningThreshold:
		info.IsWarning = true
		info.Recommendation = fmt.Sprintf(
			"WARNING: VN-Index down %.2f%% – hold positions, avoid new buys",
			-dropPercent)
	default:
		info.Recommendation = fmt.Sprintf(
			"VN-Index change %.2f%% – normal conditions",
			dropPercent)
	}

	return info
}

// AdjustedEmergencyThreshold returns a lower emergency floor threshold to use
// when the market is in a CRITICAL drop, making exits more aggressive.
//
// The original threshold is halved (e.g. 30% → 15%) so that higher-probability
// floor hits are treated as emergencies even without 3 consecutive floor days.
func (m *VNIndexMonitor) AdjustedEmergencyThreshold(originalThreshold float64, drop *VNIndexDropInfo) float64 {
	if drop.IsCritical {
		adjusted := originalThreshold / 2
		if adjusted < 5 {
			adjusted = 5 // floor at 5% to prevent always-exit
		}
		return adjusted
	}
	return originalThreshold
}
