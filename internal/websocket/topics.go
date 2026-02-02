package websocket

import "fmt"

const (
	Resolution1Min  = "1"
	Resolution1Hour = "1H"
	Resolution1Day  = "1D"
	ResolutionWeek  = "W"
)

const (
	MarketHoSE       = "STO"
	MarketHNX        = "STX"
	MarketBond       = "BDX"
	MarketDerivative = "DVX"
	MarketUpCoM      = "UPX"
	MarketCorpBond   = "HCX"
)

const (
	ProductHoSEStock   = "STO"
	ProductGovBond     = "BDX"
	ProductIndexFuture = "FIO"
	ProductBondFuture  = "FBX"
	ProductHNXStock    = "STX"
	ProductUpCoM       = "UPX"
	ProductCorpBond    = "HCX"
)

const (
	IndexVN30    = "VN30"
	IndexVNINDEX = "VNINDEX"
	IndexHNX30   = "HNX30"
	IndexHNX     = "HNX"
	IndexUPCOM   = "UPCOM"
)

func TopicStockInfo(symbol string) string {
	return fmt.Sprintf("quotes/krx/mdds/stockinfo/v1/roundlot/symbol/%s", symbol)
}

func TopicTopPrice(symbol string) string {
	return fmt.Sprintf("quotes/krx/mdds/topprice/v1/roundlot/symbol/%s", symbol)
}

func TopicBoardEvent(market, productGrpId string) string {
	return fmt.Sprintf("quotes/krx/mdds/boardevent/v1/roundlot/market/%s/product/%s", market, productGrpId)
}

func TopicMarketIndex(indexName string) string {
	return fmt.Sprintf("quotes/krx/mdds/index/%s", indexName)
}

func TopicStockOHLC(resolution, symbol string) string {
	return fmt.Sprintf("quotes/krx/mdds/v2/ohlc/stock/%s/%s", resolution, symbol)
}

func TopicDerivativeOHLC(resolution, symbol string) string {
	return fmt.Sprintf("quotes/krx/mdds/v2/ohlc/derivative/%s/%s", resolution, symbol)
}

func TopicIndexOHLC(resolution, indexName string) string {
	return fmt.Sprintf("quotes/krx/mdds/v2/ohlc/index/%s/%s", resolution, indexName)
}

func TopicTick(symbol string) string {
	return fmt.Sprintf("quotes/krx/mdds/tick/v1/roundlot/symbol/%s", symbol)
}
