package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// DNSEAuthService manages JWT authentication for DNSE API.
type DNSEAuthService struct {
	baseURL  string
	email    string
	password string
	token    string
	tokenExp time.Time
	mu       sync.RWMutex
	client   *http.Client
}

// NewDNSEAuthService creates a new authentication service.
func NewDNSEAuthService(baseURL, email, password string) *DNSEAuthService {
	return &DNSEAuthService{
		baseURL:  baseURL,
		email:    email,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoginRequest is the payload for DNSE login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the response from DNSE login.
type LoginResponse struct {
	Token     string    `json:"token"`
	Roles     []string  `json:"roles"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Login authenticates and retrieves JWT token.
func (s *DNSEAuthService) Login() error {
	payload := LoginRequest{
		Username: s.email,
		Password: s.password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest("POST", s.baseURL+"/user-service/api/auth", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	s.mu.Lock()
	s.token = loginResp.Token
	if loginResp.ExpiresAt.IsZero() {
		s.tokenExp = time.Now().Add(24 * time.Hour)
	} else {
		s.tokenExp = loginResp.ExpiresAt
	}
	s.mu.Unlock()

	return nil
}

// GetToken returns the current JWT token, refreshing if needed.
func (s *DNSEAuthService) GetToken() (string, error) {
	s.mu.RLock()
	token := s.token
	exp := s.tokenExp
	s.mu.RUnlock()

	// Refresh if token expires in less than 5 minutes
	if time.Now().Add(5 * time.Minute).After(exp) {
		if err := s.Login(); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
		s.mu.RLock()
		token = s.token
		s.mu.RUnlock()
	}

	return token, nil
}

func (s *DNSEAuthService) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token != "" && time.Now().Before(s.tokenExp)
}
