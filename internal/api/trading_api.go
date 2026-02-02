package api

import (
	"encoding/json"

	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
	"github.com/nonobeam/golang-stock-trading/pkg/httpclient"
)

type TradingAPI struct {
	client       *httpclient.Client
	tradingToken string
}

func NewTradingAPI(client *httpclient.Client, tradingToken string) *TradingAPI {
	return &TradingAPI{
		client:       client,
		tradingToken: tradingToken,
	}
}

func (a *TradingAPI) SetTradingToken(token string) {
	a.tradingToken = token
}

type OrderRequest struct {
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`      // BUY or SELL
	OrderType string  `json:"orderType"` // LO, MP, ATO, ATC
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderResponse struct {
	OrderId     string `json:"orderId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CreatedTime string `json:"createdTime"`
}

type TransferRequest struct {
	FromAccount string  `json:"fromAccount"`
	ToAccount   string  `json:"toAccount"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type TransferResponse struct {
	TransactionId string `json:"transactionId"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

func (a *TradingAPI) getHeaders() map[string]string {
	return map[string]string{
		"tradingToken": a.tradingToken,
	}
}

func (a *TradingAPI) PlaceOrder(req *OrderRequest) (*OrderResponse, error) {
	if a.tradingToken == "" {
		return nil, errors.ErrUnauthorized
	}

	logger.Info().
		Str("symbol", req.Symbol).
		Str("side", req.Side).
		Int("quantity", req.Quantity).
		Float64("price", req.Price).
		Msg("Placing order")

	respBody, err := a.client.Post("/order-service/order/place", req, a.getHeaders())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to place order")
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp OrderResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	logger.Info().Str("orderId", resp.OrderId).Msg("Order placed successfully")
	return &resp, nil
}

func (a *TradingAPI) CancelOrder(orderId string) (*OrderResponse, error) {
	if a.tradingToken == "" {
		return nil, errors.ErrUnauthorized
	}

	logger.Info().Str("orderId", orderId).Msg("Cancelling order")

	respBody, err := a.client.Post("/order-service/order/"+orderId+"/cancel", nil, a.getHeaders())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cancel order")
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp OrderResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	return &resp, nil
}

func (a *TradingAPI) Transfer(req *TransferRequest) (*TransferResponse, error) {
	if a.tradingToken == "" {
		return nil, errors.ErrUnauthorized
	}

	logger.Info().
		Str("from", req.FromAccount).
		Str("to", req.ToAccount).
		Float64("amount", req.Amount).
		Msg("Initiating transfer")

	respBody, err := a.client.Post("/account-service/transfer", req, a.getHeaders())
	if err != nil {
		logger.Error().Err(err).Msg("Failed to transfer")
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}

	var resp TransferResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	logger.Info().Str("transactionId", resp.TransactionId).Msg("Transfer successful")
	return &resp, nil
}
