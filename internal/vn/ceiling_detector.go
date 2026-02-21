// Package vn provides Vietnamese market utilities.
package vn

// CeilingDetector detects when stocks hit the +7% ceiling (HOSE/HNX).
type CeilingDetector struct{}

// NewCeilingDetector creates a new ceiling detector.
func NewCeilingDetector() *CeilingDetector {
	return &CeilingDetector{}
}

// DetectCeilingHit checks if price is at or near the +7% ceiling.
func (d *CeilingDetector) DetectCeilingHit(entryPrice, currentPrice, dailyVolume, avgVolume float64) *CeilingHitInfo {
	ceiling := entryPrice * 1.07 // +7% ceiling for HOSE
	
	// Check if within 0.1% of ceiling
	percentToCeiling := ((ceiling - currentPrice) / ceiling) * 100
	
	if percentToCeiling <= 0.1 { // Within 0.1% of ceiling
		volumeMultiple := dailyVolume / avgVolume
		
		return &CeilingHitInfo{
			IsCeilingHit:    true,
			CeilingPrice:    ceiling,
			CurrentPrice:    currentPrice,
			PercentToCeiling: percentToCeiling,
			VolumeMultiple:  volumeMultiple,
			HighVolumeSpike: volumeMultiple >= 3.0, // 3x volume spike
		}
	}
	
	return &CeilingHitInfo{IsCeilingHit: false}
}

// CeilingHitInfo contains ceiling hit detection results.
type CeilingHitInfo struct {
	IsCeilingHit     bool
	CeilingPrice     float64
	CurrentPrice     float64
	PercentToCeiling float64
	VolumeMultiple   float64
	HighVolumeSpike  bool
}

// ShouldExitOnCeiling determines if position should exit due to ceiling lock.
func (i *CeilingHitInfo) ShouldExitOnCeiling() bool {
	return i.IsCeilingHit && i.HighVolumeSpike
}
