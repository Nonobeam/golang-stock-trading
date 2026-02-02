package api

import (
	"encoding/json"

	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/pkg/httpclient"
)

type InfoAPI struct {
	client *httpclient.Client
}

func NewInfoAPI(client *httpclient.Client) *InfoAPI {
	return &InfoAPI{client: client}
}

type AccountInfo struct {
	AccountNo   string  `json:"accountNo"`
	AccountName string  `json:"accountName"`
	Balance     float64 `json:"balance"`
}

type StockPrice struct {
	Symbol     string  `json:"symbol"`
	Price      float64 `json:"price"`
	Change     float64 `json:"change"`
	ChangeRate float64 `json:"changeRate"`
	Volume     int64   `json:"volume"`
}

type MarketInfo struct {
	MarketId   string `json:"marketId"`
	MarketName string `json:"marketName"`
	Status     string `json:"status"`
}

func (a *InfoAPI) GetAccountInfo() (*AccountInfo, error) {
	logger.Debug().Msg("Fetching account info")

	respBody, err := a.client.Get("/account-service/account", nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp AccountInfo
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	return &resp, nil
}

func (a *InfoAPI) GetStockPrice(symbol string) (*StockPrice, error) {
	logger.Debug().Str("symbol", symbol).Msg("Fetching stock price")

	respBody, err := a.client.Get("/market-service/stock/"+symbol+"/price", nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp StockPrice
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	return &resp, nil
}

func (a *InfoAPI) GetMarketStatus(marketId string) (*MarketInfo, error) {
	logger.Debug().Str("marketId", marketId).Msg("Fetching market status")

	respBody, err := a.client.Get("/market-service/market/"+marketId+"/status", nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp MarketInfo
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	return &resp, nil
}

func (a *InfoAPI) GetPortfolio() ([]map[string]interface{}, error) {
	logger.Debug().Msg("Fetching portfolio")

	respBody, err := a.client.Get("/account-service/portfolio", nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	return resp, nil
}
