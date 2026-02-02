package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nonobeam/golang-stock-trading/internal/errors"
	"github.com/nonobeam/golang-stock-trading/internal/logger"
)

type Client struct {
	baseUrl    string
	httpClient *http.Client
	token      string
}

func New(baseUrl string, timeout time.Duration) *Client {
	return &Client{
		baseUrl: baseUrl,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) Get(path string, headers map[string]string) ([]byte, error) {
	return c.doRequest(http.MethodGet, path, nil, headers)
}

func (c *Client) Post(path string, body interface{}, headers map[string]string) ([]byte, error) {
	return c.doRequest(http.MethodPost, path, body, headers)
}

func (c *Client) doRequest(method, path string, body interface{}, headers map[string]string) ([]byte, error) {
	url := c.baseUrl + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, errors.ErrBadRequest)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternalServer)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	logger.Debug().
		Str("method", method).
		Str("url", url).
		Msg("Making HTTP request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("url", url).Msg("HTTP request failed")
		return nil, errors.Wrap(err, errors.ErrApiCallFailed)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidResponse)
	}

	if resp.StatusCode >= 400 {
		logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(respBody)).
			Msg("HTTP request returned error")
		return respBody, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
