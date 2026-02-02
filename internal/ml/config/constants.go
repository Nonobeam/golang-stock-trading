package config

import "time"

// Prediction Constants
const (
	// BuyThreshold is the minimum predicted p50 return (3%) to trigger a BUY recommendation.
	// Rationale: A 3% expected return over 5 days covers transaction costs and risk premium.
	BuyThreshold = 0.03

	// HighUncertainty is the threshold for the spread between p90 and p10 (20%).
	// If uncertainty > HighUncertainty, confidence is significantly penalized.
	HighUncertainty = 0.20

	// MediumUncertainty is the threshold for moderate uncertainty (10%).
	// If uncertainty > MediumUncertainty, confidence is moderately penalized.
	MediumUncertainty = 0.10
)

// Confidence Scores (Percentage 0-100)
const (
	// LowConfidence is used when model uncertainty is high (>20%).
	LowConfidence = 50.0

	// MediumConfidence is used when model uncertainty is moderate (>10%).
	MediumConfidence = 75.0

	// HighConfidence is used when model uncertainty is low (<=10%).
	HighConfidence = 90.0
)

// Timeouts
const (
	// PredictTimeout is the context timeout for the /predict command ML call.
	// Set to 5 seconds to ensure good user experience without hanging.
	PredictTimeout = 5 * time.Second

	// RecommendationMLTimeout is the context timeout for ML calls within recommendation service.
	// Set shorter (3 seconds) to prevent blocking the main recommendation flow.
	RecommendationMLTimeout = 3 * time.Second
)

// Validation
const (
	// MinPrice is the minimum allowed price to prevent negative values.
	MinPrice = 0.01

	// MaxConfidence is the maximum allowed confidence score (100%).
	MaxConfidence = 100.0

	// MinConfidence is the minimum allowed confidence score (0%).
	MinConfidence = 0.0
)
