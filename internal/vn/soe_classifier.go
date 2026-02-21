// Package vn provides Vietnamese market utilities.
package vn

// SOEClassifier classifies stocks as state-owned enterprises (SOEs).
type SOEClassifier struct {
	// In production, this would load from database or API
	soeList map[string]bool
}

// NewSOEClassifier creates a new SOE classifier.
func NewSOEClassifier() *SOEClassifier {
	// Common Vietnamese SOEs (this should be loaded from database in production)
	soeList := map[string]bool{
		"VNM": true,  // Vinamilk
		"GAS": true,  // PetroVietnam Gas
		"PLX": true,  // Petrolimex
		"POW": true,  // PetroVietnam Power
		"PVD": true,  // PetroVietnam Drilling
		"VCB": true,  // Vietcombank
		"CTG": true,  // VietinBank
		"BID": true,  // BIDV
		"EVN": true,  // Electricity Vietnam
		// Add more as needed
	}
	
	return &SOEClassifier{soeList: soeList}
}

// IsSOE checks if a stock is a state-owned enterprise.
func (c *SOEClassifier) IsSOE(symbol string) bool {
	return c.soeList[symbol]
}

// GetSOEAllocation returns adjusted allocation for SOEs.
// SOEs get 30%/40%/30% (vs normal 30%/30%/40%) to take more profit earlier.
func (c *SOEClassifier) GetSOEAllocation() (int, int, int) {
	return 30, 40, 30 // Target1: 30%, Target2: 40%, Target3: 30%
}

// GetNormalAllocation returns normal allocation for non-SOEs.
func (c *SOEClassifier) GetNormalAllocation() (int, int, int) {
	return 30, 30, 40 // Target1: 30%, Target2: 30%, Target3: 40%
}

// GetAllocation returns the appropriate allocation based on whether stock is SOE.
func (c *SOEClassifier) GetAllocation(symbol string) (int, int, int) {
	if c.IsSOE(symbol) {
		return c.GetSOEAllocation()
	}
	return c.GetNormalAllocation()
}
