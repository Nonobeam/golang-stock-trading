package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	AppName  string
	AppEnv   string
	LogLevel string

	// DNSE API
	DnseApiBaseUrl string
	DnseWsUrl      string
	DnseUsername   string
	DnsePassword   string

	// Telegram
	TelegramBotToken string
	TelegramChatID   int64

	// SMTP
	SmtpHost     string
	SmtpPort     int
	SmtpUser     string
	SmtpPassword string
	SmtpFrom     string

	// Trading Configuration
	Trading TradingConfig
	
	// API Server
	ServerPort  string
	JWTSecret   string
	CORSOrigins []string

	// Redis
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
	RedisOTPKey   string
	RedisOTPTTL   int
	RedisJWTKey   string
}

// TradingConfig holds trading-related parameters that can be overridden via env.
type TradingConfig struct {
	// Vietnam Market
	VNDailyLimitPercent float64 // Daily price limit (default: 0.07 = 7%)
	VNLotSize           int     // Standard lot size (default: 100)
	VNSettlementDays    int     // T+N settlement (default: 2)

	// Risk Management
	MaxStopPercent    float64 // Maximum stop loss distance (default: 0.07 = 7%)
	MaxPositionPercent float64 // Maximum position size as % of capital (default: 20%)
	DefaultRiskPct    float64 // Default risk per trade (default: 0.01 = 1%)
	GapRiskFactor     float64 // Overnight gap risk multiplier (default: 1.5)
	DefaultATRMult    float64 // Default ATR multiplier for stops (default: 2.0)
	MinStopPercent    float64 // Minimum stop distance % (default: 0.01 = 1%)
	MinStopDistance   float64 // Minimum stop distance in VND (default: 500)
	SwingLookback     int     // Periods to look back for swing low (default: 20)
	PreemptiveEnabled bool    // Enable pre-emptive exit alerts (default: true)

	// Indicators
	DefaultRSIPeriod int     // Default RSI period (default: 14)
	DefaultATRPeriod int     // Default ATR period (default: 14)
	DefaultSMAPeriod int     // Default SMA period (default: 20)
	DefaultBBPeriod  int     // Default Bollinger period (default: 20)
	DefaultBBStdDev  float64 // Default Bollinger std dev (default: 2.0)
}

var cfg *Config

