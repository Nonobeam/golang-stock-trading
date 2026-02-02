package scoring

import "fmt"

// VolumeConfirmResult holds the result of volume pattern analysis.
type VolumeConfirmResult struct {
	Confirms                  bool
	Description               string
	PullbackVolumeDeclined    bool
	BounceVolumeIncreased     bool
	BounceVolumeChangePercent float64
}

// CheckVolumeConfirmation analyzes volume patterns to confirm bullish setup.
// For a bullish setup, we want:
// - Declining volume during pullback (selling dries up)
// - Increasing volume on bounce (buyers stepping in)
func CheckVolumeConfirmation(volumes []float64, pullbackStartIdx, pullbackEndIdx int) VolumeConfirmResult {
	result := VolumeConfirmResult{
		Confirms:    false,
		Description: "",
	}

	// Need at least 2 pullback bars + 1 bounce bar
	if len(volumes) < pullbackEndIdx+2 {
		result.Description = "Insufficient data"
		return result
	}

	if pullbackStartIdx >= pullbackEndIdx {
		result.Description = "Invalid pullback range"
		return result
	}

	// Calculate average volume during pullback
	pullbackVolumes := volumes[pullbackStartIdx : pullbackEndIdx+1]
	avgPullbackVolume := 0.0
	for _, v := range pullbackVolumes {
		avgPullbackVolume += v
	}
	avgPullbackVolume /= float64(len(pullbackVolumes))

	// Get volume on bounce day (day after pullback)
	bounceVolume := volumes[pullbackEndIdx+1]

	// Check if volume decreased during pullback
	// Compare early pullback vs late pullback
	mid := len(pullbackVolumes) / 2
	if mid == 0 {
		mid = 1
	}

	earlyPullbackSum := 0.0
	for i := 0; i < mid; i++ {
		earlyPullbackSum += pullbackVolumes[i]
	}
	earlyPullbackAvg := earlyPullbackSum / float64(mid)

	latePullbackSum := 0.0
	latePullbackCount := len(pullbackVolumes) - mid
	for i := mid; i < len(pullbackVolumes); i++ {
		latePullbackSum += pullbackVolumes[i]
	}
	latePullbackAvg := latePullbackSum / float64(latePullbackCount)

	volumeDeclined := latePullbackAvg < earlyPullbackAvg
	result.PullbackVolumeDeclined = volumeDeclined

	// Check if bounce volume increased (20%+ above pullback average)
	bounceChangePercent := 0.0
	if avgPullbackVolume > 0 {
		bounceChangePercent = ((bounceVolume / avgPullbackVolume) - 1) * 100
	}
	volumeIncreased := bounceVolume > avgPullbackVolume*1.2 // 20% threshold
	result.BounceVolumeIncreased = volumeIncreased
	result.BounceVolumeChangePercent = bounceChangePercent

	// Confirmation requires both conditions
	result.Confirms = volumeDeclined && volumeIncreased

	if volumeDeclined {
		result.Description = fmt.Sprintf("Pullback vol declined, bounce vol +%.0f%%", bounceChangePercent)
	} else {
		result.Description = fmt.Sprintf("No vol decline on pullback, bounce vol +%.0f%%", bounceChangePercent)
	}

	return result
}
