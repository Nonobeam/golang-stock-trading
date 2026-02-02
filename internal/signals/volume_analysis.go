package signals

import "fmt"

// An analyzePullbackVolume analyzes volume patterns during pullbacks and bounces.
// Confirms accumulation: volume declines during pullback, spikes on bounce day.
func AnalyzePullbackVolume(volumes, closes []float64, daysBack int) *VolumePatternResult {
	if len(volumes) < daysBack || daysBack < 3 {
		return &VolumePatternResult{
			Confirms:    false,
			Description: "Insufficient volume data",
		}
	}
	
	recentVolumes := volumes[len(volumes)-daysBack:]
	
	// Split into early and late periods
	midPoint := len(recentVolumes) / 2
	earlyVolumes := recentVolumes[:midPoint]
	lateVolumes := recentVolumes[midPoint : len(recentVolumes)-1] // Exclude current day
	
	// Calculate averages
	earlyAvg := average(earlyVolumes)
	lateAvg := average(lateVolumes)
	currentVolume := volumes[len(volumes)-1]
	
	// Check if volume declined during pullback (10%+ decline)
	volumeDeclined := lateAvg < earlyAvg*0.9
	
	// Check if bounce day has volume spike (20%+ above recent average)
	avgRecent := average(recentVolumes[:len(recentVolumes)-1])
	bounceSpike := currentVolume > avgRecent*1.2
	
	// Calculate percentage changes for description
	declinePercent := 0.0
	if earlyAvg > 0 {
		declinePercent = ((lateAvg / earlyAvg) - 1.0) * 100
	}
	
	spikePercent := 0.0
	if avgRecent > 0 {
		spikePercent = ((currentVolume / avgRecent) - 1.0) * 100
	}
	
	description := fmt.Sprintf("Pullback vol: %.0f%%, Bounce vol: +%.0f%%", declinePercent, spikePercent)
	
	return &VolumePatternResult{
		Confirms:                     volumeDeclined && bounceSpike,
		VolumeDeclinedDuringPullback: volumeDeclined,
		BounceVolumeSpike:            bounceSpike,
		Description:                  description,
		EarlyAvgVolume:               earlyAvg,
		LateAvgVolume:                lateAvg,
		CurrentVolume:                currentVolume,
	}
}

// CalculateVolumePercentile ranks current volume in historical distribution.
// Returns percentile (0-100) where current volume stands.
func CalculateVolumePercentile(currentVol float64, historicalVols []float64) float64 {
	if len(historicalVols) == 0 {
		return 50.0 // Neutral if no data
	}
	
	// Count how many historical volumes are below current volume
	belowCount := 0
	for _, vol := range historicalVols {
		if vol < currentVol {
			belowCount++
		}
	}
	
	percentile := (float64(belowCount) / float64(len(historicalVols))) * 100.0
	return percentile
}

// average calculates the mean of a float64 slice.
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