func Load() (*Config, error) {
	_ = godotenv.Load()

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("APP_NAME", "golang-stock-trading")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("LOG_LEVEL", "debug")
	viper.SetDefault("DNSE_API_BASE_URL", "https://api.dnse.com.vn")
	viper.SetDefault("DNSE_WS_URL", "wss://pricestream.dnse.com.vn")
	viper.SetDefault("SMTP_PORT", 587)

	// Trading config defaults
	viper.SetDefault("TRADING_VN_DAILY_LIMIT_PERCENT", 0.07)
	viper.SetDefault("TRADING_VN_LOT_SIZE", 100)
	viper.SetDefault("TRADING_VN_SETTLEMENT_DAYS", 2)
	viper.SetDefault("TRADING_MAX_STOP_PERCENT", 0.07)
	viper.SetDefault("TRADING_MAX_POSITION_PERCENT", 20.0)
	viper.SetDefault("TRADING_DEFAULT_RISK_PCT", 0.01)
	viper.SetDefault("TRADING_GAP_RISK_FACTOR", 1.5)
	viper.SetDefault("TRADING_DEFAULT_ATR_MULT", 2.0)
	viper.SetDefault("TRADING_MIN_STOP_PERCENT", 0.01)
	viper.SetDefault("TRADING_MIN_STOP_DISTANCE", 500.0)
	viper.SetDefault("TRADING_SWING_LOOKBACK", 20)
	viper.SetDefault("TRADING_PREEMPTIVE_ENABLED", true)
	viper.SetDefault("TRADING_DEFAULT_RSI_PERIOD", 14)
	viper.SetDefault("TRADING_DEFAULT_ATR_PERIOD", 14)
	viper.SetDefault("TRADING_DEFAULT_SMA_PERIOD", 20)
	viper.SetDefault("TRADING_DEFAULT_BB_PERIOD", 20)
	viper.SetDefault("TRADING_DEFAULT_BB_STDDEV", 2.0)

	// API Server defaults
	viper.SetDefault("APP_PORT", ":8080")
	viper.SetDefault("JWT_SECRET", "")
	viper.SetDefault("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_OTP_TTL", 300) // 5 minutes

	cfg = &Config{
		AppName:  viper.GetString("APP_NAME"),
		AppEnv:   viper.GetString("APP_ENV"),
		LogLevel: viper.GetString("LOG_LEVEL"),

		DnseApiBaseUrl: viper.GetString("DNSE_API_BASE_URL"),
		DnseWsUrl:      viper.GetString("DNSE_WS_URL"),
		DnseUsername:   viper.GetString("DNSE_USERNAME"),
		DnsePassword:   viper.GetString("DNSE_PASSWORD"),

		TelegramBotToken: viper.GetString("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   viper.GetInt64("TELEGRAM_CHAT_ID"),

		SmtpHost:     viper.GetString("SMTP_HOST"),
		SmtpPort:     viper.GetInt("SMTP_PORT"),
		SmtpUser:     viper.GetString("SMTP_USER"),
		SmtpPassword: viper.GetString("SMTP_PASSWORD"),
		SmtpFrom:     viper.GetString("SMTP_FROM"),

		Trading: TradingConfig{
			VNDailyLimitPercent:  viper.GetFloat64("TRADING_VN_DAILY_LIMIT_PERCENT"),
			VNLotSize:            viper.GetInt("TRADING_VN_LOT_SIZE"),
			VNSettlementDays:     viper.GetInt("TRADING_VN_SETTLEMENT_DAYS"),
			MaxStopPercent:       viper.GetFloat64("TRADING_MAX_STOP_PERCENT"),
			MaxPositionPercent:   viper.GetFloat64("TRADING_MAX_POSITION_PERCENT"),
			DefaultRiskPct:       viper.GetFloat64("TRADING_DEFAULT_RISK_PCT"),
			GapRiskFactor:        viper.GetFloat64("TRADING_GAP_RISK_FACTOR"),
			DefaultATRMult:       viper.GetFloat64("TRADING_DEFAULT_ATR_MULT"),
			MinStopPercent:       viper.GetFloat64("TRADING_MIN_STOP_PERCENT"),
			MinStopDistance:      viper.GetFloat64("TRADING_MIN_STOP_DISTANCE"),
			SwingLookback:        viper.GetInt("TRADING_SWING_LOOKBACK"),
			PreemptiveEnabled:    viper.GetBool("TRADING_PREEMPTIVE_ENABLED"),
			DefaultRSIPeriod:     viper.GetInt("TRADING_DEFAULT_RSI_PERIOD"),
			DefaultATRPeriod:     viper.GetInt("TRADING_DEFAULT_ATR_PERIOD"),
			DefaultSMAPeriod:     viper.GetInt("TRADING_DEFAULT_SMA_PERIOD"),
			DefaultBBPeriod:      viper.GetInt("TRADING_DEFAULT_BB_PERIOD"),
			DefaultBBStdDev:      viper.GetFloat64("TRADING_DEFAULT_BB_STDDEV"),
		},

		ServerPort:  viper.GetString("APP_PORT"),
		JWTSecret:   viper.GetString("JWT_SECRET"),
		CORSOrigins: strings.Split(viper.GetString("CORS_ORIGINS"), ","),

		// Redis
		RedisHost:     viper.GetString("REDIS_HOST"),
		RedisPort:     viper.GetInt("REDIS_PORT"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),
		RedisDB:       viper.GetInt("REDIS_DB"),
		RedisOTPTTL:   viper.GetInt("REDIS_OTP_TTL"),
	}

	// Validate JWT secret: require explicit configuration in production
	if cfg.JWTSecret == "" {
		if cfg.AppEnv == "production" {
			return nil, fmt.Errorf("JWT_SECRET must be set in production environment")
		}
		// Generate a random secret for development to avoid using a known default
		cfg.JWTSecret = generateRandomSecret()
	}

	return cfg, nil
}

func Get() *Config {
	if cfg == nil {
		cfg, _ = Load()
	}
	return cfg
}

// generateRandomSecret creates a cryptographically random hex string for development use.
func generateRandomSecret() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback should never happen with crypto/rand
		return "fallback-dev-secret-" + hex.EncodeToString(bytes[:8])
	}
	return hex.EncodeToString(bytes)
}
